# Phase 1C — end-to-end integration test: execution spec

**Audience:** an executor with no prior context on this repository. Everything needed is
quoted inline. Where any other document in this repo disagrees with the source quoted here,
the source wins.

**Repository root:** the directory containing `backend/`, `frontend/`, `infra/`.
**Go module:** `transit-backend` (`backend/go.mod`), `go 1.23.0`; toolchain observed go1.26.0.
**All `go` commands run from `backend/`.**

---

## 0. STOP — this document is an approval request, not an approved plan

`CLAUDE.md` §8.3 ends with:

> - testcontainers is a new dependency — **ask before adding it.**

### Entry-check results (run, not assumed)

| Check | Command | Result |
|---|---|---|
| testcontainers in `go.mod` | `grep -rn 'testcontainers' backend/go.mod` | **absent** |
| testcontainers in `go.sum` | `grep -rn 'testcontainers' backend/go.sum` | **absent** (grep exit 2 — see note) |
| test directory | `ls backend/test/` | **does not exist** |
| compose images | `cat infra/docker-compose.yml` | `timescale/timescaledb-ha:pg15-latest`, `redis:7-alpine` |

Note on grep exit 2: `backend/go.sum` does not exist as a separate readable path in that
invocation's glob — the dependency is absent from the module graph either way. Confirm with
`go list -m all | grep testcontainers` (expect no output).

**testcontainers is genuinely new.** It is not a transitive dependency of anything already
present. Adding it pulls in the Docker client and its transitive tree (`docker/docker`,
`opencontainers/*`, `moby/*`, `containerd/*`, and others) — expect `go.mod`'s indirect block
to grow by roughly 40–60 entries and `go.sum` substantially. It is test-only: no production
binary imports it, and with the build tag in §2 it does not enter the default `go test ./...`
path.

**This document is the ask.** The council's Wave 3 adjudication
(`docs/council/03-verdict.md` §2.6, recoverable at commit `3922967` — the directory is
gitignored at `.gitignore:10`, so it is not in the working tree) ruled:

> **Ruling: IN, conditional on explicit approval.** The integration test is the single most
> valuable non-DEFECT-6 artifact in the entire plan — it's the only thing that proves the
> five-technology stack integrates rather than five components that each pass in isolation.
> Testcontainers is a test-only dependency. CLAUDE.md's "ask before adding" convention
> applies, and this verdict constitutes the ask: the PLAN.md will specify testcontainers as a
> test-only dependency, flagged for human approval before execution begins.

**Do not run `go get` and do not create any file until a human has approved. Produce nothing
past this document until then.**

### Two blockers the approver must decide on at the same time

**(a) Docker is not running on this machine.**
`docker version` fails: `failed to connect to the docker API at unix:///Users/…/docker.sock`.
Nothing in this plan can be verified until a Docker daemon is up.

**(b) The production Postgres image is amd64-only and this machine is arm64.**
Verified against the Docker Hub registry API, not assumed:

| Image | Manifest type | Architectures | Last pushed |
|---|---|---|---|
| `timescale/timescaledb-ha:pg15-latest` | single manifest (`manifest.v2+json`) | **amd64 only** | **2023-04-24** |
| `redis:7-alpine` | manifest list | amd64, arm64, arm, 386, ppc64le, riscv64, s390x | current |

`uname -m` on this machine returns `arm64`. So the Postgres container runs under emulation
here. It does run, but slowly — this is the single largest contributor to the wall-clock
figures in §8 and the reason a 60-second first run is not a hang.

Newer tags in the same repository *are* multi-arch:

```
pg15                  ['amd64', 'arm64']
pg15-ts2.28           ['amd64', 'arm64']
pg15.18-ts2.28.3      ['amd64', 'arm64']
```

**Recommendation: still pin `pg15-latest` in the test**, matching `docker-compose.yml`
exactly. The instruction not to let the test drift onto a different image than production
runs is the right call, and a test that passes on an image production never uses proves less
than a slow test on the real one. Accept the emulation cost.

**But raise this separately:** production is pinned to an image last pushed 2023-04-24,
roughly 3.5 years stale as of 2026-09-03, on a tag (`-latest`) that is mutable in name but
has not actually moved. Upgrading it is an infrastructure change (`CLAUDE.md` §5.3, Phase
5+), out of scope here. Record it in `IDEAS.md` as **D32** (§7).

---

## 1. What this test must prove

One path, end to end, against the real stack:

```
POST /api/location  →  row in location_history  →  connected WS client gets LOCATION_UPDATE
```

Not mocks. Real Postgres (TimescaleDB + PostGIS), real Redis, the real Echo server, the real
hub, a real WebSocket client. `CLAUDE.md` §8.2:

> - **Integration:** `backend/test/integration_test.go`, testcontainers with real Postgres
>   (TimescaleDB + PostGIS) and Redis. One end-to-end path: `POST /api/location` → row in
>   `location_history` → connected WebSocket client receives the update.

It is also a **live regression check for DEFECT-5** (numeric fields must not be dropped by
`omitempty`) and for **DEFECT-2** (the envelope's flat shape and `LOCATION_UPDATE` literal).

---

## 2. Current source, verbatim

Everything below is the real code as of this writing. The executor extends it; it does not
rewrite it from memory.

### 2.1 `backend/cmd/server/main.go` — the composition root the test must replicate

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"transit-backend/internal/cache"
	"transit-backend/internal/config"
	"transit-backend/internal/database"
	"transit-backend/internal/handlers"
	"transit-backend/internal/hub"
	"transit-backend/internal/middleware"
	"transit-backend/internal/services"
	"transit-backend/migrations"

	"github.com/labstack/echo/v4"
)

// Compile-time interface satisfaction checks. These live in the composition
// root because it is the one place allowed to import both an interface's
// package and its concrete implementation's package without inverting the
// handlers -> services -> repositories -> database layering (CLAUDE.md
// Sec 3.2). A service or handler package importing its own concrete
// dependency here would be the layering violation DEFECT-1 was.
var (
	_ handlers.ArrivalService        = (*services.ArrivalsService)(nil)
	_ handlers.StopsService          = (*services.StopsService)(nil)
	_ handlers.RoutesService         = (*services.RoutesService)(nil)
	_ handlers.IngestService         = (*services.IngestService)(nil)
	_ handlers.NearbyService         = (*services.GeofencingService)(nil)
	_ services.StopRepository        = (*database.StopRepository)(nil)
	_ services.TripRepository        = (*database.TripRepository)(nil)
	_ services.LocationRepository    = (*database.LocationRepository)(nil)
	_ services.RouteRepository       = (*database.RouteRepository)(nil)
	_ services.DeviceCache           = (*cache.DeviceCache)(nil)
	_ services.DeviceRouteRepository = (*database.DeviceRouteRepository)(nil)
)

