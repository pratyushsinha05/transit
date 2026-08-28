# CLAUDE.md — Transit

> **Read this file completely before touching any code. Every session. No exceptions.**
>
> This file is the contract. If the code contradicts this file, the code is wrong.
> If this file contradicts reality, **stop and fix this file first**, then proceed.

---

## 0. Read this first: what changed and why

An earlier version of this file specified a **different project**. It said:

- No database until persistence is concretely needed
- No HTTP router libraries
- No frontend framework
- No Leaflet abstraction layers

**Every one of those constraints is now void.** The repo runs PostgreSQL + TimescaleDB +
PostGIS + Redis, uses the Echo HTTP router, and ships a React 19 SPA with Zustand and
Tailwind. The old constraints were not followed, were never formally revoked, and left the
file describing a project that does not exist.

This is the **first of two documented drifts** in this repo. The second is architectural
(see §4). Both happened for the same reason: a stated rule was silently overridden by
expedience and nothing forced a check back.

**This file is the forcing function.** It is version-controlled, read every session, and
updated *before* the code, not after.

---

## 1. Mission

### 1.1 What this repo is

A **real-time geo-telemetry ingestion and broadcast engine**.

It ingests high-frequency GPS pings from tracked devices, persists them to a time-series
store, holds hot state in memory, evaluates geofences, and fans out live updates to browser
clients over WebSockets.

The bus/transit UI is a **demo skin over a generic engine**. It is not the product.

### 1.2 What this repo is NOT

- **Not a consumer transit app.** Gurugaman covers GMCBL, Chalo covers 50+ Indian cities,
  and Google Maps covers agencies that publish GTFS-realtime feeds. There is no consumer
  gap to fill and no route to real bus GPS data.
- **Not a platform or a product.** No tenants, no customers, no SLA, no on-call.
- **Not a competitor to Uber, Zomato, HyperTrack, or Google Fleet Engine.** The live map is
  the cheapest part of those systems. Their value is marketplace liquidity, dispatch,
  routing graphs, live traffic, and mobile SDKs — none of which are in scope here.
- **Not a research-paper showcase.** That is `llm-router`'s job. Duplicating its papers here
  adds **zero** new signal and makes both repos look padded.

### 1.3 Why it exists

This is a **portfolio artifact** for a Platform / Infrastructure Engineer targeting
infrastructure-heavy roles (Uber, DoorDash, Datadog, Cloudflare, GKE-shop platform teams,
finance-sector platform orgs).

The signal split is **70% SRE / infra, 30% SDE**:

- **The 70%** is how the workload is built, deployed, observed, and operated. That work is
  a later phase and is out of scope for now.
- **The 30%** — this phase — is proving the author writes real, well-structured,
  well-tested Go. This is what separates a platform engineer from someone who only writes
  YAML.

**Decision rule derived from this:** when a change improves *code quality, structure, or
testability*, it is in scope. When it improves *prediction accuracy or product features*,
it is almost certainly out of scope. The one deliberate exception is Phase 4.5 (§6.5).

### 1.4 The honest one-line description

> Real-time geo-telemetry ingestion and broadcast engine — the vehicle-location and
> live-fan-out core of a Fleet-Engine-style backend, at demo scale, without trip lifecycle
> management, routing-grade ETAs, or client SDKs.

Do not inflate this sentence. Every clause is defensible under interview questioning.
Anything stronger is not.

---

## 2. Current phase and phase gates

**Current phase: Phase 0 → Phase 4.5 (code cleanup). Infrastructure work is deferred.**

| Phase | Goal | Est. | Gate to pass |
|---|---|---|---|
| **0** | Ground truth | 2h | `v0-poc` tag exists; `BASELINE.md` records real coverage; `Device` collision resolved; this file is accurate |
| **1** | Kill 3 known defects | 1d | All three defects in §4 closed; tests green |
| **2** | Domain rename | 1d | No `Bus`/`Stop`/`Arrival` identifiers in `backend/internal/` or `backend/pkg/`; zero behavior change |
| **3** | Tests off zero | 2d | No package at 0% coverage; `internal/services` ≥ 60%; one integration test passes |
| **4** | Geofence events | 2d | Four event types emitted; `GeofenceService` is on the request path |
| **4.5** | Route-projected distance | 1d | ETA uses along-route distance; no ETA shown for a device past the zone |
| **5+** | Infrastructure | — | **Not started. Do not begin without explicit instruction.** |

