# Phase 1B Execution Plan

> **Purpose.** Self-contained instructions for a model with **no project context**,
> no CLAUDE.md auto-read, no subagents, and no history of this repository.
> Follow every step literally. When a step says "verify", run the command and
> confirm the output matches before continuing.

---

## 0. Orientation

### 0.1 What this repo is

A Go monolith (`backend/`) + React SPA (`frontend/`). You will work only in
`backend/`. The repo ingests GPS pings via HTTP, persists them to TimescaleDB,
caches hot state in Redis, evaluates H3 geofences, and fans out updates to
WebSocket clients.

### 0.2 Repository state at entry

```
backend/internal/hub        37.0% coverage (3 test files exist: hub_shutdown_test.go, hub_drop_test.go, message_test.go)
backend/internal/services    6.0% coverage (1 test file exists: geofencing_test.go)
backend/internal/handlers    0.0% coverage (no test files)
backend/pkg/geo            100.0% coverage
```

### 0.3 Layering rule (CLAUDE.md §3.2)

```
handlers → services → repositories → database
```

- handlers import only service **interfaces** (from `handlers/interfaces.go`)
- services import only repository **interfaces** (from `services/interfaces.go`)
- **No** handler or service imports `internal/database` directly
- `pkg/geo` imports nothing internal

### 0.4 Test rules (CLAUDE.md §8.3)

- Table-driven tests for pure functions.
- **No `time.Sleep` in tests.** Use channels, `sync.WaitGroup`, or `context.WithTimeout`.
- Tests must not depend on execution order or shared global state.

### 0.5 Entry check — run before doing anything

```bash
cd backend && go test ./... -cover
bash scripts/gate.sh --check=layering
```

Expected coverage:
- `internal/hub` → 37.0%
- `internal/services` → 6.0%
- `internal/handlers` → 0.0%
- Layering gate → `✓ Layering rule passed`

If these numbers differ, STOP. The repo has changed since this plan was written.

---

## 1. Work Item 2.1 — Hub concurrency tests (6h)

**Target:** `internal/hub` from 37.0% → ≥50%

**File to create:** `backend/internal/hub/hub_concurrency_test.go`

### 1.1 Source files you need to understand

Read these files completely before writing any test:

#### `backend/internal/hub/hub.go` (74 lines)

```go
package hub

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
)

// DroppedMessages tracks how many messages were dropped due to full client buffers.
var DroppedMessages atomic.Int64

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
			for client := range h.Clients {
				close(client.Send)
				delete(h.Clients, client)
			}
			h.mu.Unlock()
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
```

#### `backend/internal/hub/client.go` — Client struct

```go
type Client struct {
	Hub  *Hub
	Conn *websocket.Conn
	Send chan interface{}
}
```

ReadPump/WritePump require a real websocket.Conn and are not testable without
httptest/websocket setup. Do NOT test them here.

#### `backend/internal/hub/hub_shutdown_test.go` (114 lines) — existing test

This file already contains `waitForClients(t, h, want)`, a deterministic
poller bounded by 2s timeout. **Reuse this helper for all new tests.** It is
in `package hub` so it's directly callable.

```go
func waitForClients(t *testing.T, h *Hub, want int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		h.mu.RLock()
		n := len(h.Clients)
		h.mu.RUnlock()
		if n == want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for %d clients, got %d", want, n)
		default:
		}
	}
}
```

### 1.2 What to test

Write `backend/internal/hub/hub_concurrency_test.go` with these tests. All
tests are `package hub` (same package).

#### Test 1: `TestBroadcastToMultipleClients`

Purpose: verify that a message sent to `Broadcast` is delivered to **all**
registered clients.

```
1. h := New(); ctx, cancel := WithCancel(Background())
2. go h.Run(ctx); defer func() { cancel(); h.Shutdown() }()
3. Create 5 clients with Send: make(chan interface{}, 16)
4. Register all 5 via h.Register <- c
5. waitForClients(t, h, 5)
6. h.Broadcast <- "hello"
7. For each client, read from c.Send with a 2s timeout (select + time.After).
   Assert the received value is "hello".
8. cancel(); h.Shutdown()
```

#### Test 2: `TestUnregisterRemovesClient`

Purpose: verify that unregistering a client removes it from the map and closes
its Send channel.