func main() {
	log.Println("Starting Transit Backend Server...")

	// 1. Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("Config loaded: ENV=%s, DB=%s@%s:%s/%s",
		cfg.Env, cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// 2. Connect to DB
	dbPool, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()
	log.Println("Database connected")

	// 2a. Run migrations
	if err := database.RunMigrations(context.Background(), dbPool, migrations.Files); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations up to date")

	// 3. Connect to Redis
	redisClient, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Redis connected")

	// 4. Initialize Repositories
	locRepo := database.NewLocationRepositoryWithResolution(dbPool, cfg.H3Resolution)
	stopRepo := database.NewStopRepository(dbPool)
	tripRepo := database.NewTripRepository(dbPool)
	routeRepo := database.NewRouteRepository(dbPool)
	deviceRouteRepo := database.NewDeviceRouteRepository(dbPool)

	// Legacy cache wrapper for backward compatibility
	deviceCache := cache.NewDeviceCache(redisClient)

	// 5. Initialize Services
	geoService := services.NewGeofencingServiceWithResolution(locRepo, stopRepo, cfg.H3Resolution)
	arrivalsService := services.NewArrivalsService(stopRepo, tripRepo, locRepo, geoService)
	stopsService := services.NewStopsService(stopRepo)
	routesService := services.NewRoutesService(routeRepo)

	// 6. Initialize Hub
	wsHub := hub.New()
	hubCtx, hubCancel := context.WithCancel(context.Background())
	go wsHub.Run(hubCtx)
	log.Println("WebSocket hub started")

	ingestService := services.NewIngestService(locRepo, deviceRouteRepo, deviceCache, wsHub)

	// 7. Initialize Handlers
	locHandler := handlers.NewLocationHandler(ingestService)
	routeHandler := handlers.NewRouteHandler(routesService)
	stopHandler := handlers.NewStopHandler(stopsService)
	arrivalHandler := handlers.NewArrivalHandler(arrivalsService)
	wsHandler := handlers.NewWebSocketHandler(wsHub)
	nearbyHandler := handlers.NewNearbyHandler(geoService)

	// 8. Setup Echo
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = middleware.HTTPErrorHandler

	// Middleware stack
	e.Use(middleware.Recovery)
	e.Use(middleware.CORS())
	e.Use(middleware.Logging)

	// 9. Register Routes

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// API routes
	api := e.Group("/api")

	// Location ingestion
	api.POST("/location", locHandler.IngestLocation)

	// Routes and stops
	api.GET("/routes", routeHandler.GetRoutes)
	api.POST("/routes", routeHandler.CreateRoute)
	api.GET("/stops", stopHandler.GetStops)

	// Arrivals
	api.GET("/arrivals", arrivalHandler.GetArrivals)

	// Nearby queries (H3+PostGIS)
	api.GET("/nearby/buses", nearbyHandler.GetNearbyBuses)
	api.GET("/nearby/stops", nearbyHandler.GetNearbyStops)

	// Geo utilities
	api.GET("/geo/hex", nearbyHandler.GetHexInfo)

	// WebSocket
	e.GET("/ws", wsHandler.HandleWS)

	// Serve static files
	e.Static("/", "public")

	// 10. Start Server
	go func() {
		addr := cfg.ServerHost + ":" + cfg.ServerPort
		log.Printf("Server starting on %s", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server", err)
		}
	}()

	// 11. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Printf("Echo shutdown error: %v", err)
	}

	hubCancel()
	wsHub.Shutdown()

	log.Println("Server stopped")
}
```

The test replicates steps 2→9 and the shutdown ordering of step 11. It does **not** replicate
`config.LoadConfig()` (which reads `.env` and the process environment), `e.Static`, or the
signal handling. It builds a `*config.Config` literal instead — see §4.3.

### 2.2 `backend/internal/handlers/location.go` — the endpoint under test

```go
package handlers

import (
	"net/http"
	"time"

	"transit-backend/internal/models"

	"github.com/labstack/echo/v4"
)

// LocationHandler handles GPS location ingestion. Depends only on the
// IngestService interface (CLAUDE.md Sec 3.2) -- no repository, cache, or
// hub imports.
type LocationHandler struct {
	service IngestService
}

// NewLocationHandler creates a new LocationHandler
func NewLocationHandler(service IngestService) *LocationHandler {
	return &LocationHandler{service: service}
}

// IngestLocation handles POST /api/location
// Receives GPS data, validates it, then delegates storage, cache, and
// broadcast to IngestService.
func (h *LocationHandler) IngestLocation(c echo.Context) error {
	var loc models.Location
	if err := c.Bind(&loc); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Validation
	if loc.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}
	if loc.Latitude < -90 || loc.Latitude > 90 || loc.Longitude < -180 || loc.Longitude > 180 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid coordinates"})
	}
	if loc.Speed < 0 || loc.Speed > 350 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid speed"})
	}

	// Set timestamp if missing
	if loc.Timestamp == 0 {
		loc.Timestamp = time.Now().Unix()
	}

	// IngestService.IngestLocation mutates loc.HexRes9 as a side effect
	// (calculated during Insert), which is why we still have it below.
	if err := h.service.IngestLocation(c.Request().Context(), &loc); err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Return success with H3 hex
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"hex_res9": loc.HexRes9,
	})
}
```

Response on success is `200` with `{"status":"ok","hex_res9":"<h3 cell>"}`. The test asserts
that `hex_res9` is non-empty and matches the DB row and the WS message.

### 2.3 `backend/internal/services/ingest.go` — what happens between HTTP and the hub

```go
func (s *IngestService) IngestLocation(ctx context.Context, loc *models.Location) error {
	if err := s.locRepo.Insert(ctx, loc); err != nil {
		return fmt.Errorf("insert location: %w", err)
	}

	// Route resolution, cache, and broadcast are enrichment/side effects
	// that don't fail the request -- the durable write already succeeded.
	routeID, err := s.routeRepo.GetActiveRouteID(ctx, loc.DeviceID)
	if err != nil {
		log.Printf("failed to resolve active route for %s: %v", loc.DeviceID, err)
	}

	deviceState := map[string]interface{}{
		"latitude":  loc.Latitude,
		"longitude": loc.Longitude,
		"speed":     loc.Speed,
		"hex_res9":  loc.HexRes9,
		"last_seen": loc.Timestamp,
	}
	if err := s.cache.SetDeviceState(ctx, loc.DeviceID, deviceState); err != nil {
		log.Printf("failed to update device cache for %s: %v", loc.DeviceID, err)
	}

	msg := hub.Message{
		Type:      hub.MsgTypeLocationUpdate,
		DeviceID:  loc.DeviceID,
		RouteID:   routeID,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		Speed:     loc.Speed,
		Accuracy:  loc.Accuracy,
		H3Hex:     loc.HexRes9,
		Timestamp: loc.Timestamp,
	}
	select {
	case s.hub.Broadcast <- msg:
	default:
		// Drop message if buffer full - backpressure handling
		log.Printf("WebSocket broadcast buffer full, message dropped for device %s", loc.DeviceID)
	}

	return nil
}
```

Three consequences the test design depends on:

1. **Only the DB insert can fail the request.** Redis and broadcast failures are logged and
   swallowed. A green `200` therefore does not by itself prove Redis was written — see §6.
2. **`route_id` is resolved server-side** from an `IN_PROGRESS` trip, never accepted on ingest.
3. **`loc.HexRes9` is mutated in place** by `Insert` before the message is built, so the WS
   `h3_hex`, the HTTP response `hex_res9`, and the DB `hex_res9` column must all be equal.

### 2.4 `backend/internal/hub/message.go` — the canonical envelope (DEFECT-2 / DEFECT-5)

```go
package hub

