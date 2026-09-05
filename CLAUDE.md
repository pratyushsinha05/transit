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

This is the **first of three documented drifts** in this repo. The second is architectural
(§4, DEFECT-1). The third was in this file itself: §7.2 once specified a nested envelope and
a `heading` field, both written from audit documents rather than from the code, and Phase 1
began building against them before the contradiction was caught.

All three happened for the same reason: a stated claim was not checked against the code.

**This file is the forcing function.** It is version-controlled, read every session, and
updated *before* the code, not after. **Every factual claim in it must be traceable to a
file and line.** When in doubt, read the code — never another document.

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

## 2. Phase history and gate protocol

**Phases 0–3 are done. Phase 4 has not started. Phase 4.5 is blocked. Phase 5+ is deferred.**

| Phase | Goal | Outcome |
|---|---|---|
| **0** | Ground truth | **DONE.** Real coverage recorded; `Device` collision resolved — no `type Bus` exists, so there was none |
| **1** | Close every open defect in §4 | **DONE.** DEFECT-1 through 7 closed; see `AUDIT.md` §3 for the full record |
| **2** | Domain rename | **DONE.** Four commits, zero behavior change. Identifiers are generic; HTTP paths and JSON tags kept as wire contract |
| **3** | Tests off zero | **PARTIAL.** `services` and `hub` cleared their targets; `handlers` did not, and four packages remain at 0% — see §8.1 |
| **4** | Geofence events | **NOT STARTED.** Six functions are tagged `Phase4Reserved` in source for this work |
| **4.5** | Route-projected distance | **BLOCKED.** The `routes` table has no geometry column — see §6.5 and `AUDIT.md` §8 |
| **5+** | Infrastructure | **Not started. Do not begin without explicit instruction.** |

**Gate protocol.** At the end of every phase:

1. Full test suite green
2. This file re-read and corrected if the phase changed any stated fact
3. One commit, real message, no "wip"
4. If the phase overran its estimate, **cut scope** rather than carrying half-finished work
   into the next phase
