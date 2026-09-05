# Phase 4.5 — route-projected distance: BLOCKED

**Audience:** an executor with no prior context on this repository.

**Status: this is not an execution plan. Phase 4.5 cannot be executed as specified.**
The verification step that precedes planning found that the data structure the phase depends
on does not exist. Per the instruction that produced this document — *"If the geometry does
not exist, STOP at the plan and report. Do not design around it."* — no implementation is
designed below. §4 states what would have to be built first, as a scoping input for a human
decision, not as work to start.

**Repo root:** the directory containing `backend/`, `frontend/`, `infra/`.
**Go module:** `transit-backend` (`backend/go.mod`), `go 1.25.0`.
**Verified at:** commit `fb7a6fe`, 2026-09-04, working tree clean apart from untracked
planning docs.

---

## 1. What the phase was supposed to do

`CLAUDE.md` §6.5 specifies the one deliberate exception to the "no prediction-accuracy work"
scope guard (§5.1). It replaces straight-line ETA distance with along-route distance:

1. `ST_LineLocatePoint(route_geom, device_point)` → fraction 0–1
2. `ST_LineLocatePoint(route_geom, zone_point)` → fraction 0–1
3. remaining distance = `ST_Length(ST_LineSubstring(route_geom, f_device, f_zone))`
4. if `f_device > f_zone` the device has **passed** → emit **no** ETA, not a large one

This exists to kill two real defects in the current ETA, both recorded in §3.5's limitations
table as "FIXED in Phase 4.5":

- **Straight-line distance ≈ travel distance.** A device 400 m away across a river or a
  one-way system may be 3 km by road. ETAs skew systematically optimistic.
- **Proximity ⇒ approaching.** Distance is symmetric, so a device that *passed* the zone
  200 m ago reads as arriving in under a minute.

Both are real. Neither is fixed by this document.

---

## 2. The blocker: there is no route polyline, and nowhere to put one

### 2.1 The `routes` table has no geometry column

`backend/migrations/001_create_tables.sql:13-18`, verbatim and complete:

```sql
-- Routes table
CREATE TABLE IF NOT EXISTS routes (
    id TEXT PRIMARY KEY,
    name TEXT,
    description TEXT
);
```

Three columns. No `geom`. No `GEOMETRY`, no `LINESTRING`.

### 2.2 No later migration adds one

`grep -rn 'geom\|geometry\|LINESTRING\|ST_\|Geometry' backend/migrations/*.sql` returns 26
lines. Every one of them concerns **`zones`** (formerly `stops`) or **`location_history`**,
and every geometry in the schema is a **`POINT`**:

| Migration | Line | What it does | Table |
|---|---|---|---|
| `001` | `:28` | `geom GEOMETRY(POINT, 4326)` in the table body | `stops`→`zones` |
| `002` | `:2` | `CREATE INDEX … GIST(geom)` | `stops`→`zones` |
| `003` | `:19` | `AddGeometryColumn('location_history','geom',4326,'POINT',2)` | `location_history` |
| `003` | `:52` | `AddGeometryColumn('stops','geom',4326,'POINT',2)` | `stops`→`zones` |
| `003` | `:57-59` | backfill `geom` from lat/lng | `stops`→`zones` |
| `003` | `:97-116` | `update_location_geom()` + `trg_location_geom` BEFORE INSERT trigger | `location_history` |
| `004` | `:75-77` | seed backfill of `geom` | `stops`→`zones` |
| `005` | `:18` | `ALTER INDEX idx_stops_geom RENAME TO idx_zones_geom` | `zones` |

`grep -rni 'linestring\|polyline\|makeline\|ST_LineLocatePoint\|ST_LineSubstring\|LineMerge'
backend/ --include='*.sql' --include='*.go'` returns **nothing**. Exit code 1.

**Answering the question directly: the `routes` table has no polyline geometry column at
all. The only geometries in this database are the zones' `POINT`s and `location_history`'s
`POINT`s.**

### 2.3 The Go model has no place for it either

`backend/internal/models/route.go:3-7`:

```go
type Route struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
```

`backend/internal/database/routes.go:19`:

```go
query := `SELECT id, name, description FROM routes`
```

Nothing reads or writes route geometry, because there is none to read.

### 2.4 The route creator never persists the polyline it already computes

This is the part worth understanding before anyone estimates the fix, because the data is
*almost* there.

`frontend/src/services/api/osrm.ts:16-32` calls the public OSRM demo server and gets back a
real, road-snapped GeoJSON LineString:

```ts
const url = `${OSRM_BASE_URL}/${coordinates}?overview=full&geometries=geojson`;
const response = await axios.get(url);
const coords = response.data.routes[0].geometry.coordinates as [number, number][];
return coords.map(c => [c[1], c[0]] as [number, number]);   // flipped to [lat,lng]
```

`frontend/src/components/Map/RouteCreatorMarkers.tsx:45-77` stores that result in local
component state and renders it:

```ts
const [pathGeometry, setPathGeometry] = useState<[number, number][]>([]);
// …
const snappedPath = await getRouteGeometry(rawPoints);
if (isMounted) setPathGeometry(snappedPath);
```

**And then discards it.** The POST body carries only points:

```go
// backend/internal/models/route.go:10-21
type CreateRouteRequest struct {
	Name        string            `json:"name" validate:"required"`
	Description string            `json:"description"`
	Zones       []CreateZoneInput `json:"stops" validate:"required,min=2"`
}
type CreateZoneInput struct {
	Name      string  `json:"name" validate:"required"`
	Latitude  float64 `json:"latitude" validate:"required"`
	Longitude float64 `json:"longitude" validate:"required"`
}
```

`RouteRepository.Create` (`backend/internal/database/routes.go:40-88`) inserts a row into
`routes` with `(id, name, description)` and one `POINT` per zone. The snapped polyline —
the exact geometry `ST_LineLocatePoint` would need — is fetched, drawn on screen, and thrown
away on unmount.

### 2.5 A second, independent gap: trips don't carry their route

Even if the geometry existed, the query that feeds the ETA could not select which polyline to
project onto. `backend/internal/models/trip.go:6-13`:

```go
type TripWithLocation struct {
	TripID     string
	DeviceID   string
	DeviceName string
	Latitude   float64
	Longitude  float64
	Speed      float64
}
```

No `RouteID`. `TripRepository.GetActiveTripsBeforeZone`
(`backend/internal/database/trips.go:19-40`) selects
`t.id, t.device_id, d.name, lh.latitude, lh.longitude, lh.speed` — the `trips.route_id`
column exists in the schema but is never selected. Any implementation would have to widen
this struct and its query too.

### 2.6 Verdict

**Phase 4.5 cannot proceed.** It is blocked on absent schema, absent persistence, and an
absent model field — three separate pieces of work, none of which is "route-projected
distance" itself.

---

## 3. The SRID / units question — answered, because the answer outlives the blocker

Asked explicitly, and it is the single most likely way an eventual implementation goes wrong
silently, so it is recorded here.

**`ST_Length(geometry)` in SRID 4326 returns degrees, not metres.** A result near `0.0234`
would be interpreted as kilometres or metres by calling code and produce an ETA that is
wrong by roughly five orders of magnitude — while looking like a plausible small number. It
would not error.

The cast is required:

```sql
ST_Length(ST_LineSubstring(r.geom, f_device, f_zone)::geography)   -- returns METRES
```

Three rules for whoever implements this:

1. **`ST_LineLocatePoint` takes `geometry`, not `geography`.** It returns a dimensionless
   fraction 0–1, so it is unaffected by the units problem — do **not** cast its arguments.
2. **`ST_LineSubstring` also operates on `geometry`.** Cast its *result* to `::geography`
   before `ST_Length`, not its input.
3. **The existing code already establishes this convention** and should be matched, not
   re-invented. `backend/internal/database/locations.go:155-156` carries the comment
   explaining it, and `:164-174` and `backend/internal/database/zones.go:74-82` both cast to
   `::geography` for `ST_DWithin`/`ST_Distance`. Metres is the established unit at this
   boundary; `geo.Haversine` returns **kilometres**, so any new repository method returning
   metres must be divided by 1000 before it meets `CalculateETA`'s existing contract, or the
   units diverge at the seam.

