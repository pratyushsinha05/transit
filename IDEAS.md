# IDEAS.md — deferred work

Each entry: what it is, why it's deferred, where it's discussed. Per `CLAUDE.md` §5.4: if a
change would make the code better structured/tested/honest it's in scope; if it would make
the product smarter or more featureful, it belongs here.

---

## From CLAUDE.md §5 — explicit scope guards

### Algorithms (§5.1)
- Map-matching, Kalman/particle filtering, event-time processing — no prediction-accuracy
  work beyond Phase 4.5's route-projected distance.
- Distributed-systems papers (CRAQ, Count-Min Sketch, Prequal, HyperLogLog, DDSketch) — these
  belong to `llm-router`, not here. Duplicating them here adds zero signal.
- A new/different routing engine — the existing frontend OSRM client
  (`frontend/src/services/api/osrm.ts`) is neither extended nor removed; see §5.5.

### Product features (§5.2)
- Multi-tenancy, tenant isolation, API keys, quotas, rate limits
- Webhooks with retry semantics, versioned public API contracts, SLAs
- Trip/task/journey lifecycle state machines, waypoint ordering, shared pooling
- Mobile SDKs, battery-aware tracking frequency modulation
- Ticketing, payments, user accounts, notifications
- Historical analytics dashboards beyond what already exists

### Infrastructure (§5.3)
Terraform, GKE, Istio, Helm, ArgoCD, Prometheus, Grafana, k6, chaos engineering — Phase 5+,
not started until Phases 0–4.5 pass their gates.

---

## From BASELINE.md — defects found, not fixed in Phase 1

**D4 addendum — `hex_res9` column name vs. configurable resolution.** Phase 1 (C7, deferred
to Phase 3 per the estimate cut) wires `H3_RESOLUTION` for real. Once it's live, setting
`H3_RESOLUTION=7` stores resolution-7 hexes in a DB column literally named `hex_res9`.
Renaming the column is a migration and a behavior question (does changing resolution
mid-flight require a backfill?) — out of scope for the wiring fix itself.

**D5 — `LOG_LEVEL` / `REDIS_POOL_SIZE` are dead.** `cfg.LogLevel` is never read; logging is
`log.Printf` plus Echo's default, not structured or levelled as §7.1 claims.
`cfg.RedisPoolSize` is never read; `cache/redis.go:26` hard-codes `PoolSize: 50`. Real
structured, levelled logging is bigger than a Phase 1 wiring fix — it's a logging-library
decision. Flagged as the one open item at the Phase 0 gate; needs an owner decision
(implement vs. soften the §7.1/§7.4 claim) before Phase 2.

**D8 — `GET /api/stops` cannot return all stops.** `handlers/stops.go` hard-requires
`route_id` and 400s without it. `StopRepository.GetAll` (`database/stops.go:100`) is
written, correct, and unreachable — no handler calls it. The frontend has wanted an
all-stops endpoint since it was written (`services/api/stops.ts` makes `routeId` optional).
Needs a decision: add a handler branch for "no route_id → GetAll", or a separate endpoint.
Not fixed in Phase 1 — `useStops.ts` was reverted to its working `'route-101'` form instead
of building the real fix under time pressure disguised as a two-line hook change.

**D12 — `messageHandler.ts` `handleArrivalUpdate` is a dead, inconsistent path.** It builds
an ad-hoc arrival object (`stopId`, `busId`, `eta`, `status`, `route`, `timestamp`) from a
WebSocket `ARRIVAL_UPDATE` message type the backend never emits — `hub/message.go` only
defines `MsgTypeLocationUpdate`. It calls `updateArrival(stopId, arrival as any)`, so the
`as any` cast let it survive the C2 `Arrival` type rewrite (now `tripId`/`deviceId`/
`etaMinutes`/...) without a compile error, but it would push objects with no `tripId` into
the store if it ever ran. Inert today; revisit when/if a WS arrival-style event exists —
Phase 4's `GEOFENCE_EVENT` is the closer real analog, not this shape.

**D13 — `BusLocation.heading` and `BusMarkers.tsx`'s HEADING popup row are a permanent
ghost.** `heading` was never real — always defaulted to `0` — but until the CLAUDE.md §7.2
correction there was at least a live plan (the original, wrong §7.2) to eventually populate
it. That plan is now explicitly cancelled: `heading` does not exist in `models.Location`,
the DB, or the ingest payload, and per the corrected §7.2 it never will. This is the same
shape as DEFECT-3 (a UI element with no real data behind it) but wasn't fixed alongside the
WS envelope commit since it wasn't part of what was asked there. Fix: either make
`BusLocation.heading` optional and drop the popup row (DEFECT-3-style), or, if heading ever
becomes a real requirement, compute it client-side from consecutive position deltas — the
only place data for it could plausibly come from without adding a sensor input the devices
don't have.