```
1. h := New(); ctx, cancel := WithCancel(Background())
2. go h.Run(ctx); defer func() { cancel(); h.Shutdown() }()
3. Create 2 clients, register both, waitForClients(t, h, 2)
4. h.Unregister <- clients[0]
5. waitForClients(t, h, 1)
6. Assert clients[0].Send is closed (receive returns ok==false)
7. Assert clients[1] is still registered (waitForClients already confirmed)
```

#### Test 3: `TestUnregisterIdempotent`

Purpose: double-unregister must not panic.

```
1. h := New(); ctx, cancel := WithCancel(Background())
2. go h.Run(ctx); defer ...
3. Create 1 client, register, waitForClients(t, h, 1)
4. h.Unregister <- client
5. waitForClients(t, h, 0)
6. h.Unregister <- client  // again — must not panic or block
7. Wait with select + time.After(500ms) to confirm the hub processes
   the second unregister without blocking. The test completing is the assertion.
```

#### Test 4: `TestBroadcastDuringUnregister`

Purpose: concurrent broadcast and unregister must not race.

```
1. h := New(); ctx, cancel := WithCancel(Background())
2. go h.Run(ctx); defer ...
3. Create 10 clients, register all, waitForClients(t, h, 10)
4. var wg sync.WaitGroup
5. wg.Add(2)
6. Go: send 100 broadcasts: for i := 0; i < 100; i++ { h.Broadcast <- i }; wg.Done()
7. Go: unregister clients[5..9] one per iteration; wg.Done()
8. wg.Wait()
9. cancel(); h.Shutdown()
10. The test completing without panic is the assertion.
    This is a race-detector test — run with -race.
```

#### Test 5: `TestSlowConsumerDropsMessages`

Purpose: exercise the `default:` drop path with explicit counter assertion.

```
1. h := New(); ctx, cancel := WithCancel(Background())
2. go h.Run(ctx); defer ...
3. DroppedMessages.Store(0)
4. Create 1 "slow" client with Send: make(chan interface{}, 1) and
   1 "fast" client with Send: make(chan interface{}, 256)
5. Register both, waitForClients(t, h, 2)
6. Start a goroutine draining fast client's Send channel
7. Broadcast 10 messages without reading from slow client
8. Poll DroppedMessages.Load() >= 1 with 2s timeout
9. Assert at least 1 message was dropped
```

### 1.3 Fix `hub_drop_test.go` time.Sleep

While in this work item, also fix the three `time.Sleep` calls in
`backend/internal/hub/hub_drop_test.go`.

**Current code (lines 21, 28, 32):**
```go
h.Register <- slowClient
time.Sleep(10 * time.Millisecond)
// ...
h.Broadcast <- "msg1"
time.Sleep(10 * time.Millisecond)
// ...
h.Broadcast <- "msg2"
time.Sleep(10 * time.Millisecond)
```

**Replacement:**
- After `h.Register <- slowClient`: call `waitForClients(t, h, 1)`
- After `h.Broadcast <- "msg1"`: poll `len(slowClient.Send) == 1` with 2s
  timeout to confirm the message was delivered
- After `h.Broadcast <- "msg2"`: poll `DroppedMessages.Load() >= 1` with 2s
  timeout to confirm the drop happened

### 1.4 Verification

```bash
cd backend

# No time.Sleep in hub tests
grep -rn 'time.Sleep' internal/hub/
# Expected: no output

# Tests pass with race detector, 5 iterations
go test -race -count=5 ./internal/hub/...
# Expected: ok, no failures

# Coverage meets target
go test -cover ./internal/hub/...
# Expected: >= 50%
```

### 1.5 Commit

```
test(hub): add concurrency tests and remove remaining time.Sleep

Cover broadcast fan-out, unregister, double-unregister, concurrent
broadcast+unregister, and slow-consumer drop path. Remove all
time.Sleep from hub tests per CLAUDE.md §8.3.
```

### 1.6 What NOT to touch

- Do not modify `hub.go`, `client.go`, or `message.go`.
- Do not modify `hub_shutdown_test.go` or `message_test.go`.
- Do not add a `DroppedMessages` accessor — the field is a package-level
  `atomic.Int64` and tests are in the same package.

---

## 2. Work Item 2.2 — Services tests (5h)

**Target:** `internal/services` from 6.0% → ≥30%

**Files to create:**
- `backend/internal/services/ingest_test.go`
- `backend/internal/services/geofencing_method_test.go`
- `backend/internal/services/arrivals_test.go`

### 2.1 Source files you need to understand