// Message defines the structure of messages broadcast to clients over the
// WebSocket connection. Canonical shape: CLAUDE.md Sec 7.2. Flat, no `data`
// wrapper, no `omitempty` on numerics -- a stopped device must serialize
// "speed": 0, not omit the key.
type Message struct {
	Type      string  `json:"type"`
	DeviceID  string  `json:"device_id"`
	RouteID   string  `json:"route_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
	Accuracy  float64 `json:"accuracy"`
	H3Hex     string  `json:"h3_hex"`
	Timestamp int64   `json:"timestamp"`
}

const (
	MsgTypeLocationUpdate = "LOCATION_UPDATE"
)
```

Nine keys. No `omitempty` anywhere. The type literal is uppercase `LOCATION_UPDATE`.

### 2.5 `backend/internal/hub/hub.go` — post-D28-fix, affects teardown

```go
const (
	// clientDrainTimeout bounds how long shutdown waits for a single client's
	// buffered messages to be consumed by its WritePump. A client that has not
	// drained within this window has its remaining buffered messages dropped —
	// an accepted trade-off so that one wedged client cannot hold up shutdown.
	clientDrainTimeout = 250 * time.Millisecond

	// clientDrainPollInterval is how often the drain loop re-checks len(c.Send).
	clientDrainPollInterval = 1 * time.Millisecond
)

type Hub struct {
	Clients    map[*Client]bool
	Broadcast  chan interface{}
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
	done       chan struct{}
}

func New() *Hub {
	return &Hub{
		Broadcast:  make(chan interface{}, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		done:       make(chan struct{}),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			clients := make([]*Client, 0, len(h.Clients))
			for client := range h.Clients {
				clients = append(clients, client)
				delete(h.Clients, client)
			}
			h.mu.Unlock()

			var wg sync.WaitGroup
			for _, client := range clients {
				wg.Add(1)
				go func(c *Client) {
					defer wg.Done()
					drainAndClose(c)
				}(client)
			}
			wg.Wait()

			close(h.done)
			return
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
		case message := <-h.Broadcast:
			h.mu.RLock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					DroppedMessages.Add(1)
					log.Printf("hub: dropped message for client %p (buffer full)", client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Shutdown blocks until Run has drained all clients and exited.
func (h *Hub) Shutdown() {
	<-h.done
}

func drainAndClose(c *Client) {
	defer close(c.Send)

	if len(c.Send) == 0 {
		return
	}

	deadline := time.After(clientDrainTimeout)
	ticker := time.NewTicker(clientDrainPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			remaining := len(c.Send)
			DroppedMessages.Add(int64(remaining))
			log.Printf("hub: shutdown drain timed out for client %p, dropped %d buffered messages", c, remaining)
			return
		case <-ticker.C:
			if len(c.Send) == 0 {
				return
			}
		}
	}
}
```

Note `h.mu` is **unexported**. The test lives in a different package and therefore **cannot**
lock it. Reading `h.Clients` from the test without that lock is a data race that `-race` will
fail on. This is why §3.4 adds one small accessor.

### 2.6 `backend/internal/handlers/websocket.go` — where the registration race lives

```go
func (h *WebSocketHandler) HandleWS(c echo.Context) error {
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Println(err)
		return err
	}

	client := &hub.Client{Hub: h.hub, Conn: conn, Send: make(chan interface{}, 256)}
	client.Hub.Register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.WritePump()
	go client.ReadPump()

	return nil
}
```

**`upgrader.Upgrade` writes the HTTP 101 response before `Register <- client` runs.** The
client's `websocket.Dial` can therefore return successfully while the hub has not yet
registered it. A `POST` issued immediately after a successful dial can be broadcast to zero
clients. This is flakiness source #4 in §5 and the reason for §3.4.

### 2.7 `backend/internal/database/db.go` — the real migration path

```go
// RunMigrations applies any unapplied SQL migrations from the provided filesystem.
// It tracks applied migrations in a schema_migrations table.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrations fs.FS) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrations, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version := entry.Name()

		var exists bool
		if err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version,
		).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if exists {
			continue
		}

		sql, err := fs.ReadFile(migrations, version)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}

		if _, err := pool.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)", version,
		); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}

		log.Printf("Applied migration: %s", version)
	}
	return nil
}
```

Driven by `backend/migrations/migrations.go`:

```go
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
```

`fs.ReadDir` returns entries **sorted by filename**, so `001…`, `002…`, `003…`, `004…` apply
in order. The test calls this exact function with this exact `embed.FS`.

### 2.8 `backend/migrations/*.sql` — the schema the container must end up with

`001_create_tables.sql`:

```sql
-- Enable extensions
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS postgis;

-- Devices table
CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    name TEXT,
    status TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Routes table
CREATE TABLE IF NOT EXISTS routes (
    id TEXT PRIMARY KEY,
    name TEXT,
    description TEXT
);

-- Stops table
CREATE TABLE IF NOT EXISTS stops (
    id TEXT PRIMARY KEY,
    route_id TEXT REFERENCES routes(id),
    name TEXT,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    sequence_number INT,
    geom GEOMETRY(POINT, 4326)
);

-- Location History table (Hypertable)
CREATE TABLE IF NOT EXISTS location_history (
    time TIMESTAMP NOT NULL,
    device_id TEXT REFERENCES devices(id),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    speed FLOAT,
    accuracy FLOAT,
    metadata JSONB
);

-- Convert to hypertable
SELECT create_hypertable('location_history', 'time', if_not_exists => TRUE);

-- Trips table
CREATE TABLE IF NOT EXISTS trips (
    id TEXT PRIMARY KEY,
    route_id TEXT REFERENCES routes(id),
    device_id TEXT REFERENCES devices(id),
    started_at TIMESTAMP,
    current_stop INT,
    status TEXT
);
```

**`location_history.device_id` is `TEXT REFERENCES devices(id)`.** A `POST /api/location` for
a device that does not exist in `devices` fails the insert and returns **500**, not 200. The
test must create its device first. This is the single most likely way to get a confusing red
run.

`002_add_indexes.sql` adds two indexes. `003_add_h3_postgis.sql` is the important one — it
adds `hex_res9`, adds the PostGIS `geom` column via `AddGeometryColumn`, creates four
indexes, sets TimescaleDB compression, and installs the trigger that generates geometry:

```sql
ALTER TABLE location_history ADD COLUMN IF NOT EXISTS hex_res9 TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'location_history' AND column_name = 'geom'
    ) THEN
        PERFORM AddGeometryColumn('location_history', 'geom', 4326, 'POINT', 2);
    END IF;
END $$;
```

```sql
CREATE OR REPLACE FUNCTION update_location_geom()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL THEN
        NEW.geom := ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'trg_location_geom'
    ) THEN
        CREATE TRIGGER trg_location_geom
        BEFORE INSERT ON location_history
        FOR EACH ROW
        EXECUTE FUNCTION update_location_geom();
    END IF;
END $$;
```

This trigger is why the test can assert `geom IS NOT NULL` without the app ever writing it —
a genuine PostGIS integration signal, and the only one this path produces.

`004_seed_data.sql` seeds 5 devices (`bus-001`…`bus-005`), 3 routes, 15 stops, trips, and —
guarded so it is idempotent — 11 `location_history` rows:

```sql
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM location_history) THEN
        INSERT INTO location_history (...) VALUES ...
    END IF;
END $$;
```

**The test's assertions must therefore be scoped to its own device id, never a bare
`SELECT count(*) FROM location_history`,** which is 11 before the test posts anything.

### 2.9 `infra/docker-compose.yml` — the images to match

```yaml
  postgres:
    image: timescale/timescaledb-ha:pg15-latest
    environment:
      POSTGRES_USER: transit_user
      POSTGRES_PASSWORD: transit_password
      POSTGRES_DB: transit_poc
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/home/postgres/pgdata/data
      - ../backend/migrations:/docker-entrypoint-initdb.d:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U transit_user -d transit_poc"]

  redis:
    image: redis:7-alpine
    command: >
      redis-server 
      --maxmemory 256mb 
      --maxmemory-policy allkeys-lru
      --appendonly yes
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
```

Note the compose `postgres` service **also** bind-mounts `migrations/` into
`/docker-entrypoint-initdb.d`. That is the DEFECT-7 double-apply mechanism. **The test must
not replicate that mount** — see §3.3.

---

## 3. Container setup

### 3.1 Which modules

Three modules, all under the testcontainers-go umbrella:

| Module | Purpose |
|---|---|
| `github.com/testcontainers/testcontainers-go` | core (`wait` strategies, `testcontainers.Container`) |
| `github.com/testcontainers/testcontainers-go/modules/postgres` | Postgres lifecycle + `ConnectionString` |
| `github.com/testcontainers/testcontainers-go/modules/redis` | Redis lifecycle + `ConnectionString` |

Add them with:

```bash
cd backend
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest
go get github.com/testcontainers/testcontainers-go/modules/redis@latest
go mod tidy
```

**Verify the resolved API before writing against it.** testcontainers-go renamed the module
constructors: older versions expose `postgres.RunContainer(ctx, opts...)`, newer ones expose
`postgres.Run(ctx, image, opts...)` with `RunContainer` deprecated. Do not guess:

```bash
go doc github.com/testcontainers/testcontainers-go/modules/postgres | head -40
go doc github.com/testcontainers/testcontainers-go/modules/redis | head -40
```

Use whichever constructor the resolved version actually documents.

### 3.2 Exact images

Pin these, matching `docker-compose.yml` byte for byte:

```go
const (
	postgresImage = "timescale/timescaledb-ha:pg15-latest"
	redisImage    = "redis:7-alpine"
)
```

Credentials matching compose (`postgres.WithDatabase` / `WithUsername` / `WithPassword`, or
the equivalent env for a generic container):

```
POSTGRES_DB       transit_poc
POSTGRES_USER     transit_user
POSTGRES_PASSWORD transit_password
```

On arm64 the Postgres container runs emulated (§0b). If the resolved testcontainers version
supports it, no platform override is needed — Docker selects the only manifest available.
Do **not** add `--platform` juggling or swap the tag to make it faster; that is the drift the
task explicitly forbids.

### 3.3 Migrations: the real path, not the shortcut

**Decision: apply migrations by calling the app's own `database.RunMigrations(ctx, pool,
migrations.Files)`. Do not use `postgres.WithInitScripts`, and do not mount
`migrations/` into `/docker-entrypoint-initdb.d`.**

Three reasons, in order of weight:

1. **It is the path production actually uses.** `main.go` step 2a calls `RunMigrations` on
   every boot. A test that instead lets Postgres's entrypoint run the `.sql` files proves the
   files parse, but never executes `RunMigrations` — leaving the `schema_migrations` ledger,
   the sorted-apply order, and the already-applied skip logic completely untested. The whole
   point of this test is that the real wiring works.
2. **The init-script route reproduces DEFECT-7.** Compose mounts the same directory into
   `/docker-entrypoint-initdb.d` *and* lets the Go binary replay it — that double-apply is a
   closed defect whose fix was to guard the seed inserts. Replicating the mount in the test
   would re-run a known-bad configuration for no benefit.
3. **It keeps the failure legible.** A migration failure surfaces as a Go error with the
   filename (`apply migration 003_add_h3_postgis.sql: …`) rather than as container startup
   log noise.

Concretely, after the container is ready the test builds a `*config.Config` pointing at the
mapped host/port and calls `database.New(cfg)` then `database.RunMigrations(...)` — exactly
`main.go` steps 2 and 2a.

### 3.4 The one production-code addition: `Hub.ClientCount()`

§2.6 establishes that a successful WS dial does not imply hub registration, and §2.5
establishes the test cannot take `h.mu`. Without an observable registration signal the test
must either sleep (forbidden by §8.3) or POST in a retry loop (which writes an
unpredictable number of rows and weakens the DB assertion).

**Add exactly this to `backend/internal/hub/hub.go`:**

```go
// ClientCount returns the number of currently registered clients. Exported so
// out-of-package tests can wait for registration deterministically without
// touching the unexported mutex.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.Clients)
}
```

This is additive and read-only. It does not touch `Run`, `Shutdown`, `drainAndClose`, either
drain constant, or the envelope — nothing on the §9 must-not-touch list. It mirrors what the
hub's own in-package helper `waitForClients` already does. Five lines, and it is the
difference between a deterministic test and a flaky one.

If the approver rejects even this, the fallback is the retry-POST loop; say so explicitly and
change the DB assertion to "the most recent row for this device matches", accepting N rows.

---

## 4. Test file: location, package, build tag

### 4.1 Path

`backend/test/integration_test.go` — exactly as `CLAUDE.md` §8.2 names it. The directory does
not exist; create it. Nothing in `.gitignore` matches `backend/test/` (line 6 is `/server`,
anchored to the root, and was already narrowed per D26).

### 4.2 Package and build tag

```go
//go:build integration