Also note: `ST_LineLocatePoint` requires a `LINESTRING`, not a `MULTILINESTRING`. OSRM can
return a discontinuous geometry for some waypoint sets; whatever persists it must either
enforce `ST_LineMerge` or reject the multi-part case, or `ST_LineLocatePoint` will error at
runtime on exactly the routes that are hardest to debug.

---

## 4. What unblocking would require — scoping input only, not an instruction to build

Four pieces, in dependency order. This is deliberately not broken into commits, because
whether to do it at all is a scope decision, not an execution detail.

**(a) Schema.** A new migration — next number is **`006`** (`005_rename_stops_zones.sql` is
the current highest; verified by `ls backend/migrations/`) — adding
`geom GEOMETRY(LINESTRING, 4326)` to `routes`, plus a GIST index. Nullable, because every
existing route has no geometry and `004`'s three seeded routes never will unless someone
backfills them.

**(b) Persistence.** `CreateRouteRequest` gains a polyline field; `createRoute.ts` sends the
`pathGeometry` that `RouteCreatorMarkers.tsx` already computes; `RouteRepository.Create`
writes it via `ST_GeomFromGeoJSON` or `ST_MakeLine`. **This changes the public API request
contract** — which `CLAUDE.md` §6.3's "zero behavior change" rule governed for Phase 2 but
which no rule covers here, so it needs an explicit decision.

**(c) Model widening.** `TripWithLocation` gains `RouteID`; `GetActiveTripsBeforeZone`
selects `t.route_id`. Mechanical, but it touches `trips.go`, which `CLAUDE.md` §5.5 currently
scopes **out** of Phases 0–4.5. That guard would have to be lifted explicitly.

**(d) Seed data.** The three seeded routes (`004_seed_data.sql:23-27`) have no geometry.
Without a backfill, every seeded route takes the Haversine fallback and the new path is never
exercised outside a hand-created route — meaning the phase's own headline feature would be
invisible in the demo. But `004` must not be edited (§6.3), so this needs its own migration
or a fixture change.

**Honest assessment of the ordering problem:** §6.5 budgets Phase 4.5 at 1 day and says "No
new dependencies. PostGIS is already running." That is true of the *algorithm*. It is not
true of the *data*: (a)–(d) are the majority of the work, and (b) and (c) each cross a line
that an existing scope guard draws. The 1-day estimate was written against an assumed schema.

---

## 5. What must not be touched — unchanged by this blocker, and still binding

Recorded because they will still apply whenever this phase does run.

| Item | Why |
|---|---|
| **`DefaultSpeed = 20.0`** (`backend/pkg/geo/distance.go:8`) and the `speedKmh < MinSpeedKmh` branch at `:31-33` | §6.5 is explicit: *"The ETA time model is unchanged: still `distance / speed`. Only distance improves. Do not touch `DefaultSpeed` here."* §3.5 lists the stopped-device fallback as an **accepted** limitation to document honestly, not fix. `TestCalculateETA` (`backend/pkg/geo/distance_test.go:23`) must pass **unmodified** |
| **`hub.Message`** — flat, nine keys, no `omitempty` on numerics, `"LOCATION_UPDATE"` literal, no `heading` | DEFECT-2 / DEFECT-5. `message_test.go` pins the serialized shape and the absence of `heading` |
| **`hub.Run`'s `ctx.Done()` drain**, `drainAndClose`, `clientDrainTimeout`, `clientDrainPollInterval` | D28. Untouched by anything in this phase |
| **Phase 2's naming** — `zones` table, `models.Zone`, `ZoneRepository`, `GeofenceService`, `GeofencePrediction`, `GetPredictionsForZone`, `GetActiveTripsBeforeZone` | Landed at `d6a4728`…`de57309`. Any Phase 4.5 work builds on these names; it does not revisit them |
| **HTTP paths and JSON tags** — `/api/arrivals`, `stop_id`, `json:"stops"`, `json:"stop_count"`, `json:"stop_ids"` | Deliberately kept during Phase 2 as wire contract. Item (b) above would break this deliberately and needs its own decision |
| **Migrations `001`–`005`** | §6.3: never edit an applied migration |