Read completely before writing any test:

#### `backend/internal/services/interfaces.go` (50 lines)

```go
package services

import (
	"context"
	"transit-backend/internal/models"
)

type StopRepository interface {
	GetByID(ctx context.Context, stopID string) (*models.Stop, error)
	GetByRouteID(ctx context.Context, routeID string) ([]models.Stop, error)
	GetNearby(ctx context.Context, lat, lng float64, radiusMeters int) ([]models.Stop, error)
}

type TripRepository interface {
	GetActiveTripsBeforeStop(ctx context.Context, stopSequence int) ([]models.TripWithLocation, error)
}

type LocationRepository interface {
	Insert(ctx context.Context, loc *models.Location) error
	GetLatestLocation(ctx context.Context, deviceID string) (*models.Location, error)
	GetBusesInHexes(ctx context.Context, hexes []string, maxAgeMinutes int) ([]models.NearbyBus, error)
	GetBusesNearStop(ctx context.Context, lat, lng float64, radiusMeters int, maxAgeMinutes int) ([]models.NearbyBus, error)
}

type RouteRepository interface {
	GetAll(ctx context.Context) ([]models.Route, error)
	Create(ctx context.Context, req models.CreateRouteRequest) (*models.CreateRouteResponse, error)
}

type DeviceCache interface {
	SetDeviceState(ctx context.Context, deviceID string, state map[string]interface{}) error
}

type DeviceRouteRepository interface {
	GetActiveRouteID(ctx context.Context, deviceID string) (string, error)
}
```

#### `backend/internal/services/ingest.go` (82 lines) — see §1.1 above

#### `backend/internal/services/geofencing.go` (236 lines) — key methods:

- `CalculateHex(lat, lng)` → string (lines 37-44)
- `IsAtStop(busLat, busLng, stopLat, stopLng)` → bool (lines 61-65)
- `IsAtStopWithHysteresis(busLat, busLng, stopLat, stopLng, wasAtStop)` → bool (lines 73-106)
- `GetNeighborHexes(lat, lng, k)` → []string (lines 109-127)
- `DetectArrival(oldLoc, newLoc, stop)` → bool (lines 178-187)
- `DetectDeparture(oldLoc, newLoc, stop)` → bool (lines 193-201)

#### `backend/internal/services/arrivals.go` (172 lines) — key methods:

- `isApproaching(busLat, busLng, stopLat, stopLng)` → bool (lines 115-131) — **unexported**
- `CalculateETAWithTraffic(fromLat, fromLng, toLat, toLng, speed)` → int (lines 138-149)

#### Key models

```go
// models/location.go
type Location struct {
	DeviceID  string  `json:"device_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
	Accuracy  float64 `json:"accuracy"`
	Timestamp int64   `json:"timestamp"`
	HexRes9   string  `json:"hex_res9,omitempty"`
}

// models/stop.go
type Stop struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Sequence  int     `json:"sequence"`
}
```

### 2.2 Mocking strategy

The services tests are `package services` (same package). Define mock structs
at the top of each test file using function fields:

```go
type mockLocationRepo struct {
	insertFn           func(ctx context.Context, loc *models.Location) error
	getLatestFn        func(ctx context.Context, deviceID string) (*models.Location, error)
	getBusesInHexesFn  func(ctx context.Context, hexes []string, maxAge int) ([]models.NearbyBus, error)
	getBusesNearStopFn func(ctx context.Context, lat, lng float64, radius int, maxAge int) ([]models.NearbyBus, error)
}

func (m *mockLocationRepo) Insert(ctx context.Context, loc *models.Location) error {
	return m.insertFn(ctx, loc)
}
// ... implement all interface methods
```

Use the same pattern for `mockDeviceRouteRepo`, `mockDeviceCache`.

### 2.3 Test file 1: `ingest_test.go`

#### Test 1: `TestIngestLocation_HappyPath`

```
1. Mocks: locRepo.Insert returns nil, routeRepo returns ("route-1", nil),
   cache.SetDeviceState returns nil
2. Create a real hub.New(), hub ctx, go h.Run(ctx), register 1 test client
   with Send: make(chan interface{}, 16)
3. svc := NewIngestService(locRepo, routeRepo, cache, h)
4. loc := &models.Location{DeviceID: "dev-1", Latitude: 28.6, Longitude: 77.2,
   Speed: 30.0, Timestamp: 1000}
