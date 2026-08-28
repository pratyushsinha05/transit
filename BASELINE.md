# BASELINE.md — Phase 0 ground truth

**Git SHA:** `c53488d8a52d4120acde65dd93cee9763387854b`
**Date:** 2026-08-27 (UTC)
**Tag:** `v0-poc` — **NOT CREATED.** The working tree is dirty (6 modified files). See §0.
**Toolchain:** Go 1.26.0 darwin/arm64 (`go.mod` declares `go 1.23.0`); Node/npm via `npm ci`; TypeScript 5.9.3; Vite 7.3.0.

This file is the "before" column. Numbers here are measured, not estimated.

---

## 0. Blocking precondition: the working tree is not clean

Phase 0 §2.1 requires a clean tree before tagging `v0-poc`. It is not clean. Six tracked
files carry uncommitted edits:

| File | What the uncommitted edit does |
|---|---|
| `backend/internal/handlers/arrivals.go` | `var arrivals []models.ArrivalEvent` → `make([]models.ArrivalEvent, 0)` so JSON emits `[]` not `null`. Correct and harmless. |
| `backend/migrations/004_seed_data.sql` | Pre-computes `hex_res9` literals for seed rows; deletes the redundant `UPDATE ... SET geom` (the `trg_location_geom` trigger from 003 already does it). Correct. |
| `frontend/src/services/api/arrivals.ts` | Null-guards `response.data`. Correct. |
| `frontend/src/config/wsConfig.ts` | Sets `location_update: 'LOCATION_UPDATE'` (correct — matches the Go const), **and** hard-codes `routeId: null, heading: null, h3_hex: null`. |
| `frontend/src/services/websocket/messageHandler.ts` | Rewrites `handleLocationUpdate` to read `device_id` / unix-seconds `timestamp` and to skip `route_id` / `h3_hex` entirely. |
| `frontend/src/hooks/useStops.ts` | Changes `fetchStops('route-101')` → `fetchStops()`, with the comment *"GET /api/stops calls StopRepository.GetAll() — no route_id needed."* |

Two of these conflict with the plan and must be resolved before Phase 1 starts:

1. **`wsConfig.ts` + `messageHandler.ts` resolve DEFECT-2 in the direction opposite to
   `CLAUDE.md` §7.2.** They make the *frontend* accept the backend's reduced shape. §7.2
   and prompt §3.2 say the *backend* adds `route_id` and `h3_hex`. Phase 1.2 will partially
   revert this work.
2. **`useStops.ts` is factually wrong and breaks stop loading.**
   `internal/handlers/stops.go:19-22` returns HTTP 400 `{"error":"route_id is required"}`
   when `route_id` is absent. It calls `StopRepository.GetByRouteID`, never `GetAll`.
   `GetAll` exists (`internal/database/stops.go:100`) but no handler reaches it. The
   committed version (`'route-101'`) at least returned rows if that route existed; the
   uncommitted version guarantees a 400 on every load.

**Both frontend and backend build clean with these edits applied** (see §1), so the tree is
buildable — it is just not committed, and not aligned with the contract.

---

## 1. Baseline build, vet, and coverage

### Backend

```
$ go build ./...          # clean, no output
$ go vet ./...            # clean, no output
$ go test ./... -cover
```

| Package | Coverage | Note |
|---|---|---|
| `transit-backend/pkg/geo` | **100.0%** | `distance_test.go`, table-driven |
| `transit-backend/internal/services` | **6.9%** | `geofencing_test.go` only; tests 2 pure funcs |
| `transit-backend/internal/handlers` | **0.0%** | no test files |
| `transit-backend/internal/hub` | **0.0%** | no test files |
| `transit-backend/internal/database` | **0.0%** | no test files |
| `transit-backend/internal/cache` | **0.0%** | no test files |
| `transit-backend/internal/config` | **0.0%** | no test files |
| `transit-backend/internal/middleware` | **0.0%** | no test files |
| `transit-backend/cmd/server` | **0.0%** | no test files |
| `transit-backend/internal/models` | — | `[no test files]` (structs only) |
| `transit-backend/migrations` | — | `[no test files]` (embed only) |

Integration tests: **none**. No `test/` directory, no testcontainers.

The `internal/services` 6.9% comes entirely from `TestCalculateHexAtResolution` and
`TestHexEdgeLengthMeters` — two package-level pure functions. **No method on
`GeofencingService` or `ArrivalsService` is exercised by any test.**