package integration
```

**Decision: yes, gate it behind a build tag.** Justification:

- The repo's routine verification loop is `go test ./... -race -count=1`. Every prior phase
  ran it dozens of times. If this test ran by default, that loop would require a live Docker
  daemon and grow by ~60s — and would fail outright on this machine right now, where Docker
  is not running (§0a).
- The failure mode of an ungated Docker test is bad: a red suite that says nothing about the
  code. Tests that fail for environmental reasons train people to ignore red.
- `CLAUDE.md` §8.3's "a test skipped for more than one commit gets deleted" targets
  `t.Skip()` rot, not deliberate suite separation. A tagged suite is not skipped; it is a
  different suite with its own command, and §7's verification command runs it.

Trade-off to state plainly in the commit message: `internal/database` and `internal/cache`
are listed in §8.1 as "Covered by integration test", and with the tag they are covered
**only** under `-tags=integration`. A default `go test ./... -cover` will still show them at
0%. That is honest and expected; do not report it as a coverage regression.

### 4.3 Config construction

`config.LoadConfig()` reads `.env` and process env and would point at localhost. Build the
struct directly from the container's mapped endpoints instead:

```go
cfg := &config.Config{
	DBHost: pgHost, DBPort: pgPort,
	DBName: "transit_poc", DBUser: "transit_user", DBPassword: "transit_password",
	DBMaxConns: 10, DBMinConns: 2,
	RedisHost: redisHost, RedisPort: redisPort, RedisPassword: "",
	RedisPoolSize: 10,
	ServerHost: "127.0.0.1", ServerPort: "0",
	H3Resolution: 9,
	Env: "test", LogLevel: "debug",
}
```

`config.Config` and all these fields are exported. `DBMaxConns: 50` from compose is
unnecessary for one test; 10 is fine and starts faster. `H3Resolution: 9` matches compose.

---

## 5. The exact sequence

Numbered, in order. Every wait is deterministic — no `time.Sleep` anywhere (`CLAUDE.md` §8.3).

**1. Guard the environment.** At the top of the test, fail fast with a clear message if
Docker is unreachable, rather than emitting a 2-minute timeout. testcontainers surfaces this
on the first `Run`; wrap that error with "is the Docker daemon running?".

**2. Start both containers, concurrently.** Postgres and Redis are independent. Starting them
in parallel saves roughly the smaller of the two startup times. Use the module wait
strategies (§6), and give the whole setup a `context.WithTimeout` of **180s** — generous
because of arm64 emulation and a possible cold image pull (§8).

**3. Register cleanup immediately after each container is created**, before any assertion
that could fail. See §6 for exact ordering.

**4. Read mapped endpoints.** `pgContainer.ConnectionString(ctx, "sslmode=disable")` or
`Host()` + `MappedPort(ctx, "5432/tcp")`; same shape for Redis on `6379/tcp`. Never assume
5432/6379 on the host — testcontainers maps to a random free port, which is precisely what
makes parallel runs safe.

**5. Build `cfg` (§4.3), then `database.New(cfg)`.** This is `main.go` step 2, including its
`pool.Ping`.

**6. `database.RunMigrations(ctx, dbPool, migrations.Files)`.** `main.go` step 2a. Assert no
error. This applies 001–004 in filename order and records them in `schema_migrations`.

**7. `cache.New(cfg)`.** `main.go` step 3. Assert no error — this proves Redis is reachable.

**8. Seed the test's own fixtures.** Directly via `dbPool.Exec`, using a device id unique to
the run (e.g. `itest-dev-<nanos>`) so the test is isolated from the 5 seeded buses and from
concurrent runs:

```sql
INSERT INTO routes (id, name, description) VALUES ($1, 'itest route', '') ON CONFLICT DO NOTHING;
INSERT INTO devices (id, name, status) VALUES ($2, 'itest device', 'ACTIVE') ON CONFLICT DO NOTHING;
INSERT INTO trips (id, route_id, device_id, started_at, status)
VALUES ($3, $1, $2, NOW(), 'IN_PROGRESS') ON CONFLICT DO NOTHING;
```

The device row is **required** — the `location_history` FK (§2.8) rejects the insert
otherwise. The trip is what makes `route_id` non-empty in the broadcast, exercising
`DeviceRouteRepository.GetActiveRouteID` rather than its empty-string path.

**9. Wire the app exactly as `main.go` steps 4–9**, minus `e.Static`: repositories, device
cache, services, `hub.New()` + `go wsHub.Run(hubCtx)`, `IngestService`, `LocationHandler`,
`WebSocketHandler`, Echo with the same three middlewares and `HTTPErrorHandler`, and the two
routes this test needs (`POST /api/location`, `GET /ws`). Registering only those two is fine
and keeps the test's intent legible.

**10. Start the server on a random port.** Prefer `httptest.NewServer(e)` — it binds
`127.0.0.1:0`, hands back the URL, and its `Close()` is well-defined. `e.Start(":0")` does
not expose the chosen port without reaching into `e.Listener`. Using `httptest` is a
deliberate, documented divergence from `main.go`'s `e.Start(addr)`: the handler chain,
middleware, and routing under test are identical; only the listener differs.

**11. Dial the WebSocket.** `websocket.DefaultDialer.Dial(wsURL, nil)` where `wsURL` is the
server URL with `http` → `ws` and path `/ws`. Assert no error and a 101 response.

**12. Wait for hub registration — deterministically.** Poll `wsHub.ClientCount() == 1` with a
`context.WithTimeout` (5s) and a `default:` yield, in the shape the hub's own tests already
use. **This is the step that makes the whole test reliable**; without it, step 14's broadcast
can reach zero clients (§2.6).

**13. Set a read deadline on the WS client.** `wsConn.SetReadDeadline(time.Now().Add(10 *
time.Second))`. This bounds `ReadMessage` without a sleep, so a lost broadcast fails in 10s
with a clear I/O timeout instead of hanging until the Go test binary's 10-minute panic.

**14. POST the payload.** `POST {serverURL}/api/location`, `Content-Type: application/json`:

```json
{
  "device_id": "itest-dev-<nanos>",
  "latitude": 28.4595,
  "longitude": 77.0266,
  "speed": 0,
  "accuracy": 5.5,
  "timestamp": <a fixed unix seconds value the test picks>
}
```

**`"speed": 0` is deliberate — it is the DEFECT-5 trigger.** A stopped device is the most
common real state and the exact case `omitempty` used to erase. Pick and keep an explicit
`timestamp` so the DB row can be located by exact primary-key-ish match rather than a range.

Assert: status `200`; body decodes to `{"status":"ok","hex_res9":"<non-empty>"}`. Capture
`hex_res9` — call it `wantHex`.

**15. Assert the DB row.** Scoped to this device and timestamp, never a global count:

```sql
SELECT latitude, longitude, speed, accuracy, hex_res9, (geom IS NOT NULL)
FROM location_history
WHERE device_id = $1 AND time = $2
```

with `$2 = time.Unix(ts, 0)` — matching `LocationRepository.Insert`, which stores
`time.Unix(loc.Timestamp, 0)`. Assert exactly one row, and:

- `latitude` = 28.4595, `longitude` = 77.0266 — note both columns are `DECIMAL(10,8)` /
  `DECIMAL(11,8)`. **`DECIMAL(11,8)` holds at most 3 integer digits**, so longitude 77.0266
  fits; a longitude ≥ 1000 would not, and latitude is capped at 2 integer digits by
  `DECIMAL(10,8)`. Scan into `float64` and compare with a small epsilon (1e-6), not `==`.
- `speed` = 0, `accuracy` = 5.5
- `hex_res9` = `wantHex` (proves the H3 value the API returned is the value persisted)
- `geom IS NOT NULL` — **proves the PostGIS trigger fired.** The app never writes this column.

**16. Assert the WebSocket message.** `wsConn.ReadMessage()`, then — and this is the part
that must not be shortcut — unmarshal into `map[string]json.RawMessage` **first**:

```go
var raw map[string]json.RawMessage
// assert all nine keys are PRESENT:
// type, device_id, route_id, latitude, longitude, speed, accuracy, h3_hex, timestamp
```

Key presence is the DEFECT-5 assertion. Unmarshalling straight into a struct cannot
distinguish "absent" from "present and zero" — which is the exact bug DEFECT-5 was. With
`speed: 0` and a `float64` field, a dropped key and a zero value both yield `0`.

Then unmarshal into `hub.Message` (or an equivalent local struct) and assert values:

| Field | Expected |
|---|---|
| `type` | `"LOCATION_UPDATE"` — uppercase; the lowercase literal was DEFECT-2 |
| `device_id` | the test's device id |
| `route_id` | the test's route id (non-empty — proves server-side trip resolution) |
| `latitude` / `longitude` | 28.4595 / 77.0266 |
| `speed` | `0` |
| `accuracy` | `5.5` |
| `h3_hex` | `wantHex` — same value as the HTTP response and the DB column |
| `timestamp` | the exact unix seconds posted |

Also assert **`heading` is absent** from `raw`. `CLAUDE.md` §7.2 and
`hub/message_test.go:51-52` both pin this; asserting it here makes the guard end-to-end.

**17. Teardown** — §6.

### 5.1 Why one test function, not several

`CLAUDE.md` §8.2 asks for "one end-to-end path". Container startup dominates the runtime, so
extra cases are cheap *within* one function but expensive as separate top-level tests that
each spin up their own stack. If more integration cases are added later, share one
`TestMain` (or a package-level `setupStack(t)` with `sync.Once`) rather than starting
containers per test. Do not build that machinery now — YAGNI; one test, one stack.

Name it `TestIntegration_LocationIngestToWebSocketBroadcast`.

---

## 6. Teardown

### 6.1 Ordering, and why it avoids the D28 drain penalty

Mirror `main.go` step 11, then close the client, then the containers:

```go
// 1. HTTP server first — stops new requests. Echo/httptest does NOT close
//    hijacked WebSocket connections, so the server-side WritePump stays alive.
srv.Close()