**D10 — `models.Trip` is dead code.** Zero references anywhere (confirmed via
`grep -rn "models\.Trip\b"`). Unlike `models.Device` (kept — see BASELINE.md §2),
`ArrivalEvent` in the same file is live and would need splitting out first. Low priority;
revisit during Phase 2 rename since `trip.go` will need eyes anyway if trips ever get
renamed away from transit vocabulary.

**D11 — migrations may double-apply seed data on a fresh volume.**
`infra/docker-compose.yml` mounts `backend/migrations` into
`/docker-entrypoint-initdb.d`, so Postgres runs every `.sql` file at first init. The Go
binary then replays the same files through its own `schema_migrations` ledger, which starts
empty. Migrations 001–003 are idempotent (`IF NOT EXISTS` / guarded `DO $$` blocks);
`004_seed_data.sql`'s three `INSERT INTO location_history` statements have no
`ON CONFLICT`, so seed pings get duplicated. Not touched — fixing it means editing an
existing migration, which §6 forbids, or restructuring the docker-entrypoint mount, which is
an infra change out of scope until Phase 5+.

**Cosmetic — `routes.go:52`.** `string(rune('1'+i))` in the stop-validation error message
garbles for `i > 8` (more than 9 stops). Error text only, not a functional bug.

## From Phase 1 estimate cut

**C7 — wire `H3_RESOLUTION`.** `cfg.H3Resolution` is read/validated and consumed by
nothing; both constructors hard-code resolution 9 even though
`NewGeofencingServiceWithResolution` / `NewLocationRepositoryWithResolution` already exist.
Deferred to Phase 3 to keep Phase 1 inside its estimate — see the plan's *Estimates*
section. Two-line fix in `main.go` when picked back up.

**C8 — give the hub a shutdown path.** `Hub.Run()` is an unbounded `for/select` with no
`done`/`ctx` case, started fire-and-forget. §7.1 requires an owner and clean drain. Deferred
to Phase 3, where it's needed anyway for the slow-consumer and disconnect-mid-broadcast hub
tests — landing it there avoids doing the work twice.

## From the Understanding Pass — Pass 1 partition

Found while partitioning the tracked file list for the read-only understanding pass. Both
are housekeeping, neither blocks any phase. Recorded, not fixed — the pass is read-only.

**U1 — a tracked file literally named `=` at the repo root.** Six bytes, ASCII, contents
`31.3.2`. A shell-redirection accident (`... >= 31.3.2` without quoting) that got committed
and has survived since. It is not referenced by anything. Verified with `git ls-files` and
`file =`. Deleting it is a one-line `git rm`, but that is a tree change and this pass does
not make them.

**U2 — `frontend/src/utils/` is an empty orphaned directory.** DEFECT-3 deleted
`h3Helpers.ts` (confirmed: `find . -name 'h3Helpers*'` returns nothing, and `git ls-files
frontend/src/utils/` is empty), but the now-empty directory was left on disk. Git does not
track empty directories, so it is invisible to `git status` and will not appear in a fresh
clone — it only exists in working trees that predate the deletion. Harmless; noted so a
future reader does not mistake it for a missing module.

**U3 — the zone spec used to drive this pass did not cover the SPA entry point.**
`frontend/index.html`, `frontend/src/main.tsx`, and `frontend/src/App.tsx` matched none of
the six zone patterns and were only read because the partition step diffed its zones against
`git ls-files` and routed the remainder to a catch-all. This is the same failure mode that
caused the prior audit to miss trips, route-creator, and OSRM: zone boundaries drawn from an
idea of the repo rather than from its file list. Not a code defect — a process note worth
keeping, since the next pass that partitions this repo should diff against `git ls-files`
first and dispatch second.

## From closing DEFECT-1 properly

Found while moving `TripWithLocation` out of `database` and into `models`. None of these were
touched by that commit; recorded here and left alone.

**D14 — the app's only WebSocket is opened as a side effect of rendering
`ConnectionStatus.tsx:11`.** Unmounting the header component kills the live feed.
Architectural smell; no fix scheduled.