### Frontend

`node_modules` was absent at baseline; restored with `npm ci` (lockfile, no version drift).

```
$ npm run build     # tsc -b && vite build
✓ 471 modules transformed.
dist/index.html                   0.88 kB │ gzip:   0.49 kB
dist/assets/index-DBP9O_UX.css   34.89 kB │ gzip:  11.08 kB
dist/assets/index-BCpVCKnv.js   446.80 kB │ gzip: 139.29 kB
✓ built in 865ms
```

Type check clean. Zero frontend tests exist (no test runner in `package.json`).

Against `CLAUDE.md` §9 / the web perf budget: the app-page JS budget is 300 kB gzipped;
139.29 kB is within it. Recorded for the "after" column, not an action item.

---

## 2. The `Device` collision (prompt §2.2)

### What `models/device.go` declares

```go
package models

type Device struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}
```

Three fields. Mirrors the `devices` table from `migrations/001_create_tables.sql:6-11`
(`id`, `name`, `status`, `created_at` — `created_at` is not modelled).

### Is it used anywhere?

```
$ grep -rn "models.Device" --include="*.go" backend/
(no output — exit 1)
```

**Zero references.** `models/location.go` does not reference it. Nothing constructs it,
returns it, or scans into it.

The `devices` **table** is live, but only inside raw SQL: `database/locations.go:78,113,158`
(`LEFT JOIN devices d`) and `database/trips.go:34` (`JOIN devices d`). Those rows scan into
`models.NearbyBus` and `database.TripWithLocation`, never into `models.Device`.

### Is it the GPS-emitting entity?

No — it is the **registry record for** the GPS-emitting entity. It is a roster row
(identity + status), not a ping. The ping is `models.Location`
(`device_id, latitude, longitude, speed, accuracy, timestamp, hex_res9`), which is already
device-shaped and already uses `device_id` in both Go and SQL.

### There is no `Bus` type to collide with it

```
$ grep -rn "type .*Bus" --include="*.go" backend/
internal/models/location.go:27:type NearbyBus struct {
```

`Bus` never appears as a standalone Go type. It appears only as a qualifier inside names:
`NearbyBus`, `GetBusesInHex`, `GetBusesInHexes`, `GetBusesNearStop`, `FindNearbyBuses`,
`NearbyBusesTTL`, and `busLat`/`busLng`/`busHex` locals (15 `Bus` + 5 `Buses` + 2 `BusesTTL`
+ 3 `BusesNearStop` + 3 `BusesInHexes` + 2 `BusesInHex` occurrences across `internal/` and
`pkg/`).

**So the §6 mapping `Bus` → `Device` cannot collide with `models.Device`: there is no
`type Bus` for `type Device` to conflict with.** The rename it actually describes is
`NearbyBus` → `NearbyDevice` and `…Buses…` → `…Devices…` in method names.

### Verdict

Closest to **(C) dead code** — with a caveat that changes the recommendation.
The *struct* is unreferenced. The *concept* (a device roster keyed to the live `devices`
table) is real, is already joined in three queries, and is exactly what Phase 4's `stale`
event will need in order to know which devices should be reporting.

**Recommendation — decision belongs to the repo owner:**

> **Keep `models.Device` unchanged.** Do not rename anything into it and do not delete it.
> Phase 2 renames `models.NearbyBus` → `models.NearbyDevice` and the `…Bus…` method names
> to `…Device…`. No collision arises, no alternative target name is needed, and the roster
> type stays available for Phase 4.
>
> The alternative — deleting `device.go` as dead code in Phase 1 housekeeping — is
> defensible today but recreates the same type in Phase 4 under the same name.

---

## 3. The three undocumented features (prompt §2.3)

### 3.1 Trips — **live, not dead**

| Artifact | Status |
|---|---|
| `models.Trip` | **Dead.** `grep -rn "models\.Trip\b"` → zero hits. |
| `models.ArrivalEvent` (same file) | **Live.** Response type of `GET /api/arrivals`. |
| `database.TripRepository` | **Live.** Constructed in `main.go:60`. |
| `database.TripWithLocation` | **Live.** Return type of the only repo method. |
| `TripRepository.GetActiveTripsBeforeStop` | **Live.** Called from `handlers/arrivals.go:38` and `services/arrivals.go:54`. |
| Trips handler | **None.** No `handlers/trips.go`. |
| Trips service | **None.** |
| Route registered in `main.go` | **None.** No `/api/trips`. |

