//go:build integration

// Package integration proves the real stack integrates end to end:
// POST /api/location -> row in location_history -> connected WebSocket
// client receives LOCATION_UPDATE. Every other test in this repo mocks its
// repository/service dependencies; this is the one place that exercises the
// real Postgres (TimescaleDB + PostGIS) and Redis containers, the real
// migration path, and the real hub.
//
// Gated behind the "integration" build tag because it requires a reachable
// Docker daemon and is not part of the routine `go test ./...` loop. Run it
// with:
//
//	go test -tags=integration ./test/... -v -count=1 -timeout 120s
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"transit-backend/internal/cache"
	"transit-backend/internal/config"
	"transit-backend/internal/database"
	"transit-backend/internal/handlers"
	"transit-backend/internal/hub"
	"transit-backend/internal/middleware"
	"transit-backend/internal/models"
	"transit-backend/internal/services"
	"transit-backend/migrations"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	goredis "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Image tags must match infra/docker-compose.yml exactly -- this test proves
// the stack that actually runs, not a substitute.
const (
	postgresImage = "timescale/timescaledb-ha:pg15-latest"
	redisImage    = "redis:7-alpine"

	dbName = "transit_poc"
	dbUser = "transit_user"
	dbPass = "transit_password"
)

func TestIntegration_LocationIngestToWebSocketBroadcast(t *testing.T) {
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer setupCancel()

	// --- 1. Start Postgres and Redis containers concurrently. ---
	var (
		pgContainer    *tcpostgres.PostgresContainer
		redisContainer *tcredis.RedisContainer
		pgErr, rdErr   error
		wg             sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		pgContainer, pgErr = startPostgres(setupCtx)
	}()
	go func() {
		defer wg.Done()
		redisContainer, rdErr = startRedis(setupCtx)
	}()
	wg.Wait()

	// Register cleanup immediately after creation, before any Fatal that
	// could unwind -- containers must terminate even on partial failure.
	if pgContainer != nil {
		t.Cleanup(func() {
			if err := pgContainer.Terminate(context.Background()); err != nil {
				t.Logf("terminate postgres container: %v", err)
			}
		})
	}
	if redisContainer != nil {
		t.Cleanup(func() {
			if err := redisContainer.Terminate(context.Background()); err != nil {
				t.Logf("terminate redis container: %v", err)
			}
		})
	}
	if pgErr != nil {
		t.Fatalf("start postgres container (is the Docker daemon running?): %v", pgErr)
	}
	if rdErr != nil {
		t.Fatalf("start redis container (is the Docker daemon running?): %v", rdErr)
	}

	// --- 2. Read mapped endpoints -- never assume host ports. ---
	pgConnStr, err := pgContainer.ConnectionString(setupCtx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get postgres connection string: %v", err)
	}
	pgHost, pgPort := mustParseHostPort(t, pgConnStr, "postgres")

	redisConnStr, err := redisContainer.ConnectionString(setupCtx)
	if err != nil {
		t.Fatalf("get redis connection string: %v", err)
	}
	redisHost, redisPort := mustParseRedisHostPort(t, redisConnStr)

	// --- 3. Build config pointing at the containers (not config.LoadConfig,
	// which reads .env / process env). ---
	cfg := &config.Config{
		DBHost:     pgHost,
		DBPort:     pgPort,
		DBName:     dbName,
		DBUser:     dbUser,
		DBPassword: dbPass,
		DBMaxConns: 10,
		DBMinConns: 2,

		RedisHost:     redisHost,
		RedisPort:     redisPort,
		RedisPassword: "",
		RedisPoolSize: 10,

		ServerHost: "127.0.0.1",
		ServerPort: "0",

		H3Resolution: 9,

		Env:      "test",
		LogLevel: "debug",
	}

	// --- 4. database.New + RunMigrations -- main.go steps 2 and 2a, the
	// real migration path (schema_migrations ledger, sorted apply order). ---
	dbPool, err := database.New(cfg)
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	t.Cleanup(dbPool.Close)

	if err := database.RunMigrations(setupCtx, dbPool, migrations.Files); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// --- 5. cache.New -- main.go step 3, proves Redis is reachable. ---
	redisClient, err := cache.New(cfg)
	if err != nil {
		t.Fatalf("connect to redis: %v", err)
	}
	t.Cleanup(func() {
		if err := redisClient.Close(); err != nil {
			t.Logf("close redis client: %v", err)
		}
	})

	// --- 6. Seed this run's own fixtures, isolated by a unique device id. ---
	runID := fmt.Sprintf("%d", time.Now().UnixNano())
	deviceID := "itest-dev-" + runID
	routeID := "itest-route-" + runID
	tripID := "itest-trip-" + runID

	if _, err := dbPool.Exec(setupCtx,
		`INSERT INTO routes (id, name, description) VALUES ($1, 'itest route', '') ON CONFLICT DO NOTHING`,
		routeID); err != nil {
		t.Fatalf("seed route: %v", err)
	}
	if _, err := dbPool.Exec(setupCtx,
		`INSERT INTO devices (id, name, status) VALUES ($1, 'itest device', 'ACTIVE') ON CONFLICT DO NOTHING`,
		deviceID); err != nil {
		t.Fatalf("seed device: %v", err)
	}
	if _, err := dbPool.Exec(setupCtx,
		`INSERT INTO trips (id, route_id, device_id, started_at, status) VALUES ($1, $2, $3, NOW(), 'IN_PROGRESS') ON CONFLICT DO NOTHING`,
		tripID, routeID, deviceID); err != nil {
		t.Fatalf("seed trip: %v", err)
	}

	// --- 7. Wire the app exactly as main.go steps 4-9 (minus e.Static). ---
	locRepo := database.NewLocationRepositoryWithResolution(dbPool, cfg.H3Resolution)
	deviceRouteRepo := database.NewDeviceRouteRepository(dbPool)
	deviceCache := cache.NewDeviceCache(redisClient)

	wsHub := hub.New()
	hubCtx, hubCancel := context.WithCancel(context.Background())
	go wsHub.Run(hubCtx)

	ingestService := services.NewIngestService(locRepo, deviceRouteRepo, deviceCache, wsHub)

	locHandler := handlers.NewLocationHandler(ingestService)
	wsHandler := handlers.NewWebSocketHandler(wsHub)

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = middleware.HTTPErrorHandler
	e.Use(middleware.Recovery)
	e.Use(middleware.CORS())
	e.Use(middleware.Logging)

	api := e.Group("/api")
	api.POST("/location", locHandler.IngestLocation)
	e.GET("/ws", wsHandler.HandleWS)

	// --- 8. Start the server on a random port. ---
	srv := httptest.NewServer(e)

	// --- Teardown, registered now, LIFO: server -> hub -> WS client ->
	// DB/Redis clients -> containers (containers already registered above).
	// Order matters for the D28 drain: shutting the hub down while the
	// server-side WritePump is still alive (i.e. before closing the client
	// connection) means drainAndClose sees an empty buffer and takes its
	// fast path instead of the 250ms timeout. ---
	var wsConn *websocket.Conn
	t.Cleanup(func() {
		srv.Close()
		hubCancel()
		wsHub.Shutdown()
		if wsConn != nil {
			wsConn.Close()
		}
	})

	// --- 9. Dial the WebSocket. ---
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	wsConn, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}

	// --- 10. Wait deterministically for hub registration. Upgrade writes
	// the 101 response before Register<-client runs, so a successful dial
	// does not guarantee the hub knows about the client yet. ---
	registerCtx, registerCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer registerCancel()
	for wsHub.ClientCount() != 1 {
		select {
		case <-registerCtx.Done():
			t.Fatalf("timed out waiting for hub to register the WS client, got %d", wsHub.ClientCount())
		default:
			// yield to the hub goroutine
		}
	}

	// --- 11. Bound the read without a sleep. ---
	if err := wsConn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	// --- 12. POST the payload. speed:0 is deliberate -- the DEFECT-5
	// omitempty regression trigger: a stopped device is the most common
	// real state and the exact case omitempty used to erase. ---
	ts := time.Now().Unix()
	payload := models.Location{
		DeviceID:  deviceID,
		Latitude:  28.4595,
		Longitude: 77.0266,
		Speed:     0,
		Accuracy:  5.5,
		Timestamp: ts,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	resp, err := http.Post(srv.URL+"/api/location", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/location: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /api/location: status = %d, want 200", resp.StatusCode)
	}

	var httpResp struct {
		Status  string `json:"status"`
		HexRes9 string `json:"hex_res9"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&httpResp); err != nil {
		t.Fatalf("decode POST response: %v", err)
	}
	if httpResp.Status != "ok" {
		t.Errorf("response status = %q, want %q", httpResp.Status, "ok")
	}
	if httpResp.HexRes9 == "" {
		t.Fatal("response hex_res9 is empty, want a computed H3 cell")
	}
	wantHex := httpResp.HexRes9

	// --- 13. Assert the DB row, scoped to this device+timestamp (the seed
	// migration leaves 11 pre-existing rows -- never assert a bare count). ---
	var (
		gotLat, gotLng, gotSpeed, gotAccuracy float64
		gotHex                                string
		gotGeomNotNull                        bool
	)
	err = dbPool.QueryRow(setupCtx,
		`SELECT latitude, longitude, speed, accuracy, hex_res9, (geom IS NOT NULL)
		 FROM location_history WHERE device_id = $1 AND time = $2`,
		deviceID, time.Unix(ts, 0),
	).Scan(&gotLat, &gotLng, &gotSpeed, &gotAccuracy, &gotHex, &gotGeomNotNull)
	if err != nil {
		t.Fatalf("query location_history row: %v", err)
	}

	const epsilon = 1e-6
	if diff := gotLat - payload.Latitude; diff > epsilon || diff < -epsilon {
		t.Errorf("db latitude = %v, want %v", gotLat, payload.Latitude)
	}
	if diff := gotLng - payload.Longitude; diff > epsilon || diff < -epsilon {
		t.Errorf("db longitude = %v, want %v", gotLng, payload.Longitude)
	}
	if gotSpeed != 0 {
		t.Errorf("db speed = %v, want 0", gotSpeed)
	}
	if gotAccuracy != payload.Accuracy {
		t.Errorf("db accuracy = %v, want %v", gotAccuracy, payload.Accuracy)
	}
	if gotHex != wantHex {
		t.Errorf("db hex_res9 = %q, want %q (the value the API returned)", gotHex, wantHex)
	}
	if !gotGeomNotNull {
		t.Error("db geom IS NULL, want NOT NULL -- the trg_location_geom trigger should have populated it")
	}

	// --- 14. Assert the WebSocket message. Unmarshal into a raw map FIRST
	// to check key presence -- the DEFECT-5 assertion. A struct unmarshal
	// alone cannot distinguish "key absent" from "key present and zero",
	// which is exactly the bug omitempty caused. ---
	_, rawMsg, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("read websocket message: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawMsg, &raw); err != nil {
		t.Fatalf("unmarshal websocket message as map: %v", err)
	}

	wantKeys := []string{
		"type", "device_id", "route_id", "latitude", "longitude",
		"speed", "accuracy", "h3_hex", "timestamp",
	}
	for _, k := range wantKeys {
		if _, ok := raw[k]; !ok {
			t.Errorf("websocket message missing key %q (DEFECT-5 regression: omitempty dropping a present-but-zero field)", k)
		}
	}
	if _, ok := raw["heading"]; ok {
		t.Error(`websocket message has a "heading" key, want none -- CLAUDE.md Sec 7.2 / DEFECT-2: heading does not exist on this envelope`)
	}

	var msg hub.Message
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		t.Fatalf("unmarshal websocket message as hub.Message: %v", err)
	}

	if msg.Type != hub.MsgTypeLocationUpdate {
		t.Errorf("ws type = %q, want %q", msg.Type, hub.MsgTypeLocationUpdate)
	}
	if msg.DeviceID != deviceID {
		t.Errorf("ws device_id = %q, want %q", msg.DeviceID, deviceID)
	}
	if msg.RouteID != routeID {
		t.Errorf("ws route_id = %q, want %q (server-side trip resolution)", msg.RouteID, routeID)
	}
	if diff := msg.Latitude - payload.Latitude; diff > epsilon || diff < -epsilon {
		t.Errorf("ws latitude = %v, want %v", msg.Latitude, payload.Latitude)
	}
	if diff := msg.Longitude - payload.Longitude; diff > epsilon || diff < -epsilon {
		t.Errorf("ws longitude = %v, want %v", msg.Longitude, payload.Longitude)
	}
	if msg.Speed != 0 {
		t.Errorf("ws speed = %v, want 0", msg.Speed)
	}
	if msg.Accuracy != payload.Accuracy {
		t.Errorf("ws accuracy = %v, want %v", msg.Accuracy, payload.Accuracy)
	}
	if msg.H3Hex != wantHex {
		t.Errorf("ws h3_hex = %q, want %q", msg.H3Hex, wantHex)
	}
	if msg.Timestamp != ts {
		t.Errorf("ws timestamp = %d, want %d", msg.Timestamp, ts)
	}

	// --- 15. Assert the Redis write. IngestService.IngestLocation logs and
	// swallows a SetDeviceState error and still returns 200 (see
	// services/ingest.go), so without this assertion a totally broken Redis
	// path would still pass this test. The cache write happens before the
	// broadcast (ingest.go ordering), and by now the WS message has already
	// been read, so no polling is needed here. ---
	rdb := goredis.NewClient(&goredis.Options{Addr: fmt.Sprintf("%s:%s", redisHost, redisPort)})
	defer rdb.Close()

	redisKey := "device:" + deviceID + ":loc"
	val, err := rdb.Get(setupCtx, redisKey).Result()
	if err != nil {
		t.Fatalf("get redis key %q: %v", redisKey, err)
	}

	var cachedLoc struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Speed     float64 `json:"speed"`
		HexRes9   string  `json:"hex_res9"`
		LastSeen  int64   `json:"last_seen"`
	}
	if err := json.Unmarshal([]byte(val), &cachedLoc); err != nil {
		t.Fatalf("unmarshal cached device location: %v", err)
	}
	if diff := cachedLoc.Latitude - payload.Latitude; diff > epsilon || diff < -epsilon {
		t.Errorf("redis latitude = %v, want %v", cachedLoc.Latitude, payload.Latitude)
	}
	if diff := cachedLoc.Longitude - payload.Longitude; diff > epsilon || diff < -epsilon {
		t.Errorf("redis longitude = %v, want %v", cachedLoc.Longitude, payload.Longitude)
	}
	if cachedLoc.Speed != 0 {
		t.Errorf("redis speed = %v, want 0", cachedLoc.Speed)
	}
	if cachedLoc.HexRes9 != wantHex {
		t.Errorf("redis hex_res9 = %q, want %q", cachedLoc.HexRes9, wantHex)
	}

	// --- 16. Assert zones table exists and has seeded rows (Phase 2 migration proof). ---
	var zoneCount int
	if err := dbPool.QueryRow(setupCtx, "SELECT count(*) FROM zones").Scan(&zoneCount); err != nil {
		t.Fatalf("query zones count: %v", err)
	}
	if zoneCount != 15 {
		t.Errorf("zones count = %d, want 15", zoneCount)
	}
}

// startPostgres starts the real production Postgres image
// (timescale/timescaledb-ha:pg15-latest, matching infra/docker-compose.yml)
// with a deterministic two-part wait strategy: the image restarts itself
// once during init, so "ready to accept connections" must be matched twice
// before the port is actually serving. 120s startup timeout accounts for
// this image running under emulation on arm64 hosts, where the manifest is
// amd64-only.
func startPostgres(ctx context.Context) (*tcpostgres.PostgresContainer, error) {
	pgWait := wait.ForAll(
		wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
		wait.ForListeningPort("5432/tcp"),
	).WithStartupTimeoutDefault(120 * time.Second)

	return tcpostgres.Run(ctx, postgresImage,
		tcpostgres.WithDatabase(dbName),
		tcpostgres.WithUsername(dbUser),
		tcpostgres.WithPassword(dbPass),
		testcontainers.WithWaitStrategy(pgWait),
	)
}

// startRedis starts the real production Redis image (redis:7-alpine,
// matching infra/docker-compose.yml). The module's default wait strategy
// (listening port + "Ready to accept connections" log line) is sufficient.
func startRedis(ctx context.Context) (*tcredis.RedisContainer, error) {
	return tcredis.Run(ctx, redisImage)
}

// mustParseHostPort extracts host and port from a postgres connection URL of
// the form postgres://user:pass@host:port/db?args.
func mustParseHostPort(t *testing.T, connStr, label string) (string, string) {
	t.Helper()
	// database/sql isn't used to connect (pgx is), but its URL parsing via
	// url.Parse keeps this dependency-free and exact.
	rest := strings.TrimPrefix(connStr, "postgres://")
	atIdx := strings.LastIndex(rest, "@")
	if atIdx == -1 {
		t.Fatalf("parse %s connection string %q: no '@'", label, connStr)
	}
	rest = rest[atIdx+1:]
	slashIdx := strings.Index(rest, "/")
	if slashIdx == -1 {
		t.Fatalf("parse %s connection string %q: no '/'", label, connStr)
	}
	hostPort := rest[:slashIdx]
	colonIdx := strings.LastIndex(hostPort, ":")
	if colonIdx == -1 {
		t.Fatalf("parse %s connection string %q: no ':' in host:port", label, connStr)
	}
	return hostPort[:colonIdx], hostPort[colonIdx+1:]
}

// mustParseRedisHostPort extracts host and port from a redis connection URL
// of the form redis://host:port.
func mustParseRedisHostPort(t *testing.T, connStr string) (string, string) {
	t.Helper()
	rest := strings.TrimPrefix(connStr, "redis://")
	rest = strings.TrimPrefix(rest, "rediss://")
	colonIdx := strings.LastIndex(rest, ":")
	if colonIdx == -1 {
		t.Fatalf("parse redis connection string %q: no ':' in host:port", connStr)
	}
	return rest[:colonIdx], rest[colonIdx+1:]
}