5. err := svc.IngestLocation(context.Background(), loc)
6. Assert err == nil
7. Assert locRepo was called (use a bool flag)
8. Read from test client's Send with 2s select timeout
9. Assert message is hub.Message with Type=="LOCATION_UPDATE", DeviceID=="dev-1",
   RouteID=="route-1"
10. cancel(); h.Shutdown()
```

#### Test 2: `TestIngestLocation_InsertError`

locRepo.Insert returns error → IngestLocation returns error wrapping "insert
location". Route and cache must NOT be called.

#### Test 3: `TestIngestLocation_RouteResolveError_NotFatal`

routeRepo returns error → IngestLocation returns nil. Broadcast message has
RouteID == "".

#### Test 4: `TestIngestLocation_CacheError_NotFatal`

cache.SetDeviceState returns error → IngestLocation returns nil.

### 2.4 Test file 2: `geofencing_method_test.go`

These test **method-based behavior** on `GeofencingService`. The existing
`geofencing_test.go` tests only package-level pure functions. **Do not modify
`geofencing_test.go`.**

Since `CalculateHex`, `IsAtStop`, `IsAtStopWithHysteresis`, `GetNeighborHexes`,
`DetectArrival`, `DetectDeparture` are pure H3 functions that do NOT touch
repositories, construct the service with nil repos:

```go
svc := NewGeofencingServiceWithResolution(nil, nil, 9)
```

#### Tests to write (table-driven where possible):

1. `TestCalculateHex_ValidCoordinates` — table: Delhi, NYC, Origin → non-empty
2. `TestIsAtStop_SameLocation` — same coords → true
3. `TestIsAtStop_FarApart` — ~1.4km apart → false
4. `TestIsAtStopWithHysteresis_SameHex` — same coords, wasAtStop=false → true
5. `TestIsAtStopWithHysteresis_NotPreviouslyAtStop_NotInHex` — far, wasAtStop=false → false
6. `TestGetNeighborHexes_K0` — len == 1
7. `TestGetNeighborHexes_K1` — len == 7
8. `TestDetectArrival` — old far, new at stop → true
9. `TestDetectArrival_AlreadyAtStop` — both at stop → false
10. `TestDetectDeparture` — old at stop, new far → true
11. `TestDetectArrival_NilInputs` — nil old/new/stop → false

### 2.5 Test file 3: `arrivals_test.go`

#### Test 1: `TestIsApproaching_InKRing`

`isApproaching` is unexported — test must be `package services`.

```go
geoSvc := NewGeofencingServiceWithResolution(nil, nil, 9)
svc := &ArrivalsService{geoService: geoSvc}
// Bus at stop location → in k=3 ring → true
result := svc.isApproaching(28.6139, 77.2090, 28.6139, 77.2090)
assert true
```

#### Test 2: `TestIsApproaching_FarAway`

```go
// Bus 5km away → outside k=3 ring (~500m) → false
result := svc.isApproaching(28.6500, 77.2500, 28.6139, 77.2090)
assert false
```

#### Test 3: `TestCalculateETAWithTraffic`

```go
import "transit-backend/pkg/geo"

baseETA := geo.CalculateETA(28.5, 77.0, 28.6, 77.1, 60.0)
expected := int(float64(baseETA) * 1.2)
eta := svc.CalculateETAWithTraffic(28.5, 77.0, 28.6, 77.1, 60.0)
assert eta == expected
```

### 2.6 Verification

```bash
cd backend

go test -race -count=3 ./internal/services/...
# Expected: ok

go test -cover ./internal/services/...
# Expected: >= 30%
```

### 2.7 Commit

```
test(services): add ingest, geofencing method, and arrivals tests

Mock-based ingest tests cover happy path, DB error, route-resolve
error, and cache error. Geofencing method tests cover CalculateHex,
IsAtStop, hysteresis, k-ring neighbors, DetectArrival, DetectDeparture,
and nil-input guards. Arrivals tests cover isApproaching k-ring logic
and CalculateETAWithTraffic.
```

### 2.8 What NOT to touch

- Do not modify any production code in `services/`.
- Do not modify the existing `geofencing_test.go`.
- Do not import `internal/database` in any test file.

---

## 3. Work Item 2.3 — Ingest handler test (3h)

**Target:** `internal/handlers` from 0.0% → >0%

**File to create:** `backend/internal/handlers/location_test.go`

### 3.1 Source files

#### `backend/internal/handlers/location.go` (62 lines)

```go
package handlers