Reachability: `GET /api/arrivals` (`main.go:110`) → `ArrivalHandler.GetArrivals` →
`TripRepository.GetActiveTripsBeforeStop`. So trips are on the live request path via
arrivals only. There is no CRUD for trips; rows come from `004_seed_data.sql`.

`trips.status = 'IN_PROGRESS'` and `trips.current_stop` are only ever *read*. Nothing in
the codebase advances a trip. Seed data is the sole writer.

### 3.2 Route creator — **fully wired, end to end**

| Piece | Status |
|---|---|
| `Sidebar/RouteCreatorPanel.tsx` | Live; calls `createRoute` at line 37 |
| `Map/RouteCreatorMarkers.tsx` | Live; rendered by `MapContainer.tsx` |
| `Map/MapClickHandler.tsx` | Live; feeds `addCreatorStop` |
| `services/api/createRoute.ts` | `POST /api/routes` |
| Backend endpoint | **Exists.** `main.go:105` → `RouteHandler.CreateRoute` |
| Repository | `RouteRepository.Create` — single transaction, generates `route-<md5>` / `stop-<md5>` ids, writes `stops.geom` inline |
| Types | `models.CreateRouteRequest/Response` ↔ `CreateRoutePayload/CreateRouteResponse` in `types/domain.ts` — field names match |

This is the healthiest undocumented feature in the repo. `CLAUDE.md` does not mention it at
all; it is not in §3.2's package layout, §3.3's data flow, or §5's scope list.

Note: `RouteHandler.CreateRoute` (`handlers/routes.go:52`) builds an error string with
`string(rune('1'+i))`, which produces garbage for `i > 8`. Cosmetic, in the error message
only.

### 3.3 OSRM — **on the live path, hits a public third-party server**

`frontend/src/services/api/osrm.ts` is imported by `Map/RouteCreatorMarkers.tsx:5`
(`getRouteGeometry`). It is not a leftover.

- Endpoint: `https://router.project-osrm.org/route/v1/driving` — the **free public demo
  instance**, hard-coded, no API key, no timeout, no rate-limit handling.
- Purpose: snap operator-clicked waypoints to real roads for the drawn polyline.
- Failure mode is handled: on any error it logs and falls back to straight lines between
  the raw points.

Two things follow, both for the owner to decide, both out of Phase 0–3 scope per prompt §1.9:

- It is the **only outbound call to a third party** in the app, and it sends user-drawn
  coordinates to a server the project does not control.
- `CLAUDE.md` §5.1 lists "OSRM, Valhalla, or any routing engine" as **explicitly out of
  scope**. The repo already depends on one. The constraint and the code disagree — this is
  the same failure pattern §12.1 warns about.

---

## 4. The three known defects (prompt §2.4)

### DEFECT-1 — handler bypasses service — **CONFIRMED, and wider than documented**

```
$ grep -rn "database\." backend/internal/handlers/
internal/handlers/stops.go:11:      repo *database.StopRepository
internal/handlers/stops.go:14:func NewStopHandler(repo *database.StopRepository) *StopHandler {
internal/handlers/location.go:17:   repo  *database.LocationRepository
internal/handlers/location.go:23:func NewLocationHandler(repo *database.LocationRepository, cache *cache.DeviceCache, hub *hub.Hub) *LocationHandler {
internal/handlers/arrivals.go:13:   stopRepo *database.StopRepository
internal/handlers/arrivals.go:14:   tripRepo *database.TripRepository
internal/handlers/arrivals.go:17:func NewArrivalHandler(stopRepo *database.StopRepository, tripRepo *database.TripRepository) *ArrivalHandler {
internal/handlers/routes.go:14:     repo *database.RouteRepository
internal/handlers/routes.go:18:func NewRouteHandler(repo *database.RouteRepository) *RouteHandler {
```

`CLAUDE.md` §4 scopes DEFECT-1 to the arrivals handler. **Four of six handlers violate the
layering rule**, not one:

| Handler | Depends on | Verdict |
|---|---|---|
| `arrivals.go` | `*database.StopRepository`, `*database.TripRepository` | violation (documented) |
| `stops.go` | `*database.StopRepository` | violation (undocumented) |
| `location.go` | `*database.LocationRepository`, `*cache.DeviceCache`, `*hub.Hub` | violation (undocumented) |
| `routes.go` | `*database.RouteRepository` | violation (undocumented) |
| `nearby.go` | `*services.GeofencingService` | correct layer, concrete type not interface |
| `websocket.go` | `*hub.Hub` | fine |