// 2. Hub next, in main.go's order. Because WritePump is still running and
//    actively reading client.Send, drainAndClose sees len(Send)==0 (or drains
//    within a tick or two) and takes the fast path — NOT the 250ms timeout.
hubCancel()
wsHub.Shutdown()

// 3. Now the WS client. Nothing is left to deliver to it.
wsConn.Close()

// 4. Connections to the containers.
dbPool.Close()
redisClient.Close()

// 5. Containers last (registered via t.Cleanup at creation — see 6.2).
```

**Answering the question directly: the test does not need to wait for `clientDrainTimeout`,
and it must not close the WS client first.** The drain polls `len(c.Send) == 0`. If the test
closed `wsConn` before shutting the hub down, the server-side `WritePump` would hit a write
error and return, leaving nobody to consume `Send`; any message still buffered would then
hold shutdown for the full 250ms before being dropped. Shutting the hub down *while*
`WritePump` is alive means the buffer is already empty (the test consumed the only message in
step 16) and `drainAndClose` returns immediately via its `len(c.Send) == 0` fast path.

Worst case if something is unexpectedly buffered: 250ms, once, in parallel across clients.
With one client that is 250ms. Harmless either way — but the fast path is the correct
ordering and it mirrors production.

### 6.2 Containers must die even on failure

Register cleanup **immediately after each container is created**, before any `require`/
`t.Fatal` that could unwind:

```go
pgC, err := postgres.Run(ctx, postgresImage, ...)
if pgC != nil {
	t.Cleanup(func() {
		if err := pgC.Terminate(context.Background()); err != nil {
			t.Logf("terminate postgres: %v", err)
		}
	})
}
if err != nil {
	t.Fatalf("start postgres (is the Docker daemon running?): %v", err)
}
```

Three details that matter:

- **`if pgC != nil` before the error check.** testcontainers can return both a non-nil
  container and an error (started but unhealthy); terminating it still needs to happen.
- **`context.Background()` inside the cleanup, not the setup `ctx`.** The setup context has a
  180s timeout and may already be expired or cancelled when cleanup runs, which would make
  `Terminate` fail and leak the container.
- **`t.Cleanup`, not `defer`.** Cleanups run LIFO after the test (including after subtests),
  and survive `t.Fatal` in a way that `defer` in a helper function does not.

---

## 7. Flakiness sources and the deterministic wait for each

| # | Source | Wait strategy |
|---|---|---|
| 1 | **Postgres not ready** — TCP accepts before the DB does; the entrypoint starts, stops, and restarts the server during init, so the readiness line appears **twice** | `wait.ForLog("database system is ready to accept connections").WithOccurrence(2)` — the postgres module's `BasicWaitStrategies()` already encodes this; prefer it over hand-rolling. Occurrence 2 is the whole point: matching once catches the *pre-init* server and connections then get refused. Give it `.WithStartupTimeout(120 * time.Second)` for emulation. |
| 2 | **Redis not ready** | `wait.ForLog("Ready to accept connections")` or `wait.ForListeningPort("6379/tcp")`. The redis module's default is sufficient. |
| 3 | **Image pull on a cold machine** | Not a wait strategy — a timeout budget. The Postgres image is ~1.06 GB compressed (registry-reported layer size). Pre-pull it in pre-flight (§8) so the first `go test` is not also a download. |
| 4 | **WS dial returns before hub registration** (§2.6 — the real one) | Poll `wsHub.ClientCount() == 1` against a `context.WithTimeout(5s)`. Requires §3.4. |
| 5 | **Broadcast arrives before the test reads** | Not a race. `client.Send` is buffered at 256 and the TCP socket buffers too; the message waits. `SetReadDeadline` bounds the read. |
| 6 | **Host port collision** | None by construction — testcontainers maps to random free host ports. Never hardcode 5432/6379. This is also what lets two runs proceed concurrently. |
| 7 | **Migration race** | None by construction — one pool, one sequential `RunMigrations` call, before the server starts. There is no concurrent applier. (The DEFECT-7 double-apply came from the compose init-script mount, which §3.3 deliberately does not replicate.) |
| 8 | **Clock/timestamp mismatch** | Use one fixed `ts := time.Now().Unix()` captured once, sent in the payload and reused in the `WHERE time = $1` lookup. Do not call `time.Now()` twice and expect equality. |
| 9 | **Emulated-arch slowness** (§0b) | Budget, not a wait: 180s setup context, 120s container startup timeout. |

No `time.Sleep`. Every wait above is a testcontainers wait strategy, a context deadline, a
socket deadline, or a bounded poll.

---

## 8. Out of scope, and the Redis question

### 8.1 Redis: what the test can and cannot prove

`CLAUDE.md` §3.1 records Redis as **write-only**:

> Latest known location per device — **write-only today.** Nothing reads it back:
> `grep -rn "GetDeviceState\|GetDeviceLocation" backend/` finds no external caller (D21).

There is no read-back method in the `cache` package to assert against — `CacheStore`
(`cache/interface.go:18-27`) declares only `SetDeviceLocation`, `Ping`, and `Close`.

**But a clean assertion IS possible, and the test should make it.** The key format is fully
determined by source:

```go
// cache/redis.go:49  — SetDeviceLocation
key := fmt.Sprintf("%s%s:loc", KeyPrefixDevice, deviceID)
// cache/interface.go:41 — KeyPrefixDevice = "device:"
```

So the key is `device:<deviceID>:loc` and the value is `json.Marshal` of
`cache.DeviceLocation` (`{latitude, longitude, speed, hex_res9, last_seen}`) with a 5-minute
TTL. The test can open its **own** `redis.Client` — `github.com/redis/go-redis/v9` is already
a direct dependency (`go.mod`) — against the same container and `GET` that key.

**Decision: assert it.** Requiring no new production code, it proves the third of the three
side effects really happened rather than being silently swallowed (§2.3 — a `SetDeviceState`
failure is logged and the request still returns 200, so **without this assertion a totally
broken Redis path still passes the test**). Assert the key exists, and that the decoded
`latitude`/`longitude`/`speed`/`hex_res9` match what was posted.

One caveat to encode: the write is a side effect that races the HTTP response only in the
sense that it happens *before* the broadcast (§2.3 ordering — cache write, then broadcast). By
the time the test has read the WS message in step 16, the Redis write has already happened.
Assert Redis **after** the WS assertion and no polling is needed. If the executor prefers
belt-and-braces, poll `EXISTS` with a 5s bounded loop — still no sleep.

Note also `last_seen`: `SetDeviceState` does `state["last_seen"].(int64)` and `ingest.go`
passes `loc.Timestamp` (an `int64`), so the assertion holds. Do not assert on it via a
`float64` cast.

### 8.2 Genuinely out of scope for this test

- Geofencing / nearby / arrivals endpoints — separate paths, separate tests.
- The `frontend/` WebSocket client — Go-side only.
- Multi-client fan-out, slow consumers, disconnect-mid-broadcast — already covered by the
  hub's unit tests, which do it faster and without Docker.
- Load or throughput numbers — `CLAUDE.md` §9, and they belong in `RESULTS.md`.
- Migration rollback / down-migrations — none exist.

### 8.3 New findings to append to `IDEAS.md`

Add these; do not fix them here.

**D32 — production Postgres image is amd64-only and 3.5 years stale.**
`infra/docker-compose.yml:12` pins `timescale/timescaledb-ha:pg15-latest`. Registry API
reports a single-platform manifest, `architecture: amd64`, `tag_last_pushed
2023-04-24T08:00:26Z`. On arm64 developer machines it runs under emulation, materially
slowing container startup (and now the integration test). Multi-arch tags exist in the same
repo (`pg15`, `pg15-ts2.28`, `pg15.18-ts2.28.3` all list `['amd64','arm64']`). Changing the
pin is an infrastructure change (`CLAUDE.md` §5.3, Phase 5+); recorded, not scheduled.

**D33 — `D21` is cited in `CLAUDE.md` but has no `IDEAS.md` entry.**
`CLAUDE.md` §3.1's Redis row cites "(D21)" for the write-only finding, and §4's "Tracked but
not scheduled" cites `D21` again. `IDEAS.md` has no `D21` — its identifiers jump D19 → D26.
The finding itself is real and verifiable; only the tracking id dangles. Either add the entry
or drop the citation. (Same section, unrelated: the Redis row's last sentence is duplicated —
"Pool size hard-coded 50 in `cache/redis.go:28`; `REDIS_POOL_SIZE` unused." appears twice.)

---

## 9. Pre-flight, verification, and expected timing

### 9.1 Pre-flight — must all pass before writing any code

```bash
# 0. Approval for the new dependency has been given (§0). If not, stop.