**Gate protocol.** At the end of every phase:

1. Full test suite green
2. This file re-read and corrected if the phase changed any stated fact
3. One commit, real message, no "wip"
4. If the phase overran its estimate, **cut scope** rather than carrying half-finished work
   into the next phase

**Never work on two phases at once.** Drift is this repo's documented failure mode.

---

## 3. Actual architecture (as-built, verified)

### 3.1 Stack

| Layer | Technology | Notes |
|---|---|---|
| Language | Go | Monolith, `backend/cmd/server/main.go` is the composition root |
| HTTP | Echo | Router + middleware |
| DB driver | `pgx/v5` | Direct, no ORM |
| Persistence | PostgreSQL + TimescaleDB | `location_history` hypertable, compression policies enabled |
| Spatial | PostGIS | `ST_DWithin`, `ST_Distance`; geometry generated by insert trigger |
| Spatial index | Uber H3 | Resolution 9 (~175m edge), configurable via `H3_RESOLUTION` |
| Hot state | Redis | Latest known location per device |
| Realtime | Native Go channels + WebSocket hub | No message broker |
| Frontend | React 19 + TypeScript + Vite | SPA |
| FE state | Zustand | 5 slices |
| Styling | Tailwind | Dark "HUD" palette |
| Map | `react-leaflet` + CartoDB Dark Matter tiles | |
| Config | `godotenv` | Defaults in `infra/docker-compose.yml` |
| Local orchestration | Docker Compose | `transit-network` bridge |

> **Note:** a prior doc (`graphify.html`) claimed Redux Toolkit. That was false. The
> codebase uses **Zustand**. If any doc says otherwise, the doc is wrong.

### 3.2 Package layout and the layering rule

```
backend/cmd/server/main.go      Composition root. Wires everything. No business logic.
backend/internal/database/      Repositories. SQL only. No business rules.
backend/internal/services/      Business logic. Depends on repository interfaces.
backend/internal/handlers/      Echo HTTP controllers. Depends on service interfaces ONLY.
backend/internal/hub/           WebSocket hub. Channel-based broadcast.
backend/internal/cache/         Redis client behind an interface.
backend/internal/middleware/    CORS, errors, logging, recovery.
backend/internal/models/        Domain structs. No behavior beyond validation.
backend/internal/config/        Env loading. Every value has a documented default.
backend/pkg/geo/                Pure functions. Haversine, H3 helpers. No I/O, no state.
```

**THE LAYERING RULE — this is the one that was broken:**

```
handlers → services → repositories → database
```

- A handler **must never** import or call a repository.
- A handler **must** depend on a service interface, not a concrete type.
- A service **must** depend on a repository interface, not a concrete type.
- `backend/pkg/geo` depends on nothing internal. Pure functions only.

Any violation of this is a bug, not a shortcut, regardless of how much simpler the
shortcut looks.

### 3.3 Data flow (live pings)

1. Device `POST`s coordinates to `/api/location`
2. Handler writes to TimescaleDB (durable path)
3. Postgres trigger generates PostGIS geometry from lat/lng
4. Handler pushes update to Redis (hot path) and into the hub channel
5. Hub fans out JSON to every connected WebSocket client
6. Frontend `websocketClient.ts` → `messageHandler.ts` parses and dispatches to Zustand
7. `<BusMarkers />` (→ `<DeviceMarkers />` after Phase 2) moves the Leaflet marker

### 3.4 Core algorithms (as-built)

**Spatial indexing.** H3 resolution 9 for O(1) hex-equality geofencing; PostGIS as the
precise fallback because hex cells have hard edges and misjudge near-boundary cases.

**Distance.** `backend/pkg/geo/distance.go` → `Haversine(lat1, lng1, lat2, lng2)`,
great-circle distance on a sphere.