---

## 6. The decision this document hands back

Three options, stated without a recommendation disguised as a plan:

1. **Do the prerequisite work (a)–(d), then Phase 4.5.** Honest scope is well beyond §6.5's
   1 day, and (b)/(c) each require lifting an existing scope guard.
2. **Cut Phase 4.5 and correct `CLAUDE.md` §3.5.** Its limitations table currently claims
   both straight-line distance and proximity-⇒-approaching are "**FIXED in Phase 4.5**". If
   the phase does not run, that table asserts a fix that does not exist — which is anti-goal
   #2 ("do not present architecture the code does not execute") in miniature, sitting in the
   contract file. The two rows would move to "accepted limitation, documented in README",
   joining the two already there.
3. **Do a reduced version: direction only, no route projection.** The passed-the-zone defect
   is arguably the more visible of the two, and detecting it *may* not need a polyline —
   `trips.current_stop` versus `zones.sequence_number` already encodes ordering along a
   route, and `GetActiveTripsBeforeZone` already filters on `t.current_stop < $1`. Whether
   that is sufficient was **not** investigated here, because it is not what §6.5 specifies
   and designing it would be designing around the blocker. Flagged only so the option is not
   lost.

**Whichever is chosen, `CLAUDE.md` §3.5 and §6.5 must be updated in the same commit as the
decision.** Leaving §6.5 describing an implementation against a schema that does not exist is
how §7.2 came to specify a `heading` field that existed nowhere — the repo's documented
failure mode, and the reason the verification step that produced this report exists.

---

## 7. New findings — append to `IDEAS.md` (next free number is D41)

**D41 — `routes` has no geometry column; Phase 4.5 is blocked on it.**
`migrations/001_create_tables.sql:13-18` defines `routes` as `(id, name, description)`. No
migration adds a `LINESTRING`; `grep -rni 'linestring\|polyline\|ST_LineLocatePoint'` over
`backend/` returns nothing. `CLAUDE.md` §6.5 specifies an algorithm requiring
`ST_LineLocatePoint(route_geom, …)` against a column that does not exist, and §3.5 already
records the two defects it would fix as "FIXED in Phase 4.5". Unblocking needs four separate
pieces: a `006` migration adding the column, a persistence path, `TripWithLocation.RouteID`
(which means touching `trips.go`, scoped out by §5.5), and a seed backfill (which means a new
migration, since `004` must not be edited). See `docs/phase-4.5-execution.md`.

**D42 — the OSRM road-snapped polyline is computed on every route creation and discarded.**
`frontend/src/services/api/osrm.ts:16-32` fetches a full GeoJSON LineString from the public
OSRM demo server; `RouteCreatorMarkers.tsx:45,72-74` holds it in `useState` and renders it;
`createRoute.ts:12` POSTs only `{name, description, stops[]}` because `CreateRoutePayload`
has no geometry field. So the exact data Phase 4.5 needs is already being produced, once per
route creation, and thrown away on unmount. Cheapest possible unblock for D41(b) — but it
changes the `POST /api/routes` request contract.

**D43 — `Route.pattern` is a frontend ghost field (DEFECT-3 class).**
`frontend/src/types/domain.ts:38` declares `pattern?: GeoJSON.LineString`, and
`RoutePolyline.tsx:13-15` maps `route.pattern.coordinates` to render the route path — with
`MapContainer.tsx:56` mounting `<RoutePolyline />` on every map render. The backend
`models.Route` (`route.go:3-7`) has no such field and `RouteRepository.GetAll`
(`routes.go:19`) selects only `id, name, description`, so `pattern` is **always** undefined,
`getPositions` always returns `[]`, and the component always renders nothing.
`config/apiConfig.ts:107` and `wsConfig.ts:62` both document a `pattern` field the backend
never sends. This is the same shape as DEFECT-3 (a UI element with no data behind it) and the
same shape as D13 (`heading`). Not fixed here — and note that it would be *resolved*, not
deleted, if D41/D42 were ever done, since persisting the polyline is exactly what would make
this component work.