# 1. Docker daemon is up — currently FAILING on this machine (§0a)
docker version --format '{{.Server.Version}}'

# 2. Baseline is green
cd backend
go build ./...
go vet ./...
test -z "$(gofmt -l .)"
go test ./... -race -count=1

# 3. Pre-pull the images so the first test run is not also a ~1GB download
docker pull timescale/timescaledb-ha:pg15-latest
docker pull redis:7-alpine
```

Expected: step 1 prints a version; steps 2 all exit 0 with `ok` lines for `handlers`, `hub`,
`services`, `geo`; step 3 ends `Status: Downloaded newer image` or `Image is up to date`.
On arm64 the Postgres pull also prints a platform-mismatch warning — that is expected, not an
error.

### 9.2 Verification

```bash
cd backend
go build ./...                                   # exit 0
go vet -tags=integration ./...                   # exit 0 — tagged file must vet too
test -z "$(gofmt -l .)"                          # exit 0
go test ./... -race -count=1                     # exit 0, unchanged, no Docker needed
go test -tags=integration ./test/... -race -count=1 -v
grep -rn 'time.Sleep' test/ internal/            # must return nothing
bash ../scripts/gate.sh --check=layering          # exit 0
```

Expected output of the integration run:

```
=== RUN   TestIntegration_LocationIngestToWebSocketBroadcast
    (testcontainers + migration log lines)
    Applied migration: 001_create_tables.sql
    Applied migration: 002_add_indexes.sql
    Applied migration: 003_add_h3_postgis.sql
    Applied migration: 004_seed_data.sql