**ETA.**

```
d = Haversine(device, zone)
v = device.speed
if v < 1.0 km/h  →  v = 20.0        // DefaultSpeed fallback
ETA_minutes = round((d / v) * 60)
```

**Approach detection** (`backend/internal/services/arrivals.go`, currently orphaned):
k-ring radius 3 at resolution 9 = 37 cells ≈ 500m around the zone. If the device's hex is in
that set, `isApproaching = true`.

### 3.5 Known algorithmic limitations

| Assumption | Status |
|---|---|
| Straight-line distance ≈ travel distance | **FIXED in Phase 4.5** via `ST_LineLocatePoint` |
| Proximity ⇒ approaching | **FIXED in Phase 4.5** — along-route fraction gives direction |
| Instantaneous speed predicts the next few minutes | **Accepted limitation.** Document honestly in README. Do not fix. |
| Stopped ⇒ 20 km/h `DefaultSpeed` | **Accepted limitation.** Document honestly in README. Do not fix. |

The two accepted limitations are POC-appropriate trade-offs, not bugs. Stating them plainly
in the README is stronger than hiding them.

---

## 4. Known defects — fix these in Phase 1

All four were found by a self-directed adversarial repo audit and are **pre-diagnosed**.
This is execution, not investigation. Do not re-litigate the diagnosis.

### DEFECT-1: Handler bypasses service (architectural)

- **Where:** `backend/internal/handlers/arrivals.go` vs
  `backend/internal/services/arrivals.go`
- **What:** `ArrivalsService` is constructed in `backend/cmd/server/main.go` and then
  **ignored**. The handler reaches directly into repositories and recomputes raw math itself.
- **Consequence:** the k-ring approach-detection logic never executes. Dead code in the
  binary. The Clean Architecture the repo advertises is not the architecture it runs.
- **Fix:** handler takes a service interface in its constructor and calls it. No repository
  imports in `backend/internal/handlers/`. Add a compile-time interface assertion.
- **Verify:** `grep -rn "database\." backend/internal/handlers/` returns nothing.

### DEFECT-2: WebSocket contract mismatch

- **Where:** `backend/internal/hub/message.go` `Message` struct vs
  `frontend/src/services/websocket/messageHandler.ts` and `frontend/src/types/domain.ts`
- **What:** frontend expects `route_id` and `h3_hex` on `LOCATION_UPDATE` payloads. The
  backend struct drops both during serialization.
- **Fix:** backend adds both fields, per the canonical envelope in §7.2. `h3_hex` is useful
  to the client; `route_id` is needed for filtering.
- **Verify:** a test asserts the serialized JSON shape; the frontend type mirrors it.

### DEFECT-3: Ghost UI

- **Where:** `frontend/src/components/Sidebar/OperationsPanel.tsx` vs
  `frontend/src/components/Map/MapContainer.tsx`
- **What:** an "H3 Spatial Grid" toggle exists and toggles nothing. No rendering logic was
  ever written.
- **Fix:** **delete the toggle** and its `ui.ts` slice state. A control that does nothing is
  worse than no control. Do not implement the grid layer to justify the toggle — that is
  scope inflation. Check whether `frontend/src/utils/h3Helpers.ts` has other consumers
  before removing it.

### DEFECT-4: Test coverage

- **Current:** `backend/pkg/geo` ~100%; `backend/internal/services` ~6.9%;
  `backend/internal/handlers` 0%; `backend/internal/hub` 0%; no integration tests at all.
- **Fix:** Phase 3. Targets in §8.

---

## 5. Scope guards — what NOT to build

This repo has drifted twice. If a change requires one of these, stop and ask.

### 5.1 Algorithms — OUT

- No prediction-accuracy work beyond Phase 4.5: map-matching, Kalman filtering,
  particle filters, event-time processing.
- No distributed-systems papers. Those belong to `llm-router`.
- No NEW routing engine. Existing frontend OSRM client: do not extend, do not remove.

### 5.2 Product features — OUT