5. **Citation gate.** Extract every citation mechanically:


   ```
   grep -oE '[A-Za-z0-9_./-]+\.(go|ts|tsx|sql|yml|sh|md):[0-9]+' CLAUDE.md | sort -u
   ```

   Pass that list to a claim-checker subagent as its input. It reads **source, not this
   file.** The agent MUST account for every entry — exact, wrong (with the correct line), or
   not-applicable (forward-looking, e.g. §8.2's `integration_test.go`). **A checker that
   returns fewer rows than the extracted list has under-covered; re-run it.**

   *Rationale:* the first run of this gate found the DEFECT-3 staleness but omitted four §6
   JOIN citations and three `ui.ts` line numbers from its table, then concluded all citations
   were accurate. Asking an agent to both *find* and *check* citations reproduces the failure
   the gate exists to catch. The list is mechanical; only the verification is judgment.
   Earlier evidence for the gate itself: three §3.1 citations were wrong across two passes —
   the Redis `PoolSize` line, both H3 resolution lines, and an OSRM claim that contradicted
   the Map row two lines above it — and two propagated into `docs/explain/00-inventory.md`
   before anyone checked.   
   **Three failure categories, not two.** Classify every failure as *transcription*
   (source read correctly, copied wrong), *drift* (right when written, the code has since
   moved), or **stale-at-generation** (the report inherited a claim from an earlier audit
   round instead of observing source, so it was already false when written). The third is
   the one that hides. Worked example: `docs/explain/00-inventory.md` §7.3.l records
   `services/interfaces.go:6` as importing `database`. Commit `fe918e0` removed that import
   *before* commit `57afe4e` generated the zone reports — the claim was false the moment it
   was written, and it propagated into three Pass 3 documents.

   **Verify the specific claim, not the shape of the line.** A citation reading "imports
   `database`" passes only if that line imports `database` — not merely if it contains an
   import statement. The Pass 3 claim-checker *passed* the wrong-package claim in two of the
   three documents carrying it: it confirmed line 6 exists and holds an import, and stopped.
   The error surfaced only when the consistency pass grepped all five docs for the repeated
   citation string. A checker that confirms a weaker property than the claim and reports
   pass is anti-goal 10 in miniature.

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
| Spatial index | Uber H3 | **Hard-coded resolution 9** (~175m edge) in `services/geofencing.go:22` and `database/locations.go:23`. `H3_RESOLUTION` is read (`config.go:66`) and validated (`config.go:82-83`) but **never consumed** — see §7.4(a) and D4/C7 in `IDEAS.md`. |
| Hot state | Redis | Latest known location per device — **write-only today.** Nothing reads it back: `grep -rn "GetDeviceState\|GetDeviceLocation" backend/` finds no external caller (D21). Pool size hard-coded 50 in `cache/redis.go:28`; `REDIS_POOL_SIZE` unused.Pool size hard-coded 50 in `cache/redis.go:28`; `REDIS_POOL_SIZE` unused. |
| Realtime | Native Go channels + WebSocket hub | No message broker |
| Frontend | React 19 + TypeScript + Vite | SPA |
| FE state | Zustand | 5 slices |
| Styling | Tailwind | Dark "HUD" palette |
| Map | `react-leaflet` + CartoDB Dark Matter tiles | |
| Routing (frontend only) | OSRM public demo instance | `frontend/src/services/api/osrm.ts:9`, called by `frontend/src/components/Map/RouteCreatorMarkers.tsx:5`. Snaps operator-drawn waypoints to roads. Falls back to straight lines on error. The only third-party **API** call in app code — but not the only outbound request: the CartoDB tiles (`MapContainer.tsx:44`) load on every map render and the Space Mono webfont (`index.html:10-12`) on every page load. |
| Config | `godotenv` | Defaults in `infra/docker-compose.yml` |
| Local orchestration | Docker Compose | `transit-network` bridge |

> **Note:** a prior doc (`graphify.html`) claimed Redux Toolkit. That was false. The
> codebase uses **Zustand**. `docs/audit/ESSENCE.md:6` claims OSRM was stripped out. Also
> false — see the table above. (That source's wording is softer than this rebuttal implies —
> "strips away complex routing engines (like OSRM) in favor of high-throughput telemetry
> ingestion" reads as emphasis rather than a claim of removal — but a document saying the
> system strips OSRM away is misleading when OSRM is live on the route-creator path.) If a doc disagrees with the code, the doc is wrong.

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
backend/internal/config/        Env loading.
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

**Approach detection** (`backend/internal/services/arrivals.go`): k-ring radius 3 at
resolution 9 = 37 cells ≈ 500m around the zone. If the device's hex is in that set,
`isApproaching = true`. On the live request path as of `5ab7448` — it was orphaned before
DEFECT-1 was closed.

### 3.5 Known algorithmic limitations

| Assumption | Status |
|---|---|
| Straight-line distance ≈ travel distance | **OPEN — blocked.** Route-projected distance requires a route polyline; `routes` has no geometry column (`IDEAS.md` D41). Was previously recorded here as "FIXED in Phase 4.5"; that was wrong |
| Proximity ⇒ approaching | **OPEN — blocked**, same root cause and same correction |
| Instantaneous speed predicts the next few minutes | **Accepted limitation.** Document honestly in README. Do not fix |
| Stopped ⇒ 20 km/h `DefaultSpeed` | **Accepted limitation.** Document honestly in README. Do not fix |

The two accepted limitations are POC-appropriate trade-offs, not bugs. The two blocked ones
are not fixed and must not be described as fixed — that error is recorded in `AUDIT.md` §8.

---

## 4. Defects

**The full record — what each defect was, why it was missed, and what the fix cost — is in
`AUDIT.md` §3.** It is kept there rather than here because it is evidence, not a working
rule. Nothing was summarized away in the move; the postmortems are intact.

| Defect | Status |
|---|---|
| DEFECT-1 — handler bypasses service | **CLOSED** at `5ab7448` (incomplete) → `fe918e0`. Was five handlers, not one |
| DEFECT-2 — WebSocket contract mismatch | **CLOSED.** Canonical envelope in §7.2 |
| DEFECT-3 — ghost UI (H3 grid toggle) | **CLOSED** at `066b129` |
| DEFECT-4 — test coverage | **OPEN.** See below |
| DEFECT-5 — `omitempty` drops zero values | **CLOSED.** Same commit as DEFECT-2 |
| DEFECT-6 — hub has no shutdown path | **CLOSED**, then extended by D28 (drain before close) |
| DEFECT-7 — migrations apply twice | **CLOSED.** The compose-level double-apply was later removed outright (D45) |

### DEFECT-4 — test coverage (open)

The only defect in this section still open. `internal/handlers` sits at 12.7% against a 50%
target, and `internal/cache`, `internal/config`, `internal/middleware` and `cmd/server` are
at 0.0%, against a hard rule in §8.1 that no package sits at 0% except the composition root.
Current numbers and targets are in §8.1.

### Tracked but not scheduled

`D4` (H3_RESOLUTION / LOG_LEVEL / REDIS_POOL_SIZE dead config), `D8` (`GET /api/stops`
hard-requires `route_id`; `ZoneRepository.GetAll` is written and unreachable), `D15`
(`GetActiveTripsBeforeZone` never scopes by route), `D48` (`GET /api/nearby/devices` returns
500 — pgx cannot encode an int for `$2 || ' minutes'`). All in `IDEAS.md`, which is the live
findings log. Each needs a scope decision.

---

## 5. Scope guards — what NOT to build

This repo has drifted three times. If a change requires one of these, stop and ask.

### 5.1 Algorithms — OUT

- No prediction-accuracy work beyond Phase 4.5: map-matching, Kalman filtering,
  particle filters, event-time processing.
- No distributed-systems papers. Those belong to `llm-router`.
- No NEW routing engine. The existing frontend OSRM client is live (§3.1): do not extend,
  do not remove.

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

### 5.5 Pre-existing features the audit missed

Three features exist in the tree and appear in **no** audit document. Phase 0 established
their status:

- **Trips.** `models.Trip` is dead (zero references), but `database.TripRepository` is live
  and reachable: `GET /api/arrivals` → `ArrivalHandler` → `GetActiveTripsBeforeStop`.
  Nothing advances a trip at runtime; `004_seed_data.sql` is the only writer.
- **Route creator.** Fully wired end to end — `RouteCreatorPanel.tsx` → `createRoute.ts` →
  `POST /api/routes` → `RouteRepository.Create`. The healthiest undocumented feature here.
- **OSRM.** Live on the route-creator path (§3.1).

**Do not modify any of them in Phases 0–4.5**, except where a rename in §6 mechanically
touches them.

---

## 6. Domain rename (Phase 2) — shipped

The rename is **done**, in four commits with zero behavior change. The full identifier tables
are in git history; they are a record of completed work, not instructions. What remains
binding from that phase:

> **The `Device` collision was RESOLVED in Phase 0. There is none.** `grep -rn "type .*Bus"`
> finds no `type Bus`, so nothing collides with `models.Device`.
>
> **Decision: keep `models.Device` unchanged.** Do not rename into it. Do not delete it —
> Phase 4's `stale` event needs a device roster to know which devices should be reporting.

- Core identifiers are generic; the demo layer may stay transit-flavoured. Transit vocabulary
  is permitted **only** in `migrations/004_seed_data.sql`, demo fixtures, and user-facing UI
  copy.
- **HTTP route paths, query parameters and JSON struct tags were deliberately NOT renamed.**
  They are wire contract; renaming them would be a behavior change. This is why
  `GET /api/stops` is served by `ZoneHandler.GetZones`.
- **Never edit an existing migration.** The rename shipped as `005_rename_stops_zones.sql`.
- Use a type-aware rename tool. **Never blind `sed -i`** across the tree — it hits strings,
  comments, and seed data.

---

## 6.5 Phase 4.5 — route-projected distance (BLOCKED)

The one deliberate exception to §5.1, specified but **not executable as written**: it requires
projecting a device and a zone onto a route polyline, and **`routes` has no geometry column**
in any migration. `ST_LineLocatePoint` has nothing to project onto. See `AUDIT.md` §8 and
`IDEAS.md` D41–D43.

The intended algorithm, kept for whenever the data exists:

1. Project device position onto its route polyline: `ST_LineLocatePoint` → fraction 0–1
2. Project the zone onto the same polyline → fraction 0–1
3. Remaining distance = difference along the line, via `ST_LineSubstring` + `ST_Length`
4. If device fraction > zone fraction, the device has **passed**. Emit **no ETA**. Do not
   emit a large one.

**Rules that still bind if it is ever unblocked:**

- `ST_Length` on a `geometry` in SRID 4326 returns **degrees**. Cast the result of
  `ST_LineSubstring` to `::geography` for metres — not its inputs; `ST_LineLocatePoint` and
  `ST_LineSubstring` both take `geometry`.
- Devices with no assigned route fall back to Haversine. Keep that path working and tested.
- Test the passed-the-zone case explicitly — it is the failure this phase exists to kill.
- The ETA **time model is unchanged**: still `distance / speed`. Only distance improves.
  Do not touch `DefaultSpeed` here.

**Unblocking needs four pieces before the algorithm is even reachable:** a migration adding
the column, a persistence path (the OSRM polyline is already computed browser-side and
discarded — D42), widening `TripWithLocation` with `RouteID`, and a seed backfill. Two of
those cross existing scope guards. The one-day estimate was written against an assumed schema.

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
  fire-and-forget. **Currently violated — see DEFECT-6.**
- Channels: the writer closes, never the reader. Buffered sizes are named constants with a
  comment justifying the number.
- No global mutable state. No `init()` for anything but constant registration.
- `pkg/geo` stays pure: no I/O, no logging, no clock reads, no state.
- Logging: **target state** is structured and levelled by `LOG_LEVEL`. **Current state is
  neither** — `log.Printf` plus Echo's default logger throughout, and `cfg.LogLevel` is
  never read. Tracked as D4/D5 in `IDEAS.md`, unscheduled. Do not write new `log.Printf`
  calls into a tight ingestion loop regardless.

### 7.2 WebSocket envelope (canonical — resolves DEFECT-2, DEFECT-5)

Flat envelope. No `data` wrapper. No `heading`, and none is being added.

**The "does not exist anywhere in this codebase" claim was wrong when written.** It was
backend-only true. `heading` was removed from the Go envelope in C5, but the frontend kept
declaring it **required** on `BusLocation` and kept fabricating it: `messageHandler.ts`
hardcoded `heading: 0` and `transformers.ts` read a key the backend never sends, defaulted it
to `0`, then range-validated the constant it had just produced. `BusMarkers.tsx` rendered the
result as a labelled `HEADING` readout showing `0°` for every device. Removed from the
frontend in `ec8a921`. The claim is true as of that commit and was not before it.

`backend/internal/hub/message_test.go:51-52` asserts the key is absent from the serialized
message. That test is the guard; do not "clean up" the word `heading` out of it.

An earlier version of this section specified a nested `data` object and a `heading` field;
both were written from audit docs, not the code, and were wrong. This is the correction.

```go
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
```

- **No `omitempty` on numeric fields** (DEFECT-5). A stopped device must serialize
  `"speed": 0`, not omit the key.
- **`timestamp` is unix seconds (`int64`), not RFC3339.** Matches what `models.Location`
  already carries — no conversion at the point the message is built.
- **DB column and Go model field stay `hex_res9`** (`location_history.hex_res9`,
  `models.Location.HexRes9`). `Message.H3Hex` with JSON tag `h3_hex` is a field on this
  struct only, populated from `loc.HexRes9`. One name internally, one on the wire.
- **The type literal is `"LOCATION_UPDATE"`, uppercase.** The frontend once used
  `'location_update'`, which matched nothing.
- **`route_id` is derived server-side from the device's active trip. It is never accepted
  on ingest.** `POST /api/location` takes no `route_id` field. Resolved via
  `database.DeviceRouteRepository` (`backend/internal/database/device_routes.go`) —
  deliberately a new file, not an addition to `trips.go`, since §5.5 scopes `trips.go` out.
  One query per ingest against `trips WHERE device_id = $1 AND status = 'IN_PROGRESS'`. Trip
  data is static in this POC, so there is nothing to cache-invalidate and a per-ping query
  is cheap and correct. If ingestion ever approaches the §9 throughput target and this query
  shows up in the p99 budget, the fix is a Redis-cached device→route mapping — not built now
  (YAGNI).

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

Most config values are declared in `infra/docker-compose.yml` with a default. Three are not —
see the second dead category below. Every line number here was verified against source.

**Actually wired — 7.** `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` reach the
pool via `database.New`; `DB_MAX_CONNS` and `DB_MIN_CONNS` are consumed at
`database/db.go:31-32`. `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD` are consumed at
`cache/redis.go:22,26`. `SERVER_HOST` / `SERVER_PORT` build the listen address at
`cmd/server/main.go:157`.

**`ENV` is wired, weakly.** Parsed at `config/config.go:69` and read at
`cmd/server/main.go:53` in a startup `log.Printf`. It changes no behavior, but it is **not**
dead — an earlier pass listed it as ignored, which was wrong.

There are two dead categories, and they are different bugs with different fixes.

**(a) Read, validated, then ignored — 3.** These reach `Config` and are then never consumed:

| Var | Parsed at | Also validated at | Value actually used |
|---|---|---|---|
| `REDIS_POOL_SIZE` | `config.go:59` | — | hard-coded `PoolSize: 50` at `cache/redis.go:28` |
| `H3_RESOLUTION` | `config.go:66` | `config.go:83` (range 0–15) | hard-coded `9` at `services/geofencing.go:22` and `database/locations.go:23` |
| `LOG_LEVEL` | `config.go:70` | — | nothing; logging is bare `log.Printf` |

`H3_RESOLUTION` is the most misleading of the three: it is *validated*, which reads as
evidence that it matters. Fix is to thread `cfg` into the constructors —
`NewGeofencingServiceWithResolution` and `NewLocationRepositoryWithResolution` already exist
and are never called. Tracked as D4/C7 in `IDEAS.md`.

**(b) Never parsed at all — 3.** These appear only in `.env.example`. They are absent from
`Config` (`config.go:12-38`) **and** from `infra/docker-compose.yml`, so setting them in the
compose environment would not even reach the process:

| Var | Declared at | Value actually used |
|---|---|---|
| `WS_READ_BUFFER_SIZE` | `.env.example:60` | hard-coded `1024` at `handlers/websocket.go:14` |
| `WS_WRITE_BUFFER_SIZE` | `.env.example:61` | hard-coded `1024` at `handlers/websocket.go:15` |
| `WS_PING_INTERVAL` | `.env.example:64` | derived, `pingPeriod = (pongWait * 9) / 10` at `hub/client.go:18` |

Fix for (a) is wiring; fix for (b) is deciding whether the knob should exist at all and then
either adding it to `Config` and compose, or deleting it from `.env.example`. Do not conflate
them.

Adding a config value without wiring it is a bug. Adding one without a documented default
is also a bug. Adding one to `.env.example` alone — category (b) — is both.

---

## 8. Testing

### 8.1 Targets and current state

| Package | Baseline | Current | Target | Met? |
|---|---|---|---|---|
| `backend/pkg/geo` | 100.0% | **100.0%** | Keep ≥ 95% | ✅ |
| `backend/internal/services` | 6.9% | **95.0%** | ≥ 60% | ✅ |
| `backend/internal/hub` | 0.0% | **86.1%** | ≥ 50% | ✅ |
| `backend/internal/handlers` | 0.0% | **12.7%** | ≥ 50% | ❌ |
| `backend/internal/database` | 0.0% | 0.0% | Covered by integration test | only under `-tags=integration` |
| `backend/internal/cache` | 0.0% | 0.0% | Covered by integration test | only under `-tags=integration` |
| `backend/internal/config` | 0.0% | 0.0% | ≥ 40% | ❌ |
| `backend/internal/middleware` | 0.0% | 0.0% | ≥ 40% | ❌ |
| `backend/cmd/server` | 0.0% | 0.0% | **Exempt** — composition root | — |

**`internal/hub` coverage is nondeterministic** — it has been observed at both 86.1% and
87.3% across consecutive runs. **Any gate on it must use the observed minimum.** Gating at
87% would fail four runs in five (`IDEAS.md` D40).

**Hard rule: no package sits at 0%**, except `cmd/server`. **This rule is currently not met**
— four packages are at zero. Stated plainly rather than softened.

### 8.2 What to test

- **Handlers:** `httptest` + mocked service interface. Every endpoint gets a happy path and
  at least one failure path (bad input, service error).
- **Services:** mocked repository interface. Cover the geofence logic DEFECT-1 was hiding —
  k-ring membership, hex boundary cases, the `DefaultSpeed` fallback branch.
- **Hub:** connect, broadcast to N clients, disconnect mid-broadcast, **slow consumer that
  does not read**. The slow-consumer case matters most — it is the real failure mode, it
  pins down the silent-drop `default:` branch at `hub.go:42-49`, and it feeds the later
  infrastructure story. **Requires DEFECT-6 closed first.**
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
**DEFECT-7 must be closed before any of these are measured** — duplicated seed rows would
corrupt the baseline.

---

## 10. Operations (as-built)

- `./deploy.sh` — verifies prerequisites, brings the backend stack up via the Makefile,
  builds the Vite frontend, serves a preview
- `./cleanup.sh` — tears everything down, destroys volumes and `node_modules`
- The `Makefile` is the command registry and holds far more than the handful once listed
  here. `make help` is the authoritative list; `README.md` documents the commonly used
  subset (`build`, `check-deps`, `docker-up`, `docker-health`, `docker-down`, `docker-clean`,
  `docker-logs`, `db-migrate`, `db-seed`, `rebuild-backend`, `test-unit`, `test-integration`)
- `make test-integration` requires Docker and passes `-tags=integration`; without the tag the
  build-tagged integration test is skipped silently and the target reports a false pass
  (`IDEAS.md` D46)
- `main.go` is the **single authority** for schema migrations. The compose file no longer
  mounts migrations into Postgres's init path (`IDEAS.md` D45)

Any new Make target must be documented in `README.md` in the same commit.

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
6. **If a fact in this file turns out to be wrong, stop and fix this file before writing any
   code against it.** Documenting the contradiction and proceeding anyway is not compliance
   — that is how Phase 1 nearly shipped a `heading` field that exists nowhere in the
   codebase.
7. Never `git push`, force-push, rewrite history, or `rm -rf`. Local commits only.

**At the end of the session:**

8. Run the full test suite.
9. Confirm this file is still accurate.
10. One commit with a real message.
11. State: what phase, what gate, what is next.

**Between phases: `/clear`.** Long contexts drift. Drift is this repo's documented failure
mode, three times over.

---

## 12. Anti-goals — the list of things that already went wrong

Kept because every one was structural, not accidental, and each will recur under schedule
pressure.

1. **Do not let a stated constraint be silently overridden.** The old version of this file
   banned databases, routers, and frontend frameworks. All three arrived anyway and the file
   was never updated. If a constraint stops making sense, change it here first.
2. **Do not present architecture the code does not execute.** `ArrivalsService` existed, was
   wired, and was bypassed for months.
3. **Do not ship UI for logic that does not exist.** The H3 grid toggle.
4. **Do not let documentation drift from the code.** `graphify.html` claimed Redux; the code
   uses Zustand. `ESSENCE.md` claims OSRM was stripped out; `api/osrm.ts` is live.
5. **Do not write a spec from another document.** §7.2 once specified a `heading` field that
   exists in no model, no column, and no payload — because it was written from audit prose
   instead of from `message.go`. Phase 1 began building against it.
6. **Do not document a contradiction and then proceed anyway.** `BASELINE.md` §6 listed ten
   doc/code conflicts, including the `heading` one, and the next commit planned work against
   the doc regardless. Finding the problem is not the same as stopping.
7. **Do not inflate the pitch.** Not a transit app, not a platform, not an Uber competitor,
   not a Fleet Engine equivalent. §1.4 is the ceiling.
8. **Do not start the interesting phase before finishing the boring one.** Infrastructure is
   more fun than writing handler tests. Handler tests come first.
9. **Do not render a value the system cannot produce.** The `HEADING` readout was hardcoded
   to `0` in both producers and displayed a fabricated number for every device. §7.3's rule
   ("no new UI control ships without working logic") did not catch it because the control was
   not new.
10. **Do not trust a check that defines its own scope.** The first CLAUDE.md citation gate
    reported "all line numbers accurate" while silently skipping seven citations. Extract the
    checklist mechanically; let the agent verify, not enumerate.

---

## 13. The five documents

| File | Contents |
|---|---|
| `README.md` | §1.4 framing, quickstart, measured numbers, honest limitations from §3.5 |
| `ARCHITECTURE.md` | System design: the ping's path, components, spatial model, concurrency, glossary |
| `AUDIT.md` | The self-directed adversarial audit and the full defect record |
| `CLAUDE.md` | This file — the working contract, kept true |
| `IDEAS.md` | The findings log: deferred items, each with the reason |

`AUDIT.md` is the most differentiated artifact in this repo. Most engineers cannot show a
case where they found their own architecture lying, twice, including in their own spec. Do
not let it get lost in a rewrite.

Diagrams live in `docs/diagrams/` and are embedded from `ARCHITECTURE.md`.