--- PASS: TestIntegration_LocationIngestToWebSocketBroadcast (XXs)
PASS
ok  	transit-backend/test	XXs
```

`go vet -tags=integration ./...` is listed separately on purpose: without the tag, vet never
type-checks the new file, and a broken tagged file can sit undetected indefinitely.

### 9.3 How long this should take — do not treat a slow run as a hang

| Phase | amd64 host | **arm64 host (this machine, emulated)** |
|---|---|---|
| Image pull (cold, one time) | 60–180s | 60–180s (network-bound, not emulated) |
| Postgres container ready | 5–15s | **20–60s** |
| Redis container ready | 1–3s | 1–3s |
| Migrations 001–004 | 1–3s | 3–10s |
| The actual assertions | < 1s | < 1s |
| Teardown | 1–3s | 2–5s |
| **Total, warm images** | **~10–25s** | **~30–80s** |

**A 60-second run on this machine is normal, not a hang.** The first run after a cold pull
can exceed three minutes. Only suspect a hang past ~5 minutes, at which point the likely
cause is wait strategy #1 (occurrence 1 instead of 2) or #4 (missing registration wait).

Set `-timeout 300s` explicitly if the default ever becomes a factor; the Go default is 10m,
which is above the worst realistic case here.

---

## 10. Must not be touched

| Thing | Why |
|---|---|
| `hub.Run`'s `ctx.Done()` branch, `drainAndClose`, `clientDrainTimeout`, `clientDrainPollInterval` | DEFECT-6 / D28. The teardown in §6.1 is designed to work *with* the drain, not to require changing it. Adding `ClientCount()` (§3.4) touches none of this. |
| `hub.Message` — field set, JSON tags, absence of `omitempty` | DEFECT-5 and DEFECT-2. The test asserts this shape; it must never be edited to make an assertion pass. |
| `MsgTypeLocationUpdate = "LOCATION_UPDATE"` | DEFECT-2. Uppercase. |
| `backend/internal/hub/message_test.go` | Pins the serialized shape including `heading`'s absence. |
| `main.go`'s shutdown ordering — `e.Shutdown(ctx)` → `hubCancel()` → `wsHub.Shutdown()` | Phase 1A. The test *mirrors* this order; it does not modify `main.go` at all. `main.go` needs zero changes for this phase. |
| Existing migrations 001–004 | `CLAUDE.md` §6.3: never edit an existing migration. The test applies them as they are. If the test needs schema, it seeds rows at runtime (§5 step 8), not via a new migration. |
| `infra/docker-compose.yml` | The test reads its image tags; it does not change them. D32 is recorded, not actioned. |
| Every existing test | The default `go test ./...` path must be byte-identical in behavior and runtime. |

---

## 11. Definition of done

1. Human approval for testcontainers recorded (§0). **Nothing before this.**
2. `go.mod` / `go.sum` updated via `go get` + `go mod tidy`; the three modules are the only
   new *direct* dependencies.
3. `backend/test/integration_test.go` created: `//go:build integration`, `package
   integration`, one test function, no `time.Sleep`.
4. `Hub.ClientCount()` added to `backend/internal/hub/hub.go` — five lines, read-only, nothing
   else in that file changed (§3.4).
5. §9.2 verification all green, including the untagged `go test ./... -race -count=1` being
   unchanged.
6. D32 and D33 appended to `IDEAS.md` (§8.3).
7. One commit, conventional format, e.g.
   `test(integration): end-to-end POST /api/location to WebSocket broadcast`.
   The commit message must state plainly that the suite is tag-gated and therefore that
   `database`/`cache` coverage appears only under `-tags=integration`.