import (
	"net/http"
	"time"
	"transit-backend/internal/models"
	"github.com/labstack/echo/v4"
)

type LocationHandler struct {
	service IngestService
}

func NewLocationHandler(service IngestService) *LocationHandler {
	return &LocationHandler{service: service}
}

func (h *LocationHandler) IngestLocation(c echo.Context) error {
	var loc models.Location
	if err := c.Bind(&loc); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if loc.DeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "device_id is required"})
	}
	if loc.Latitude < -90 || loc.Latitude > 90 || loc.Longitude < -180 || loc.Longitude > 180 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid coordinates"})
	}
	if loc.Speed < 0 || loc.Speed > 350 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid speed"})
	}
	if loc.Timestamp == 0 {
		loc.Timestamp = time.Now().Unix()
	}
	if err := h.service.IngestLocation(c.Request().Context(), &loc); err != nil {
		c.Logger().Error(err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"hex_res9": loc.HexRes9,
	})
}
```

#### `backend/internal/handlers/interfaces.go` — IngestService interface:

```go
type IngestService interface {
	IngestLocation(ctx context.Context, loc *models.Location) error
}
```

### 3.2 Mock

```go
type mockIngestService struct {
	ingestFn  func(ctx context.Context, loc *models.Location) error
	callCount int
	lastLoc   *models.Location
}

func (m *mockIngestService) IngestLocation(ctx context.Context, loc *models.Location) error {
	m.callCount++
	m.lastLoc = loc
	if m.ingestFn != nil {
		return m.ingestFn(ctx, loc)
	}
	return nil
}
```

### 3.3 Tests to write

Use `httptest` + Echo:

```go
e := echo.New()
req := httptest.NewRequest(http.MethodPost, "/api/location", strings.NewReader(body))
req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
rec := httptest.NewRecorder()
c := e.NewContext(req, rec)
err := handler.IngestLocation(c)
```

1. `TestIngestLocation_HappyPath` — valid body → 200, service called once
2. `TestIngestLocation_MissingDeviceID` — no device_id → 400
3. `TestIngestLocation_InvalidCoordinates` — lat=91 → 400
4. `TestIngestLocation_InvalidSpeed` — speed=400 → 400
5. `TestIngestLocation_NegativeSpeed` — speed=-1 → 400
6. `TestIngestLocation_ServiceError` — service returns error → 500
7. `TestIngestLocation_DefaultTimestamp` — timestamp=0 → handler sets it > 0
8. `TestIngestLocation_InvalidJSON` — malformed body → 400

### 3.4 Verification

```bash
cd backend

go test -race -count=3 ./internal/handlers/...
# Expected: ok

go test -cover ./internal/handlers/...
# Expected: > 0% (should be ~40-60% since location.go is the largest handler)
```

### 3.5 Commit

```
test(handlers): add location handler tests with mock service

Eight tests covering happy path, validation (missing device_id,
invalid coordinates, invalid speed, bad JSON), service error,
and default timestamp injection. Uses httptest + Echo test context.
```

### 3.6 What NOT to touch

- Do not modify any production code in `handlers/`.
- Do not test other handlers — only `location.go` is in scope.
- Do not import `internal/database` or `internal/services`.

---

## 4. Work Item 2.4 — BenchmarkIngestLocation (3h)

**Target:** Produce a baseline benchmark using `testing.B`, record results in
`RESULTS.md`.

**Files to create:**
- `backend/internal/services/ingest_bench_test.go`
- `RESULTS.md` (repo root)

### 4.1 Benchmark file

`backend/internal/services/ingest_bench_test.go`:

```go
package services

import (
	"context"
	"testing"

	"transit-backend/internal/hub"
	"transit-backend/internal/models"
)

// No-op mocks for benchmarking
type benchLocationRepo struct{}
func (r *benchLocationRepo) Insert(_ context.Context, _ *models.Location) error { return nil }
func (r *benchLocationRepo) GetLatestLocation(_ context.Context, _ string) (*models.Location, error) { return nil, nil }
func (r *benchLocationRepo) GetBusesInHexes(_ context.Context, _ []string, _ int) ([]models.NearbyBus, error) { return nil, nil }
func (r *benchLocationRepo) GetBusesNearStop(_ context.Context, _, _ float64, _, _ int) ([]models.NearbyBus, error) { return nil, nil }

