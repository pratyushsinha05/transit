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
