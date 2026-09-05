# Transit

**Real-time geo-telemetry ingestion and broadcast engine — the vehicle-location and
live-fan-out core of a Fleet-Engine-style backend, at demo scale, without trip lifecycle
management, routing-grade ETAs, or client SDKs.**

Go backend (Echo, `pgx/v5`, PostgreSQL + TimescaleDB + PostGIS, Redis, WebSocket hub) and a
React 19 SPA (Zustand, Vite, Leaflet).

| | |
|---|---|
| **Design** | [`ARCHITECTURE.md`](ARCHITECTURE.md) — data flow, components, spatial model, concurrency |
| **Self-audit** | [`AUDIT.md`](AUDIT.md) — the defect record, with evidence |
| **Working contract** | [`CLAUDE.md`](CLAUDE.md) — conventions, anti-goals, session protocol |
| **Open findings** | [`IDEAS.md`](IDEAS.md) — deferred work, each with its reason |

---

## What it does

It receives a stream of GPS coordinates from moving things — each report is a **ping** —
remembers where each one is, works out when one is near a place someone cares about, and
pushes live position updates to a map in a browser as they happen.

There is a demo web page on top that draws bus icons on a city map, because "buses on a map"
is easy to picture. Nothing about the underlying system is specific to buses: the database
calls the moving things *devices*, the places they are tracked against *zones*, and a
scheduled run between them a *trip*. Swap the seed data and the map skin and the same server
would track delivery scooters, forklifts, or hikers' phones.

**The problem it solves.** Two things are hard to do well at once when tracking moving objects
in real time: keeping a permanent, queryable record of every position report, and telling
everyone watching *right now* that something moved, within a fraction of a second, without
them polling for it. The first wants a database built for large volumes of timestamped rows;
the second wants an open connection the server can push into. This system does both on every
ping — one durable write to a time-series table, one cheaper write to an in-memory cache, and
one copy out over a WebSocket to every browser currently watching.

**What goes in.** `POST /api/location` carrying device id, latitude, longitude, speed, and
claimed GPS accuracy. Bounds are checked before the ping is accepted.

**What comes out.** A live WebSocket stream (`GET /ws`) that moves the dot without a page
reload; a read API for the current picture (`/api/nearby/devices`, `/api/nearby/stops`,
`/api/arrivals`, plus route and zone lookups); and a durable history table that makes "was
this device near here in the last five minutes" an indexed question rather than a full scan.

---

## What it deliberately does not do

This is a tracking-and-broadcast engine, not a finished transit product.

- **No trip lifecycle.** A trip exists as a database row that seed data writes once. Nothing
  in the running server ever advances, starts, or ends one.
- **No traffic prediction or road-network routing.** The one arrival estimate the server
  computes is straight-line distance divided by speed.
- **No authentication or authorization.** No login, no API key, no per-device credential. CORS
  allows every origin and the WebSocket upgrade accepts any origin unconditionally — fine for
  a local demo, not something to expose on the open internet as-is.
- **No multi-tenancy.** No organization, account, or key scoping which devices belong to whom.

It is also not a consumer transit app, not a platform, and not a competitor to Uber, Google
Fleet Engine, or similar. The live map is the cheapest part of those systems; their value is
marketplace liquidity, dispatch, routing graphs, and mobile SDKs — none of which are in scope
here.

### Honest limitations

| Assumption | Status |
|---|---|
| Straight-line distance ≈ travel distance | **Open — blocked.** Route-projected distance needs a route polyline; the `routes` table has no geometry column. See `AUDIT.md` §8 |
| Proximity ⇒ approaching | **Open — blocked**, same root cause |
| Instantaneous speed predicts the next few minutes | **Accepted.** A POC-appropriate trade-off, documented rather than fixed |
| A stopped device is treated as moving at 20 km/h (`DefaultSpeed`) | **Accepted.** Same |

The two accepted limitations are deliberate. Stating them plainly is stronger than hiding
them.

---

## Measured performance

These numbers are measured, not claimed. Methodology and raw output are included, because a
benchmark without its scope is a marketing number.

**What it measures:** HTTP boundary overhead for `POST /api/location` using the Echo router
and `httptest` with a mocked `IngestService` — JSON parsing and type binding, coordinate and
speed validation, default timestamp injection, and response serialization.