The documented consequence holds and is confirmed: `main.go:68` constructs
`arrivalsService`, `main.go:69` discards it (`_ = arrivalsService`), and `main.go:80` builds
`ArrivalHandler` from repositories instead. `ArrivalsService.GetArrivalsForStop`,
`GetNearbyArrivals`, `isApproaching`, `CalculateETAWithTraffic`, and `DetectArrivalEvent`
are all unreachable at runtime. The k-ring approach detection at `services/arrivals.go:112`
never executes.

The verification command in `CLAUDE.md` §4 (`grep` returns nothing) can only pass if all
four handlers are converted. Prompt §3.1 describes converting only the arrivals handler.

### DEFECT-2 — WebSocket contract mismatch — **CONFIRMED, three-way not two-way**

There are **three** different envelopes in play, not two.

**A. `CLAUDE.md` §7.2 (declared canonical):** nested `data` object, `lat`/`lng` keys,
RFC3339 string timestamp.

```json
{"type":"LOCATION_UPDATE","data":{"device_id":"","route_id":"","lat":0.0,"lng":0.0,
  "speed":0.0,"heading":0.0,"h3_hex":"","timestamp":"RFC3339"}}
```

**B. `internal/hub/message.go` (what the backend actually sends):** flat, no `data`
wrapper, `latitude`/`longitude` keys, unix-seconds `int64` timestamp.