- Multi-tenancy, tenant isolation, API keys, quotas, rate limits
- Webhooks with retry semantics, versioned public API contracts, SLAs
- Trip / task / journey lifecycle state machines, waypoint ordering, shared pooling
- Mobile SDKs, battery-aware tracking frequency modulation
- Ticketing, payments, user accounts, notifications
- Historical analytics dashboards beyond what already exists

### 5.3 Infrastructure — OUT *for now*

Terraform, GKE, Istio, Helm, ArgoCD, Prometheus, Grafana, k6, chaos engineering.

All of this is the eventual 70% of the signal and it **is** coming. It is Phase 5+ and must
not start until Phases 0–4.5 have passed their gates. Infra on top of a broken monolith
proves nothing.

### 5.4 The general test

> Does this change make the **code** better structured, better tested, or more honest?
> → In scope.
>
> Does this change make the **product** smarter or more featureful?
> → Out of scope. Note it in `IDEAS.md` and move on.

### 5.5 Pre-existing features not covered by the audit

`models/trip.go`, `database/trips.go`, the route-creator UI
(`RouteCreatorPanel.tsx`, `RouteCreatorMarkers.tsx`, `api/createRoute.ts`), and
`frontend/src/services/api/osrm.ts` exist in the tree and appear in **no** audit document.

**Do not modify any of them in Phases 0–4.5.** Phase 0 documents whether they are live or
dead. They are scoped separately after that.

---

## 6. Domain rename (Phase 2)

> **BLOCKER: `backend/internal/models/device.go` already exists.** The `Bus` → `Device`
> mapping below may collide with it. Phase 0 resolves this. **Do not begin Phase 2 until
> the collision is resolved and this table is amended accordingly.**

The repo's identity is a generic telemetry engine with a transit demo skin. Core types must
be generic; only the demo layer may be transit-flavoured.

| Current | Rename to | Notes |
|---|---|---|
| `Bus` | `Device` | The thing that emits pings. Pending collision check above. |
| `BusLocation` | `LocationPing` | One GPS observation |
| `Stop` | `Zone` | A geofenced area of interest |
| `Arrival` | `GeofenceEvent` | An event, not a transit concept |
| `ArrivalsService` | `GeofenceService` | |
| `ArrivalHandler` | `GeofenceHandler` | |
| `BusMarkers.tsx` | `DeviceMarkers.tsx` | |
| `StopMarkers.tsx` | `ZoneMarkers.tsx` | |
| `ArrivalCard.tsx` | `GeofenceEventCard.tsx` | |
| `ArrivalsList.tsx` | `GeofenceEventList.tsx` | |
| `StopDetails.tsx` | `ZoneDetails.tsx` | |
| `busLocations.ts` (slice) | `devices.ts` | |
| `arrivals.ts` (slice) | `geofenceEvents.ts` | |
| `stops.ts` (slice) | `zones.ts` | |
| `Route` | `Route` | **Keep.** A polyline is a real, generic concept. |
| `location_history` | `location_history` | **Keep.** Already generic. |

**Rules for this phase:**

- Mechanical rename only. **Zero behavior change.** If a test's assertions change, something
  is wrong.
- Four commits: (1) Go models + database, (2) Go services + handlers, (3) SQL migration,
  (4) frontend. Verify the build between each.
- **Never edit an existing migration.** Add `migrations/005_rename_stops_zones.sql`.
- Use `gopls`-aware rename. **Never blind `sed -i`** across the tree — it hits strings,
  comments, and seed data.
- Transit vocabulary is permitted **only** in `migrations/004_seed_data.sql`, demo fixtures,
  and user-facing UI copy.

---

## 6.5 Phase 4.5 — Route-projected distance

The one deliberate exception to §5.1. Replaces straight-line ETA distance with along-route
distance using PostGIS already in the stack.

1. Project device position onto its route polyline: `ST_LineLocatePoint` → fraction 0–1
2. Project the zone onto the same polyline → fraction 0–1
3. Remaining distance = difference along the line, via `ST_LineSubstring` + `ST_Length`
4. If device fraction > zone fraction, the device has **passed**. Emit **no ETA**. Do not
   emit a large one.

**Rules:**