**D15 — `GetActiveTripsBeforeStop` takes `stopSequence` but never scopes by route.** Every
`IN_PROGRESS` trip system-wide with `current_stop < N` matches, regardless of route. Invisible
with single-route seed data; wrong the moment a second route exists. Fix is a route filter —
Phase 4 scope.

**D16 — `models.ArrivalEvent` (`models/trip.go:11`) is dead:** declared, never constructed,
never returned. The two other grep hits are the substring inside `DetectArrivalEvent`.
`BASELINE.md` §3.1 wrongly lists it as the `/api/arrivals` response type; that is
`services.ArrivalPrediction`.

**D17 — two files fail `gofmt -l` at HEAD, pre-dating any current work:**
`internal/models/route.go` (struct-tag alignment in `CreateRouteResponse`, lines 22–30 —
`StopIDs []string` widened the column and the other four tags were never re-aligned) and
`pkg/geo/distance_test.go:27` (trailing whitespace). Trivial, but nothing enforces gofmt in
this repo — no pre-commit hook, no CI check. The formatting is a symptom; the missing gate is
the actual gap. Both belong in a `chore: gofmt` commit, and the gate itself is Phase 5 CI
scope.

## From closing DEFECT-7

**D18 — `004_seed_data.sql` seed pings are not stable across applies.** The three
`location_history` inserts use `NOW() - INTERVAL 'N minutes'` for their timestamps, so every
apply produces different `time` values for the same logical row. **No unique constraint could
make these seeds idempotent** — the row identity itself changes between applies, so
`ON CONFLICT` would never fire even with a key in place. That is why DEFECT-7 was closed with
an `IF NOT EXISTS (SELECT 1 FROM location_history)` guard instead. Two further constraints on
any future fix: TimescaleDB requires that a unique index on a hypertable **include the
partitioning column**, so `time` must appear in any key; and replacing the relative
timestamps with literals would make the seeds deterministic but push every row outside the
5-minute max-age filters at `geofencing.go:144` and `geofencing.go:155`, so
`/api/nearby/buses` would return nothing against seed data. Recorded, not scheduled.

**D19 — `docker compose up -d` silently tests a stale backend image.**
`backend/Dockerfile:60` bakes the migrations into the image
(`COPY --from=builder /app/migrations /app/migrations`), while
`infra/docker-compose.yml:23` mounts the same directory live into Postgres'
`/docker-entrypoint-initdb.d`. So a plain `docker compose up -d` runs the **host** copy of a
migration in Postgres and the **image** copy in the Go binary. Verifying DEFECT-7 hit exactly
this: the local image was five months old (built 2026-03-27) and carried a `004_seed_data.sql`
with `NULL` hex values, so the guard appeared not to work and the count came back 22 instead
of 11. `make docker-up` is safe because it depends on `build-docker`; bare `compose up -d` is
not. Any future migration verification must rebuild first
(`docker compose build backend`, or `make rebuild-backend`). Recorded, not scheduled — the
real fix is to stop mounting migrations into the Postgres init directory at all, which is the
Phase 5 restructure DEFECT-7 deliberately left alone.

## From Phase 1A execution

**D26 — `.gitignore` has bare `server` matching `backend/cmd/server/`.** Line 6 of `.gitignore`
contains `server` without a leading slash, matching any file or directory named `server`
in the tree. As a result, `git add backend/cmd/server/main.go` is rejected unless `-f`
is passed, even though `main.go` is already tracked. Fix is to change `.gitignore:6` to
`/server` so it only matches the compiled root binary.

## From Phase 1B execution

**D27 — H3 LatLngToCell does not error on out-of-bounds coordinates.**
`h3.LatLngToCell` wraps coordinates rather than rejecting latitudes outside `[-90, 90]`
or longitudes outside `[-180, 180]`. Spatial boundary validation must remain strictly
enforced at the HTTP handler layer (`handlers/location.go:37`).

**D28 — Hub shutdown discards buffered client messages. FIXED.**
The original write-up of this defect misdiagnosed the mechanism: closing a Go channel does
not discard values already buffered in it — a receiver still drains every buffered value
with `ok == true` before observing the close. Verified directly (`close()` on a filled
buffered channel, then `for range`, receives every value). So `Client.WritePump` does not
jump to its close-frame branch while messages remain queued.

The real loss was one level up: `Hub.Shutdown()` returned the instant `h.done` closed, which
happened immediately after `Run` closed every client's `Send` channel on `ctx.Done()` — it
never waited for each client's `WritePump` goroutine to actually consume the backlog. In
`cmd/server/main.go`, `wsHub.Shutdown()` is the last statement before `main` returns, so the
process could exit with write pumps still mid-flight, losing whatever they hadn't yet read.