```go
type Message struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload,omitempty"`
	DeviceID  string      `json:"device_id,omitempty"`
	Latitude  float64     `json:"latitude,omitempty"`
	Longitude float64     `json:"longitude,omitempty"`
	Speed     float64     `json:"speed,omitempty"`
	Accuracy  float64     `json:"accuracy,omitempty"`
	Timestamp int64       `json:"timestamp,omitempty"`
}
```

**C. `frontend/src/config/wsConfig.ts` as committed at `c53488d`:** flat, `bus_id`,
`last_updated`, and a lowercase `'location_update'` type string that never matches the Go
const `"LOCATION_UPDATE"`.

Field delta, committed frontend expectation vs. backend `Message`:

| Frontend expects | Backend sends | Status |
|---|---|---|
| `bus_id` | `device_id` | **name mismatch** |
| `latitude` | `latitude` | ok |
| `longitude` | `longitude` | ok |
| `route_id` | — | **missing** (documented) |
| `speed` | `speed` | ok |
| `heading` | — | **missing** (not documented; also absent from `models.Location` and the DB — the backend has no heading anywhere) |
| `last_updated` | `timestamp` | **name mismatch**, and ISO-string vs unix-int64 **type mismatch** |
| `h3_hex` | — | **missing** (documented; the value exists as `loc.HexRes9` at `handlers/location.go:56` and is simply not copied into the message) |
| — | `accuracy` | backend-only, unused by frontend |
| — | `payload` | backend-only, never populated |
| type literal `'location_update'` | `"LOCATION_UPDATE"` | **case mismatch — no branch ever matched** |

The case mismatch means that at `c53488d` **no `LOCATION_UPDATE` message was ever
processed**; `handleMessage` fell through to `default:`. The uncommitted `wsConfig.ts` edit
fixes exactly that line.

Note also: `hex_res9` vs `h3_hex`. `models.Location` and the DB column both use `hex_res9`;
`CLAUDE.md` §7.2 and the frontend both use `h3_hex`. Pick one in Phase 1.2.

`frontend/src/types/domain.ts` `BusLocation` declares `routeId: string` and
`heading: number` as **required, non-optional**. Under the current backend both are always
absent. The type is wrong regardless of which direction DEFECT-2 is fixed.

### DEFECT-3 — ghost UI — **CONFIRMED, and `h3Helpers.ts` is independently dead**

- Toggle: `Sidebar/OperationsPanel.tsx:59-68` — the `H3 SPATIAL GRID` label block, already
  rendered at `opacity-50`.
- State: `store/slices/ui.ts` — `grid` in the `layerVisibility` type (line 36), in the
  `toggleLayer` union (line 38), and in the default `{ stops: true, buses: true, grid: false }`
  (line 61).
- Consumers of `layerVisibility.grid`: **only `OperationsPanel.tsx` itself.**
  `MapContainer.tsx` renders `MapCameraHandler`, `MapClickHandler`, `RoutePolyline`,
  `BusMarkers`, `StopMarkers`, `RouteCreatorMarkers` — and reads no layer state at all. No
  grid layer exists.
- `layerVisibility.stops` and `.buses` **are** consumed (`StopMarkers.tsx:88-90`,
  `BusMarkers.tsx:29-32`). Only the `grid` key is a ghost; the slice itself stays.

**Correction to the prompt's assumption.** Prompt §3.3 says to delete `utils/h3Helpers.ts`
"if the toggle was its only consumer." The toggle was never its consumer:

```
$ grep -rn "h3Helpers\|groupBusesByH3\|getBusesInHex\|debugH3Info" frontend/src/
src/utils/h3Helpers.ts:31:export const groupBusesByH3 = ...
src/utils/h3Helpers.ts:52:export const getBusesInHex = ...
src/utils/h3Helpers.ts:67:export const debugH3Info = ...
```

Definitions only — **zero call sites anywhere in the app**. The file is dead independent of
the toggle, and all three exports operate on `bus.h3Hex`, which the backend never sends
(see DEFECT-2). Safe to delete; it is not imported by anything.

---

## 5. Additional defects found (prompt §8 stop condition 6)

Not in `CLAUDE.md` §4. Recorded, **not fixed**. Each needs a scope decision.

### D4 — `H3_RESOLUTION` is configuration theatre

`config.go:66` reads it, `config.go:82` validates it 0–15, and **nothing ever consumes
`cfg.H3Resolution`.** `NewGeofencingService` hard-codes `resolution: 9`
(`services/geofencing.go:24`); `NewLocationRepository` hard-codes `resolution: 9`
(`database/locations.go:22`). `NewGeofencingServiceWithResolution` and
`NewLocationRepositoryWithResolution` exist and are never called.

Contradicts `CLAUDE.md` §3.1 ("Resolution 9 (~175m edge), **configurable via
`H3_RESOLUTION`**") and §7.4. Setting `H3_RESOLUTION=7` in `infra/docker-compose.yml`
changes nothing.

### D5 — `LOG_LEVEL` and `REDIS_POOL_SIZE` are also dead

`cfg.LogLevel` is never read; logging is `log.Printf` and Echo's default logger throughout.
Contradicts `CLAUDE.md` §7.1 ("Logging: structured, levelled by `LOG_LEVEL`") — logging is
neither structured nor levelled. `cfg.RedisPoolSize` is never read; `cache/redis.go:26`
hard-codes `PoolSize: 50`. Of the four toggles §7.4 names, only `DB_MAX_CONNS` /
`DB_MIN_CONNS` are actually wired (`database/db.go:31-32`).

### D6 — `omitempty` silently drops legitimate zero values

Every numeric field on `hub.Message` is tagged `omitempty`. A stopped device — the single
most common real state, a bus at a light — serializes with **no `speed` key at all**, and
the frontend's `parseFloat(raw[schema.speed] || 0)` cannot distinguish "stopped" from "not
reported". `latitude`/`longitude`/`accuracy` have the same hazard at exactly 0.

This must be fixed in the same commit as DEFECT-2 or the new serialization test will encode
the bug.

### D7 — the hub cannot be shut down

`Hub.Run()` is `for { select { ... } }` over three channels with no `done` or `context`
case. `main.go:73` starts it with `go wsHub.Run()` and nothing ever stops it; graceful
shutdown (`main.go:145`) calls `e.Shutdown` only. Client goroutines
(`handlers/websocket.go:41-42`) are likewise fire-and-forget.

Directly contradicts `CLAUDE.md` §7.1: *"every goroutine has exactly one owner responsible
for its shutdown. No fire-and-forget. The hub owns its own goroutines and must drain
cleanly on shutdown."*

Relevant to Phase 3: the slow-consumer hub test is hard to write without a shutdown path,
and `hub.go:42-49` — the `default:` branch that silently drops for a full client buffer — is
precisely the behaviour that test needs to pin down. It currently has a five-line comment
explaining that it does nothing.

### D8 — `GET /api/stops` cannot return all stops

`handlers/stops.go:19-22` hard-requires `route_id`. `StopRepository.GetAll`
(`database/stops.go:100`) is written, correct, and unreachable. The frontend has wanted an
all-stops endpoint since it was written (`services/api/stops.ts` makes `routeId` optional).
This is the root cause the uncommitted `useStops.ts` edit tried to work around from the
wrong side.

### D9 — `GET /api/arrivals` is broken end to end, today

Backend returns `models.ArrivalEvent`:
`{trip_id, device_id, device_name, eta_minutes}`.

`API_CONFIG.schemas.arrival` maps to: `id`, `bus_id`, `eta`, `status`, `route_number`,
`route_id`, `timestamp`. **Not one field name overlaps.** `transformArrival`
(`services/api/transformers.ts:26`) therefore throws `Missing required fields: id or busId`
on every row, and `fetchArrivals` rejects. `useArrivals` is consumed by
`Sidebar/ArrivalsList.tsx:15` and `Map/StopMarkers.tsx:33`, so the arrivals panel and every
stop popup are non-functional.

This matters for sequencing: Phase 1.1 changes the arrivals response shape anyway (to
`services.ArrivalPrediction`, which adds `distance_km`, `current_speed`, `hex_res9`,
`is_approaching`). Both sides of `/api/arrivals` need to be decided together, and
`ArrivalPrediction` still has no `status` or `timestamp` field.

### D10 — dead model structs

`models.Device` (§2) and `models.Trip` — both zero references.

### D11 — migrations may apply twice on a fresh volume

`infra/docker-compose.yml:23` mounts `../backend/migrations` into
`/docker-entrypoint-initdb.d`, so Postgres runs every `.sql` at first init. The Go binary
then runs the same files through its own `schema_migrations` ledger
(`database/db.go:57`), which starts empty and therefore re-applies all four.

001–003 are idempotent (`IF NOT EXISTS`, guarded `DO $$` blocks). `004_seed_data.sql` is
not: its three `INSERT INTO location_history` statements carry no `ON CONFLICT`, so seed
pings are duplicated. Routes/stops/devices/trips do use `ON CONFLICT`, so only
`location_history` doubles.

---

## 6. Summary of contradictions between `CLAUDE.md` and the code

`CLAUDE.md` §11.6: *"If a fact in this file turns out to be wrong, fix this file before
continuing."* These need owner sign-off before Phase 1:

| § | Claim | Reality |
|---|---|---|
| §3.1 | H3 resolution "configurable via `H3_RESOLUTION`" | Hard-coded 9 in two places; config value unused (D4) |
| §4 DEFECT-1 | Scoped to the arrivals handler | Four of six handlers violate the layering rule |
| §7.1 | "Logging: structured, levelled by `LOG_LEVEL`" | Neither; `cfg.LogLevel` unused (D5) |
| §7.1 | Hub "must drain cleanly on shutdown" | No shutdown path exists (D7) |
| §7.2 | Canonical envelope is nested `data` + `lat`/`lng` + RFC3339 | Backend sends flat + `latitude`/`longitude` + unix int64; a third shape again in the frontend |
| §7.2 / §3.1 | `h3_hex` | Go and SQL both call it `hex_res9` |
| §7.4 | Four documented toggles | Only `DB_MAX_CONNS`/`DB_MIN_CONNS` are wired (D4, D5) |
| §5.1 | "OSRM … explicitly out of scope" | `services/api/osrm.ts` is live on the route-creator path (§3.3) |
| §3.2 | Package layout omits route creation | `POST /api/routes` + `RouteRepository.Create` exist and work |
| §6 | Rename `Bus` → `Device` | No `type Bus` exists; the real rename is `NearbyBus` → `NearbyDevice` (§2) |

---

## 7. Phase 3 targets vs. this baseline

| Package | Baseline | Target (`CLAUDE.md` §8.1) | Gap |
|---|---|---|---|
| `pkg/geo` | 100.0% | ≥ 95% | already met |
| `internal/services` | 6.9% | ≥ 60% | +53.1 pts |
| `internal/handlers` | 0.0% | ≥ 50% | +50 pts |
| `internal/hub` | 0.0% | ≥ 50% | +50 pts |
| `internal/database` | 0.0% | integration test | needs testcontainers (new dependency — must be approved) |
| `internal/cache` | 0.0% | "no package at 0%" | not in the §8.1 table but caught by the hard rule |
| `internal/config` | 0.0% | "no package at 0%" | same |
| `internal/middleware` | 0.0% | "no package at 0%" | same |
| `cmd/server` | 0.0% | "no package at 0%" | composition root; §8.1 lists no target |

`CLAUDE.md` §8.1's "no package sits at 0%" is stricter than its own table, which names five
packages. Four more (`cache`, `config`, `middleware`, `cmd/server`) sit at 0% and have no
stated target.