- No new dependencies. PostGIS is already running.
- Devices with no assigned route fall back to Haversine. Keep that path working and tested.
- Test the passed-the-zone case explicitly — it is the failure this phase exists to kill.
- The ETA **time model is unchanged**: still `distance / speed`. Only distance improves.
  Do not touch `DefaultSpeed` here.

---

## 7. Conventions

### 7.1 Go

- Errors: wrap with `fmt.Errorf("doing X: %w", err)`. Never discard. Never `panic` outside
  `main()`.
- Interfaces are declared by the **consumer**, not the producer. `internal/handlers` defines
  the service interface it needs; `internal/services` defines the repository interface it
  needs.
- Constructors take interfaces and return concrete types.
- Context: every function that touches I/O takes `ctx context.Context` first and honours
  cancellation.
- Goroutines: every goroutine has exactly one owner responsible for its shutdown. No
  fire-and-forget. The hub owns its own goroutines and drains cleanly on shutdown.
- Channels: the writer closes, never the reader. Buffered sizes are named constants with a
  comment justifying the number.
- No global mutable state. No `init()` for anything but constant registration.
- `pkg/geo` stays pure: no I/O, no logging, no clock reads, no state.
- Logging: structured, levelled by `LOG_LEVEL`. Never log inside a tight ingestion loop.

### 7.2 WebSocket envelope (canonical — resolves DEFECT-2)

```json
{
  "type": "LOCATION_UPDATE",
  "data": {
    "device_id": "string",
    "route_id":  "string",
    "lat":       0.0,
    "lng":       0.0,
    "speed":     0.0,
    "heading":   0.0,
    "h3_hex":    "string",
    "timestamp": "RFC3339"
  }
}
```

Any change to this shape requires updating, in the same commit: the Go struct, the
TypeScript type, the serialization test, and this section.

Phase 4 adds message type `GEOFENCE_EVENT`. Document its envelope here when it lands.

### 7.3 Frontend

- Strict TypeScript. No `any`. No non-null assertions without a comment.
- Zustand slices stay independent — no slice reads another slice's state directly.
- **No new UI control ships without working logic behind it.** This is the DEFECT-3 rule.
- Map rendering stays in `react-leaflet` components. No imperative Leaflet calls scattered
  through unrelated components.

### 7.4 Configuration

Every config value is declared in `infra/docker-compose.yml` with a default. Known toggles:
`DB_MAX_CONNS`, `DB_MIN_CONNS`, `H3_RESOLUTION`, `LOG_LEVEL`. Adding a config value without
a documented default is a bug.

---

## 8. Testing

### 8.1 Targets (Phase 3 exit criteria)

| Package | Current | Target |
|---|---|---|
| `backend/pkg/geo` | ~100% | Keep ≥ 95% |
| `backend/internal/services` | ~6.9% | ≥ 60% |
| `backend/internal/handlers` | 0% | ≥ 50% |
| `backend/internal/hub` | 0% | ≥ 50% |
| `backend/internal/database` | 0% | Covered by integration test |

**Hard rule: no package sits at 0%.**

### 8.2 What to test

- **Handlers:** `httptest` + mocked service interface. Every endpoint gets a happy path and
  at least one failure path (bad input, service error).
- **Services:** mocked repository interface. Cover the geofence logic DEFECT-1 was hiding —
  k-ring membership, hex boundary cases, the `DefaultSpeed` fallback branch.
- **Hub:** connect, broadcast to N clients, disconnect mid-broadcast, **slow consumer that
  does not read**. The slow-consumer case matters most — it is the real failure mode and it
  feeds the later infrastructure story.
- **Integration:** `backend/test/integration_test.go`, testcontainers with real Postgres
  (TimescaleDB + PostGIS) and Redis. One end-to-end path: `POST /api/location` → row in
  `location_history` → connected WebSocket client receives the update.

### 8.3 Rules

- Table-driven tests for pure functions.
- **No `time.Sleep` in tests.** Use channels, `sync.WaitGroup`, or `context.WithTimeout`.
- Tests must not depend on execution order or shared global state.
- A test skipped for more than one commit gets deleted.
- testcontainers is a new dependency — **ask before adding it.**