type benchRouteRepo struct{}
func (r *benchRouteRepo) GetActiveRouteID(_ context.Context, _ string) (string, error) { return "route-bench", nil }

type benchCache struct{}
func (c *benchCache) SetDeviceState(_ context.Context, _ string, _ map[string]interface{}) error { return nil }

func BenchmarkIngestLocation(b *testing.B) {
	h := hub.New()
	ctx, cancel := context.WithCancel(context.Background())
	go h.Run(ctx)
	defer func() { cancel(); h.Shutdown() }()

	// Drain client to prevent backpressure
	drainClient := &hub.Client{Hub: h, Send: make(chan interface{}, 256)}
	h.Register <- drainClient
	go func() { for range drainClient.Send {} }()

	svc := NewIngestService(&benchLocationRepo{}, &benchRouteRepo{}, &benchCache{}, h)
	loc := &models.Location{
		DeviceID: "bench-dev", Latitude: 28.6139, Longitude: 77.2090,
		Speed: 30.0, Accuracy: 5.0, Timestamp: 1000, HexRes9: "891f5a44a4bffff",
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := svc.IngestLocation(ctx, loc); err != nil {
			b.Fatal(err)
		}
	}
}
```

### 4.2 Run and capture

```bash
cd backend
go test -bench=BenchmarkIngestLocation -benchmem -count=5 -run='^$' ./internal/services/... | tee /tmp/bench.txt
```

### 4.3 Create RESULTS.md

At the repo root, create `RESULTS.md`:

```markdown
# RESULTS.md — Measured performance

> Measured, not claimed. Raw output included.
> See CLAUDE.md §9 for the targets these track toward.

## Benchmark: IngestService.IngestLocation (Phase 1B baseline)

**What it measures:** Service-layer overhead per location ingest call with
mocked I/O (no database, no Redis, no network). Isolates struct allocation,
map construction, hub broadcast, and channel send.

**What it does NOT measure:** Database insert, Redis write, network I/O, or
HTTP handler overhead.

**Environment:**
- Go version: (fill from `go version`)
- OS/Arch: (fill)
- CPU: (fill)

**Results (5 runs):**
```
(paste raw output)
```

**Interpretation:**
- ns/op: service-layer latency per ingest call
- B/op: heap bytes per call
- allocs/op: heap allocations per call
```

### 4.4 Verification

```bash
cd backend
go test -bench=BenchmarkIngestLocation -benchmem -count=1 -run='^$' ./internal/services/...
# Must complete without error

go test -race ./...
# Full suite still passes
```

### 4.5 Commit

```
perf(services): add BenchmarkIngestLocation and create RESULTS.md

Stdlib testing.B benchmark with mocked I/O. 5-run results recorded in
RESULTS.md as the Phase 1B baseline.
```

### 4.6 What NOT to touch

- No k6, Prometheus, or external benchmarking tools.
- No benchmarks against real databases or Redis.
- No production code modifications.

---

## 5. Phase exit block

After all four commits, run:

```bash
cd backend

# 1. Full test suite with race detector
go test -race ./...

# 2. Coverage per package
go test -cover ./internal/hub/...       # >= 50%
go test -cover ./internal/services/...  # >= 30%
go test -cover ./internal/handlers/...  # > 0%

# 3. Layering gate
bash scripts/gate.sh --check=layering   # Must pass

# 4. No time.Sleep in hub tests
grep -rn 'time.Sleep' internal/hub/     # Must return nothing

# 5. Benchmark runs
go test -bench=BenchmarkIngestLocation -benchmem -count=1 -run='^$' ./internal/services/...

# 6. RESULTS.md exists
test -f ../RESULTS.md && echo "OK" || echo "MISSING"
```

All six checks must pass.

---

## 6. DroppedMessages design decision

`hub.DroppedMessages` is a package-level `atomic.Int64` (`hub/hub.go:11`).
Tests read it directly because they're in `package hub`.

**Decision for Phase 1B:** Keep as-is. Do NOT add an exported accessor.
If a future `/metrics` endpoint needs it, that will live in `handlers/` and
will add an accessor on `Hub` — but that's not this phase.

---

## 7. Commit order

```
1. test(hub): add concurrency tests and remove remaining time.Sleep
2. test(services): add ingest, geofencing method, and arrivals tests
3. test(handlers): add location handler tests with mock service
4. perf(services): add BenchmarkIngestLocation and create RESULTS.md
```

Each commit is independently green. No commit depends on another.