Fixed in `internal/hub/hub.go`: on `ctx.Done()`, `Run` snapshots and clears the client set,
then drains each client in parallel (`drainAndClose`, one goroutine per client via
`sync.WaitGroup`) — polling `len(client.Send) == 0` every `clientDrainPollInterval` (1ms),
bounded by `clientDrainTimeout` (250ms) — before closing that client's channel. A client
that hasn't drained within the bound has its remainder counted into `DroppedMessages` and
logged, then closed anyway. Worst-case shutdown time is `clientDrainTimeout` regardless of
client count, since all clients drain concurrently. `Shutdown()`'s signature and `main.go`'s
call site needed no change. `TestHubShutdown` (the DEFECT-6 exit gate) passes unmodified.
New regression tests: `TestShutdownDrainsBufferedMessages` (an actively-read client receives
everything before close) and `TestShutdownDropsAfterDrainTimeout` (a never-read client is
dropped only after the timeout, proving `Shutdown()` waits rather than returning instantly —
confirmed to fail against the pre-fix code with `Shutdown()` returning in ~7µs instead of
≥250ms). Coverage of `internal/hub` rose from 87.0% to 86.1% (originally claimed 90.8%, corrected in D40).

**D29 — Gorilla WebSocket ReadPump unexpected close error handling.**
`client.go:54` excludes `CloseGoingAway` and `CloseAbnormalClosure`, but not
`CloseNormalClosure` (1000). Clean peer disconnects can cause `log.Printf` error noise.

**D30 — Hub package statement distribution requires client pump coverage for 50%.**
`internal/hub` has 52 statements: 25 in `hub.go` and 27 in `client.go`. 100% coverage
of `hub.go` alone reaches at most 25/52 = 48.07%, failing the ≥50.0% package gate.
Reaching the target requires exercising `client.go` (`WritePump`/`ReadPump`).

**D31 — `ReadPump`'s deferred `Unregister` send can block forever after hub shutdown.**
Found while implementing the D28 fix. `Hub.Unregister` (`hub.go:31`) is unbuffered and only
`Run`'s `select` loop ever reads it. Once `Run` returns (post-`ctx.Done()`, after D28's drain
completes), nothing reads `Unregister` again. `Client.ReadPump`'s deferred
`c.Hub.Unregister <- c` (`client.go:45`) then blocks forever if a connection error fires
during or after shutdown. Low severity as observed: `main.go` calls `wsHub.Shutdown()` as
its last statement before returning, so the leaked goroutine outlives the process by at most
the time until `main` returns — Go does not wait for leaked goroutines on exit. Not fixed
here; out of scope for D28, which is about message loss, not goroutine leaks. If this ever
needs fixing, `Run` returning is the wrong signal to design around — the real fix is a
`select` with a `ctx.Done()` case inside `ReadPump`'s own send, or draining `Unregister` in a
second goroutine alongside `drainAndClose` until all `ReadPump`s have exited.

## From Phase 1C execution (integration test)

**D32 — production Postgres image is amd64-only and 3.5 years stale.**
`infra/docker-compose.yml:12` pins `timescale/timescaledb-ha:pg15-latest`. Docker Hub
registry API reports a single-platform manifest, `architecture: amd64`, `tag_last_pushed
2023-04-24T08:00:26Z`. On this arm64 development machine it runs under Docker Desktop's
Rosetta-based emulation — confirmed via `docker image inspect …
--format '{{.Architecture}}'` returning `amd64` against a host reporting `arm64`
(`docker info --format '{{.Architecture}}'` → `aarch64`). In practice the emulation cost was
far smaller than expected (~3s container-ready time, not the 20-60s budgeted), but the
staleness is still real: multi-arch tags already exist in the same repository (`pg15`,
`pg15-ts2.28`, `pg15.18-ts2.28.3` all report `['amd64','arm64']`). Changing the pin is an
infrastructure change (`CLAUDE.md` §5.3, Phase 5+); recorded, not scheduled.

**D33 — `D21` is cited in `CLAUDE.md` but has no `IDEAS.md` entry.**
`CLAUDE.md` §3.1's Redis row cites "(D21)" for the write-only finding, and §4's "Tracked but
not scheduled" cites `D21` again. `IDEAS.md` has no `D21` — its identifiers jump D19 → D26.
The finding itself is real and independently re-verified while building the integration test
(`grep -rn "GetDeviceState\|GetDeviceLocation" backend/` still finds no external caller, and
`cache.CacheStore` — `cache/interface.go:18-27` — declares no read-back method). Either add
the entry or drop the citation. Same section, unrelated: the Redis row's last sentence is
duplicated verbatim — "Pool size hard-coded 50 in `cache/redis.go:28`; `REDIS_POOL_SIZE`
unused." appears twice in `CLAUDE.md:164`.