---

## 9. Performance targets

Not aspirational — these define what the eventual results table reports.

| Metric | Target |
|---|---|
| Ingestion throughput | ≥ 1,000 pings/sec sustained, single instance |
| Broadcast fan-out p99 | < 100ms at 1,000 concurrent WebSocket clients |
| Ingestion handler p99 | < 20ms (excluding client network) |
| Redis hot-read | < 5ms p99 |
| Memory | Stable under 1-hour sustained load — no growth trend |

Nothing here is claimed publicly until measured. Measured numbers go in `RESULTS.md`.

---

## 10. Operations (as-built)

- `./deploy.sh` — verifies prerequisites, `docker-compose` up (Postgres, Redis, Go backend),
  builds the Vite frontend, serves preview on port 4173
- `./cleanup.sh` — tears everything down, destroys volumes and `node_modules`
- `make build` — compile `backend/cmd/server/main.go`
- `make docker-up` — build and boot the backend stack
- `make rebuild-backend` — hot-swap the API container without restarting Postgres/Redis
- `make test-unit` — Go unit tests

Any new Make target must be documented here in the same commit.

---

## 11. Session protocol for Claude Code

**At the start of every session:**

1. Read this file completely.
2. State which phase (§2) you are in and which gate you are working toward.
3. Use **plan mode** before writing code. Show the plan. Wait for approval.

**During the session:**

4. One phase at a time. A problem belonging to a different phase goes in `IDEAS.md`.
5. If a change would violate §3.2 (layering) or §5 (scope), **stop and ask**. Do not
   implement the convenient version and flag it afterward — that is exactly how DEFECT-1
   happened.
6. If a fact in this file turns out to be wrong, fix this file **before** continuing.
7. Never `git push`, force-push, rewrite history, or `rm -rf`. Local commits only.

**At the end of the session:**

8. Run the full test suite.
9. Confirm this file is still accurate.
10. One commit with a real message.
11. State: what phase, what gate, what is next.

**Between phases: `/clear`.** Long contexts drift. Drift is this repo's documented failure
mode, twice over.

---

## 12. Anti-goals — the list of things that already went wrong

Kept because both failures were structural, not accidental, and both will recur under
schedule pressure.

1. **Do not let a stated constraint be silently overridden.** The old version of this file
   banned databases, routers, and frontend frameworks. All three arrived anyway and the file
   was never updated. If a constraint stops making sense, change it here first.
2. **Do not present architecture the code does not execute.** `ArrivalsService` exists,
   is wired, and is bypassed.
3. **Do not ship UI for logic that does not exist.** The H3 grid toggle.
4. **Do not let documentation drift from the code.** `graphify.html` claimed Redux; the code
   uses Zustand. The audit docs claim OSRM was stripped out; `api/osrm.ts` is in the tree.
5. **Do not inflate the pitch.** Not a transit app, not a platform, not an Uber competitor,
   not a Fleet Engine equivalent. §1.4 is the ceiling.
6. **Do not start the interesting phase before finishing the boring one.** Infrastructure
   is more fun than writing handler tests. Handler tests come first.

---

## 13. Deliverables this repo owes

| File | Contents | Phase |
|---|---|---|
| `BASELINE.md` | Coverage and behavior at `v0-poc`, for the before/after story | 0 |
| `CLAUDE.md` | This file, kept true | ongoing |
| `IDEAS.md` | Deferred ideas, each with the reason it was deferred | ongoing |
| `RESULTS.md` | Measured numbers against §9 | after 4.5 |
| `README.md` | §1.4 framing, honest limitations from §3.5, results | after 4.5 |
| `AUDIT.md` | The self-directed adversarial audit write-up | after 4.5 |

`AUDIT.md` is the most differentiated artifact in this repo. Most engineers cannot show a
case where they found their own architecture lying. Do not let it get lost in a rewrite.

**Housekeeping (do during Phase 1):** delete `graphify.html`, `tree.txt`, and
`frontend_features_prompt.md`. Move `repo-analysis/` → `docs/audit/`. Grep for inbound
references before deleting anything.