**What it does NOT measure:** database persistence, the PostGIS geometry trigger, Redis
writes, WebSocket fan-out to connected clients, or network transport. End-to-end latency will
be established when real infrastructure is under test.

**Environment:** Go 1.26.0, `darwin/arm64`, Apple M5 Pro.

```
goos: darwin
goarch: arm64
pkg: transit-backend/internal/handlers
cpu: Apple M5 Pro
BenchmarkIngestLocation-15       1691887          2127 ns/op        8419 B/op         42 allocs/op
BenchmarkIngestLocation-15       1695256          2141 ns/op        8419 B/op         42 allocs/op
BenchmarkIngestLocation-15       1707676          2127 ns/op        8419 B/op         42 allocs/op
BenchmarkIngestLocation-15       1678935          2135 ns/op        8419 B/op         42 allocs/op
BenchmarkIngestLocation-15       1686612          2135 ns/op        8419 B/op         42 allocs/op
```

| Metric | Measured | Target | Status |
|---|---|---|---|
| Ingest handler latency | **~2.13 µs** | < 20 ms p99 (excluding client network) | Well within budget — leaves the rest for DB and Redis I/O |
| Ingest handler allocations | **8419 B/op** across **42 allocs/op** | — | Baseline: Echo context, request JSON parsing, response buffer |

### Test coverage

| Package | Coverage |
|---|---|
| `pkg/geo` | 100.0% |
| `internal/services` | 95.0% |
| `internal/hub` | 86.1% |
| `internal/handlers` | 12.7% |
| `internal/cache`, `config`, `database`, `middleware`, `cmd/server` | 0.0% |

`internal/hub` coverage is nondeterministic across runs; the figure above is the observed
minimum, which is the only safe number to gate on. `internal/database` and `internal/cache`
are exercised by the integration test, which runs under a build tag. Four packages remain at
zero — see `AUDIT.md` §6.

---

## Quickstart

### Prerequisites

- **Docker & Docker Compose** — database, cache, and backend services
- **Go 1.25+** — to build or test the backend outside Docker (`go.mod` requires `go 1.25.0`)
- **Node.js & npm** — frontend dev server and build
- **Make** — the command registry

### One step

```bash
chmod +x deploy.sh   # first time only
./deploy.sh
```

Verifies prerequisites, brings up the backend Docker stack via the Makefile, installs frontend
dependencies, and builds the React app. `./cleanup.sh` tears it all down, destroying volumes
and `node_modules`.

### Backend only

```bash
make check-deps      # verify dependencies
make docker-up       # Postgres, Redis, Go backend
make docker-health   # confirm all services healthy
```

| Command | What it does |
|---|---|
| `make help` | List all available targets |
| `make build` | Compile the Go binary |
| `make rebuild-backend` | Rebuild and restart only the backend, leaving the databases up |
| `make docker-down` | Stop containers, **preserve** volumes |
| `make docker-clean` | Stop containers and **destroy** volumes |
| `make docker-logs` | Tail logs for all services |
| `make db-migrate` / `make db-seed` | Run migrations / seed sample data |
| `make test-unit` | Go unit tests |
| `make test-integration` | Integration tests (requires Docker; uses the `integration` build tag) |

### Endpoints

| Service | Address |
|---|---|
| Backend API | `http://localhost:8080` |
| Health check | `http://localhost:8080/health` |
| PostgreSQL | `localhost:5432` (`transit_user` / `transit_password`) |
| Redis | `localhost:6379` |

### Frontend only

```bash
cd frontend
npm install
npm run dev      # http://localhost:5173, with HMR
npm run build    # production build into frontend/dist
npm run preview  # serve the production build locally
```

---

## Outbound network calls

Four external hosts are contacted, **none of them by the Go server** — all four are requests
the browser makes on its own:

- `router.project-osrm.org` — public OSRM demo, called only on the route-creator screen to
  snap operator-drawn waypoints onto roads; falls back to straight lines on failure
- `basemaps.cartocdn.com` — map tiles
- `fonts.googleapis.com` / `fonts.gstatic.com` — the page's one webfont

---

## Repository layout

```
backend/     Go module: server, packages, SQL migrations
frontend/    React + TypeScript single-page app, built by Vite
infra/       Docker Compose: Postgres, Redis, backend
docs/        Diagrams
Makefile     Docker lifecycle, DB tasks, tests, smoke checks
deploy.sh    Boots the whole stack
cleanup.sh   Tears it back down
```