**D34 — adding a test-only dependency bumped two production dependencies' minimum versions.**
`go get github.com/testcontainers/testcontainers-go/...` (for the integration test) forced
`go.mod`'s minimum versions of `github.com/jackc/pgx/v5` (v5.4.3 → v5.9.2) and
`github.com/redis/go-redis/v9` (v9.0.5 → v9.7.3) upward, via Go's minimal-version-selection
resolving the *highest* version any module in the graph requires — testcontainers' postgres
and redis submodules depend on newer minimums of both than this repo's own production code
did. The `go` directive itself was also raised, 1.23.0 → 1.25.0. All of `go build ./...`,
`go vet ./...`, and the full unit suite (`go test ./... -race -count=1`) stayed green after
the bump, so nothing is known to be broken by it — but it means "testcontainers is test-only"
is not quite true at the `go.mod` level: the same two packages that appear in production
import paths (`database.New`, `cache.New`) now resolve to newer minor/patch versions than
before this change, even though no production source line changed. Worth a deliberate look
before the next dependency audit, not a defect to fix now.

## From Phase 2 execution (domain vocabulary rename)

**D35 — `LocationRepository.GetBusesInHex` is dead code.**
`backend/internal/database/locations.go:79`. Two occurrences total: the declaration and its
doc comment. It is absent from `services.LocationRepository` (`services/interfaces.go:25-30`,
which declares only `GetBusesInHexes` and `GetBusesNearStop`) and is called nowhere. Phase 2
renames it to `GetDevicesInHex` mechanically rather than deleting it, because deletion is a
scope decision, not a rename. Decide separately whether the single-hex variant should exist
at all.

**D36 — `cache.NearbyBusesTTL` is dead.**
`backend/internal/cache/interface.go:36`. Declared, never read — the only TTL actually used
is `DeviceLocationTTL` (`redis.go:57`). Same treatment as D35: renamed to `NearbyDevicesTTL`,
not deleted.

**D37 — `CLAUDE.md` §6 was stale in five distinct ways before Phase 2 began.**
Recorded because the pattern matters more than the individual errors: (a) the
`models.ArrivalEvent` row names a type that no longer exists; (b) `busHex` in the locals row
never existed; (c) the `006` migration number was a guess that the check disproved — it is
`005`; (d) §6.1 omitted ~22 in-scope identifiers, including all of `StopsService`,
`ArrivalPrediction`, and `IsAtStop`; (e) §6.2 lists only renamed files and omits the 16
frontend files needing reference updates, one of which (`ui.ts`, 39 matches) has more matches
than any file it does list. Also, §6.3's stated verify command greps `backend/internal
backend/pkg` and omits `backend/cmd`, where main.go holds 13 matches.

**D38 — `IDEAS.md` D10 and D16 are stale.**
D10 says `models.Trip` is dead code; D16 says `models.ArrivalEvent` is dead at
`models/trip.go:11`. Both types have since been removed — `models/trip.go` now contains
only `TripWithLocation`. "Dead" and "absent" need different follow-ups; these entries should
be closed, not carried.

**D39 — `trips.current_stop` keeps transit vocabulary after Phase 2.**
`migrations/001_create_tables.sql:51`. Renaming it would mean editing `004_seed_data.sql`'s
INSERT column list, which §6.3 forbids, so Phase 2 leaves it. After this phase the schema is
mixed: a `zones` table alongside a `current_stop` column. Resolve in whichever phase next
touches the schema.

**D40 — internal/hub coverage is nondeterministic.**
Measured 86.1% in 4 of 5 consecutive runs, 87.3% in 1. The ~1.2pt swing is one branch, almost
certainly drainAndClose's fast-path vs timeout-path, which depends on goroutine scheduling at
shutdown. Any coverage gate on this package must use the observed MINIMUM (86.1%), never a
best-observed figure — gating at 87% would fail 4 runs in 5.
Separately: the D28 report claimed 90.8% and described it as "stable at repeated
measurement." That figure does not reproduce and was never achievable. It propagated into
the Phase 2 report as an inherited baseline before being caught. Same failure class as this
project's citation errors, in a number rather than a line reference.


