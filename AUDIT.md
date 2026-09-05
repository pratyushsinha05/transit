# AUDIT.md — Transit: the self-directed audit and defect record

This repository documents its own mistakes with evidence. That is the point of this file.

Three times, a claim in this codebase's documentation was not checked against the code, and
three times the code diverged from what the documents said it did. Once, the specification
itself described a field that existed nowhere. This file records what was found, how it was
found, why it was missed, and what it cost.

> **Every claim here carries a `file:line` or a command.** Where a prior audit document and
> the source disagreed, the source won — and where that happened, it is recorded as a finding
> rather than quietly corrected, because the pattern of *how* documents drift is itself the
> subject.

---

## 1. Method

A self-directed adversarial read of the repository, executed 2026-07-03 by a sub-agent swarm
against a fixed protocol (`VERIFICATION_LOG.md`). The repository was partitioned into seven
analytical zones along topological boundaries, each read independently, with findings
cross-checked between zones before synthesis.

| Zone | What it proved |
|---|---|
| 1 — Operations | `deploy.sh` sequentially runs `make docker-up` and `npm run build`; compose mounts PostGIS and Redis on a shared bridge |
| 2 — Backend architecture | Cross-checked `main.go` DI wiring. **Discovered and proved `ArrivalsService` is instantiated but ignored** |
| 3 — Algorithm | Independently traced the Haversine math; extracted the exact k-ring logic (radius 3 at resolution 9) |
| 4 — Evals | Executed `make test-unit`. Log proved **6.9% coverage in services, 0% in handlers** |
| 5 — Frontend architecture | Compared `package.json` against `graphify.html`. **Proved the doc claimed Redux Toolkit; the codebase uses Zustand** |
| 6 — Frontend UI & state | Traced `handleMessage`. Found the data-contract discrepancy on `h3_hex` / `route_id` |
| 7 — Frontend components | Found the phantom UI: the panel toggles an H3 grid layer the map has no code to render |

A later mechanical pass (`00-inventory.md`) re-derived the same territory from nine per-zone
extraction reports, with **no source file opened during writing**, so every claim routed
through a citable intermediate. That method is what surfaced §5's contradiction catalogue.

**The verdict from that audit, in its own words:** the repository "presents itself
structurally as a heavily-abstracted Enterprise Clean Architecture application… However, in
execution, it violates its own architecture for expediency." A fast, highly-concurrent
telemetry pipeline dressed up as a complex Transit application — strong in deployment
simplicity and raw WebSocket throughput, weak in test coverage and in the drift between its
intended architecture and its actual execution flow.

---

## 2. The three documented drifts

### Drift 1 — the contract described a different project

An earlier version of `CLAUDE.md` specified: no database until persistence was concretely
needed, no HTTP router libraries, no frontend framework, no Leaflet abstraction layers.

Every one of those constraints was void by the time anyone noticed. The repo runs PostgreSQL
+ TimescaleDB + PostGIS + Redis, uses the Echo router, and ships a React 19 SPA with Zustand
and Tailwind. The constraints were not followed, were never formally revoked, and left the
file describing a project that did not exist.

### Drift 2 — architecture advertised, not executed

DEFECT-1, below. A service constructed, wired, and bypassed.

### Drift 3 — the specification fabricated a field

`CLAUDE.md` §7.2 once specified a nested envelope and a `heading` field, **both written from
audit documents rather than from `message.go`**, and Phase 1 began building against them
before the contradiction was caught. See §4.

All three happened for the same reason: a stated claim was not checked against the code.

---

## 3. The defect record

### DEFECT-1: handler bypasses service (architectural) — CLOSED at `5ab7448` (incomplete) → fully closed at `fe918e0`

- **Where:** turned out to be **five** handlers, not one: `handlers/arrivals.go`, `stops.go`,
  `location.go`, `routes.go` (each held a concrete `*database.X` repository) and `nearby.go`
  (which held a concrete `*services.GeofencingService` instead of an interface).
- **What:** `ArrivalsService` was constructed in `cmd/server/main.go` and then **ignored**;
  the arrivals handler reached directly into repositories and recomputed raw math itself. The
  other four had never gone through a service at all.
- **Consequence:** the k-ring approach-detection logic never executed. Dead code in the
  binary. The Clean Architecture the repo advertised was not the architecture it ran.
- **Fix:** every handler takes a service interface declared in `handlers` (the consumer
  package). Every service that touched a repository directly now takes a repository interface
  declared in `services`. Compile-time assertions live in `cmd/server/main.go` — the one place
  allowed to import both an interface's package and its implementation's package without
  inverting the layering.

**The first closure was incomplete, and this is the instructive part.**
`internal/services/interfaces.go` imported `transit-backend/internal/database` and declared
`TripRepository.GetActiveTripsBeforeStop` as returning `[]database.TripWithLocation` until
`fe918e0`. That inverts the dependency the interface exists to break: only
`database.TripRepository` could satisfy an interface the **services** package declares, so a
mock, a test fake, or any alternative backend would have had to import `database` to comply.
Every other method in that file already used `models.*`. Fixed by moving `TripWithLocation`
into `internal/models/trip.go`. The SQL and `rows.Scan` in `database/trips.go` are
byte-for-byte unchanged — a type-location change, not a behavior change.

**Why it was missed: the verify command defined its own scope.** The original was:

```
grep -rn "database\." backend/internal/handlers/
```

**Handlers-only, and therefore too narrow.** It tested one *symptom* of the defect rather than
the layering rule itself, so it could not detect the services-layer violation — and it
reported the defect closed while it was still live. The corrected form covers both layers:

```
cd backend
grep -rn "internal/database" internal/handlers/ internal/services/
```

which must return nothing. A check that defines its own scope is anti-goal 10; this is the
worked example.

### DEFECT-2 / DEFECT-5: the WebSocket contract — three envelopes, and a literal that matched nothing

- **Where:** `hub/message.go` vs `frontend/src/services/websocket/messageHandler.ts` and
  `frontend/src/types/domain.ts`.
- **What:** **three** envelopes were in play, not two — the backend's flat struct, the
  frontend's `bus_id`/`last_updated` expectation, and the (wrong) nested shape `CLAUDE.md`
  itself declared. The committed frontend used a lowercase `'location_update'` type literal
  against the Go constant `"LOCATION_UPDATE"`, so **no location message was ever processed**:
  `handleMessage` fell through to `default:` on every single frame.
- **DEFECT-5, same commit:** every numeric field carried `omitempty`. A stopped device — the
  most common real state — serialized with no `speed` key, and the frontend's
  `parseFloat(raw[schema.speed] || 0)` could not distinguish "stopped" from "not reported".
  `latitude`, `longitude` and `accuracy` had the same hazard at exactly `0`.
- **Fix:** the canonical envelope in `ARCHITECTURE.md` §5 — flat, nine fields, `route_id` and
  `h3_hex` added backend-side, `omitempty` removed from every numeric field. Both defects were
  fixed in one commit because the serialization test would otherwise have encoded the bug.
- **Verify:** `hub/message_test.go` asserts the serialized JSON shape — nine keys, no `data`
  wrapper, and `speed` present as `0` rather than absent.

### DEFECT-3: ghost UI — CLOSED at `066b129`

- **Where:** an "H3 Spatial Grid" toggle in `components/Sidebar/OperationsPanel.tsx`, against
  `components/Map/MapContainer.tsx`, which read no layer state at all.
- **What:** the toggle toggled nothing. No grid layer existed. `frontend/src/utils/h3Helpers.ts`
  was independently dead — zero call sites, and all three exports operated on `bus.h3Hex`,
  which the backend did not send at the time.
- **Fix as applied:** the toggle, the `grid` key in `store/slices/ui.ts`, and `h3Helpers.ts`
  were all deleted. The grid layer was **not** implemented to justify the toggle — that would
  have been scope inflation.
- **Verify:** `grep -rn "grid" frontend/src/store/slices/ui.ts` returns nothing;
  `frontend/src/utils/` is empty; the operations panel has exactly two layer toggles.

**Why it was missed: it was not missed. It was reported three times and not actioned.**
This section described work already done for three review rounds. Its staleness was reported
by Pass 1 §7.2(f), by the Sections 6–7 checker, and by the first citation gate — three
separate times — before anyone acted on it. Its old citations
(`OperationsPanel.tsx:59-68`, `ui.ts` lines 36/38/61) had by then drifted onto unrelated code.
**A finding reported three times and never actioned is its own failure**, distinct from the
defect it describes.

### DEFECT-4: test coverage

Baseline: `pkg/geo` 100%; `internal/services` 6.9%; everything else 0%. No integration tests.
See §6 for the before/after.

### DEFECT-6 → D28: the hub could not be shut down, then could not drain

**DEFECT-6, as found:** `Hub.Run()` was an unbounded `for { select {…} }` over three channels
with no `done`/`ctx` case, started fire-and-forget (`go wsHub.Run()`). Client goroutines were
likewise fire-and-forget. Graceful shutdown called `e.Shutdown` and nothing else. This
contradicted the stated convention that every goroutine has exactly one owner responsible for
its shutdown, and it blocked the slow-consumer and disconnect-mid-broadcast tests, because a
test cannot start a hub and then stop it cleanly.

**D28, and the misdiagnosis worth recording.** The defect was first written up as:

> When `Hub.Run` exits on `ctx.Done()`, it immediately closes all client `Send` channels.
> `Client.WritePump` handles channel closure by immediately sending a WebSocket close frame,
> potentially dropping messages that were buffered but not yet written.

**That mechanism is wrong.** Closing a Go channel does not discard values already buffered in
it — a receiver drains every buffered value with `ok == true` before observing the close.
Verified directly: fill a buffered channel with 8 values, `close()` it, then `for range` — all
8 are received. So `WritePump` never jumps to its close-frame branch while messages remain
queued.

**The loss was real, one level up.** `Hub.Shutdown()` returned the instant `h.done` closed,
which happened immediately after `Run` closed every client's `Send` channel. It never waited
for any `WritePump` to consume the backlog. And `wsHub.Shutdown()` is the last statement
before `main` returns — so the process exited with write pumps still mid-flight, losing
whatever they had not yet read. The channels were not the problem; **nothing waited for the
write pumps.**

**Fix:** `Run` snapshots and clears the client set on `ctx.Done()`, then drains each client in
parallel — one goroutine per client, joined by a `sync.WaitGroup` — polling
`len(client.Send) == 0` every 1 ms, bounded at 250 ms, before closing that client's channel.
Worst-case shutdown is 250 ms regardless of client count. `Shutdown()`'s signature and its
`main.go` call site needed no change, and `TestHubShutdown` passed unmodified.

The regression test that pins it, `TestShutdownDropsAfterDrainTimeout`, was **confirmed to
fail against the pre-fix code**, with `Shutdown()` returning in ~7 µs instead of ≥250 ms.
That confirmation matters: the companion test — "an actively-read client receives everything
before close" — passes against the *unfixed* code too, precisely because `close()` preserves
buffered values. A regression test that passes before the fix pins nothing.

### DEFECT-7: migrations apply twice on a fresh volume — CLOSED

- **Where:** compose mounted `../backend/migrations` into `/docker-entrypoint-initdb.d`, so
  Postgres ran every `.sql` at first init. The Go binary then replayed all of them through its
  own `schema_migrations` ledger, which starts empty.
- **What:** 001–003 are idempotent (`IF NOT EXISTS`, guarded `DO $$`). `004_seed_data.sql` was
  not — its three `INSERT INTO location_history` statements had no `ON CONFLICT`, so **seed
  pings were duplicated**. Routes, zones, devices and trips did use `ON CONFLICT`.
- **Why it was Phase 1 and not later:** duplicated rows in `location_history` corrupt every
  number the performance work would report.

**The prescribed fix could not work, for two independent reasons.** `ON CONFLICT DO NOTHING`
was specified without checking the schema:

1. **There is nothing to conflict against.** `location_history` has no `PRIMARY KEY`, no
   `UNIQUE` constraint, and no `ADD CONSTRAINT` in any migration; all its indexes are plain
   `CREATE INDEX`, and `create_hypertable` adds none. **Bare `ON CONFLICT DO NOTHING` is
   syntactically legal on such a table — that is the trap.** It compiles, the migration runs
   green, and it silently never fires. The targeted form `ON CONFLICT (time, device_id)` fails
   loudly instead, with *"no unique or exclusion constraint matching the ON CONFLICT
   specification"* — the worse-looking error is the more honest one.
2. **Adding a constraint would not have helped either.** Every seed row's timestamp is
   `NOW() - INTERVAL 'N minutes'`, evaluated per apply. Postgres init runs at T₀, the Go binary
   replays at T₁, and the same logical ping lands at `T₀−5min` then `T₁−5min` — a *different
   key*. No constraint can deduplicate rows whose identity changes between applies.
   (TimescaleDB additionally requires a unique index on a hypertable to include the
   partitioning column, so `time` must appear in any future key.)

- **Fix as applied:** wrap only the three `location_history` inserts in
  `DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM location_history) THEN … END IF; END $$;` —
  idempotent without touching the schema or the timestamp semantics. The six inserts that
  already carried `ON CONFLICT` were left alone; they have real primary keys and work.
- **Verify:** `docker compose down -v && up -d`, wait for the ledger replay, then
  `SELECT count(*) FROM location_history;` returns **11** (5 + 3 + 3), not 22. **A `go build`
  alone does not test this and must not be reported as if it did.**

The compose-level double-apply was later removed outright (D45): the mount is gone and
`main.go` is the single authority for schema migrations.

---

## 4. The `heading` field — a value the system could not produce

`CLAUDE.md` §7.2 once specified a `heading` field. It existed in no model, no column, and no
payload, because the section had been written from audit prose instead of from `message.go`.
Phase 1 began building against it.

Worse than the spec error was what the frontend did with it. `heading` was removed from the Go
envelope, but the frontend kept declaring it **required** on `BusLocation` and kept
manufacturing it: `messageHandler.ts` hardcoded `heading: 0`, and `transformers.ts` read a key
the backend never sent, defaulted it to `0`, then **range-validated the constant it had just
produced**. `BusMarkers.tsx` rendered the result as a labelled `HEADING` readout showing `0°`
for every device.

So the UI displayed a fabricated number, sourced from nothing, for every device on the map.
Removed from the frontend in `ec8a921`. `hub/message_test.go` now asserts the key is **absent**
from the serialized message — that test is the guard, and the word `heading` in it must not be
"cleaned up".

The rule that should have caught it — *no new UI control ships without working logic behind
it* — did not, **because the control was not new**. That gap is now anti-goal 9: do not render
a value the system cannot produce.

---

## 5. The contradiction catalogue

The mechanical inventory pass compared every zone-report finding against `CLAUDE.md`,
`BASELINE.md`, and `docs/audit/`. It found **more than thirty** contradictions, grouped by
kind. A representative sample:

**Citations that no longer resolved.** H3 resolution 9 was cited at `geofencing.go:24` and
`locations.go:22`; the actual lines are `:22` and `:23` — both wrong, **in opposite
directions**. A `PoolSize` literal was cited at `cache/redis.go:26`, which is the
`cfg.RedisPassword` line; no zone report found a `PoolSize` literal in that file at all. The
conclusion (`REDIS_POOL_SIZE` unused) was right; the evidence for it was not.

**Claims about state that had already changed.** DEFECT-3 was written as open, with
instructions to delete a toggle that no longer existed. The `internal/hub` baseline was listed
as 0% while `message_test.go` existed with two tests.

**The document's own rules, violated in the document's own repo.** Three concrete types sat
where the stated rule wants an interface: `ArrivalsService.geoService`, `IngestService.hub`,
and `WebSocketHandler.hub`. §7.4 named exactly three dead config values; `ENV` was a fourth
with no reader and the three `WS_*` variables a fifth through seventh.

**Prior-audit output contradicted by the code it described.** `ESSENCE.md:6` says the system
"strips away complex routing engines (like OSRM)" — OSRM is live on the route-creator path.
`ARCHITECTURE.md:36` and `ALGORITHM.md:31` say `ArrivalHandler` bypasses its service — by then
it held a service interface. `ARCHITECTURE.md:38` describes an H3 grid toggle that had been
deleted.

**And the exhibit that makes the point on its own:** `STATE.md:5` reads

```
CONTRADICTIONS OPEN: 0
```

in a document set whose siblings the same inventory pass contradicts fourteen times over. A
status line is not a verification.

**Legacy design documents, wrong about which code runs.** `backend_lld.md` recorded the Redis
TTL as 1 hour (it is 5 minutes), a `stops.sequence` column (it is `sequence_number`), and
broadcast behavior as "currently blocks" (it drops via a `default:` arm).
`component_documentation.md` documented a file `internal/cache/device.go` that does not exist,
carried a `Message` struct with a `Payload` field and `omitempty` on every numeric — *the
DEFECT-5 bug, preserved as documentation* — and used a path prefix for every one of its links
that does not match this tree.

### 5.1 Three failure categories, not two

Classifying those failures produced a taxonomy the citation gate now uses:

- **Transcription** — the source was read correctly and copied wrong.
- **Drift** — right when written; the code has since moved.
- **Stale-at-generation** — the report inherited a claim from an earlier round instead of
  observing source, so it was **already false when written**.

The third is the one that hides. Worked example: a zone report recorded
`services/interfaces.go:6` as importing `database`. Commit `fe918e0` removed that import
*before* commit `57afe4e` generated the zone reports — the claim was false the moment it was
written, and it propagated into three downstream documents.

### 5.2 Verify the claim, not the shape of the line

A citation reading "imports `database`" passes only if that line imports `database` — not
merely if it contains an import statement. A claim-checker **passed** the wrong-package claim
in two of the three documents carrying it: it confirmed line 6 exists and holds an import, and
stopped. The error surfaced only when a later consistency pass grepped all five documents for
the repeated citation string.

A checker that confirms a weaker property than the claim and reports pass is anti-goal 10 in
miniature. The gate that resulted extracts the checklist **mechanically** and lets the agent
verify rather than enumerate:

```
grep -oE '[A-Za-z0-9_./-]+\.(go|ts|tsx|sql|yml|sh|md):[0-9]+' CLAUDE.md | sort -u
```

Its first run found a real staleness while silently skipping seven citations, then concluded
all citations were accurate. Asking one agent to both *find* and *check* reproduces the exact
failure the gate exists to catch.

---

## 6. Coverage: before and after

**Before** — Phase 0 ground truth at `c53488d`, measured not estimated. `go build ./...` and
`go vet ./...` clean; integration tests: none, no `test/` directory, no testcontainers.

**After** — current, from one `go test ./... -cover` run.

| Package | Before | After | Target | Met? |
|---|---|---|---|---|
| `pkg/geo` | 100.0% | **100.0%** | ≥ 95% | ✅ |
| `internal/services` | 6.9% | **95.0%** | ≥ 60% | ✅ |
| `internal/hub` | 0.0% | **86.1%** | ≥ 50% | ✅ |
| `internal/handlers` | 0.0% | **12.7%** | ≥ 50% | ❌ |
| `internal/database` | 0.0% | 0.0% | integration test | covered only under `-tags=integration` |
| `internal/cache` | 0.0% | 0.0% | no package at 0% | ❌ |
| `internal/config` | 0.0% | 0.0% | ≥ 40% | ❌ |
| `internal/middleware` | 0.0% | 0.0% | ≥ 40% | ❌ |
| `cmd/server` | 0.0% | 0.0% | exempt | — |

**Stated plainly: the "no package sits at 0%" rule is not met.** Four packages remain at zero.
`internal/services` overshot its target; `internal/handlers` is at roughly a quarter of its.

### 6.1 D40 — a coverage number that was reported, believed, gated on, and did not reproduce

`internal/hub` coverage is **nondeterministic**: 86.1% in 4 of 5 consecutive runs, 87.3% in 1.
The ~1.2-point swing is one branch — almost certainly `drainAndClose`'s fast path versus its
timeout path, which depends on goroutine scheduling at shutdown. **Any gate on this package
must use the observed minimum (86.1%), never a best-observed figure**; gating at 87% would
fail four runs in five.

Separately, and more seriously: the D28 report claimed **90.8%** and described it as "stable at
repeated measurement." **That figure does not reproduce and was never achievable.** It
propagated into the Phase 2 report as an inherited baseline before being caught.

This is the same failure class as the citation errors above — stale-at-generation — expressed
in a number rather than a line reference. It is the reason this file reports 86.1%.

---

## 7. Dead code: what was verified, and the error rate of the audit that found it

A 19-claim dead-code report was verified claim by claim against source, by word-boundary grep
across the whole repository, checking indirect reachability through interfaces, wrappers, and
compile-time assertions rather than direct calls alone.

**Result: 14 TRUE, 5 PARTIALLY TRUE, 0 FALSE** — plus one separately-checked claim (that a
frontend config still pointed at a renamed API path) which was **FALSE**: the file contains no
reference to that route family at all.

Read narrowly, the report's hit rate was high — every "zero callers" grep checked out. Read as
delivered, the error rate is more serious: **one claim was filed in the wrong category
entirely**, listed as a free deletion when the function carries an explicit
`Phase4Reserved: … Do not delete` tag in its own source, identical to two others the same
report correctly treated as scope calls. Six functions carry that tag, added deliberately in
their own commit before the Phase 2 rename so they could not be confused with dead code.

**The report's own risk framing was also backwards.** It warned that deleting the candidates
would lower `internal/services` coverage. Measured against a real coverage profile, the
direction depends entirely on *which* set is removed, and the swing is under one point either
way — the tagged functions are individually at or near 100% coverage. A report that gets
"does X exist" right nineteen times but "is X safe to act on" wrong once is not a safe input
to a deletion pass without independent verification.

Genuinely free items were deleted. Deliberately kept, each for a recorded reason: the six
`Phase4Reserved` functions; `ZoneRepository.GetAll` (a pending product decision, D8);
`cfg.LogLevel` and `cfg.RedisPoolSize` (pending owner decision, D5); `models.Device` (an
explicit keep — Phase 4's `stale` event needs a device roster); and the two hand-rolled
middlewares, because swapping in the framework's built-in `Recover()` would silently drop
panic detail from the client-visible error body. That last one is a **behavior swap, not a
deletion** — and nothing in the test suite would have caught it, because no test exercises the
panic path.

---

## 8. Phase 4.5: blocked, and the documentation that claimed otherwise

`CLAUDE.md` §3.5 listed two algorithmic limitations as **"FIXED in Phase 4.5"** — straight-line
distance, and proximity-implies-approaching. The phase was specified in detail: project the
device and the zone onto the route polyline with `ST_LineLocatePoint`, take the difference
along the line, and emit no ETA when the device is past the zone.

**The phase is blocked, and those rows were wrong as written.** The `routes` table is
`(id, name, description)`. There is no geometry column, in that migration or any later one; a
search for `LINESTRING` across the repository returns nothing. `ST_LineLocatePoint` has
nothing to project onto. Both rows now read **OPEN — blocked** in `ARCHITECTURE.md` §7.1.

Three findings came out of that pre-flight, all recorded in `IDEAS.md`:

**D41 — the route geometry never existed.** Unblocking needs four separate pieces, none of
which is the algorithm itself: a migration adding the column; a persistence path; widening
`TripWithLocation` with `RouteID` (which means touching a file an existing scope guard
excludes); and a seed backfill, since the seeded routes would otherwise all take the fallback
and the feature would be invisible in the demo.

**D42 — the polyline is computed on every route creation and thrown away.** The route-creator
screen calls the public OSRM server, gets back a real road-snapped GeoJSON LineString, holds
it in component state, draws it — and discards it on unmount. The `POST /api/routes` body
carries only `{name, description, stops[]}`. The exact data the blocked phase needs is already
being produced, once per route creation, and dropped on the floor.

**D43 — a component that has rendered nothing since it was written.** The frontend declared
`Route.pattern` as an optional GeoJSON LineString, and `RoutePolyline` read
`route.pattern.coordinates` to draw the route path — mounted on every map render. The backend
`Route` model has no such field and its query selects only `id, name, description`, so
`pattern` was **always** undefined and the component **always** rendered nothing. Same shape as
DEFECT-3 and as `heading`: a UI element with no data behind it.

The honest read: the phase's one-day estimate was written against an assumed schema. The
algorithm is a day; the data it needs is not.

---

## 9. The anti-goals

Each of these was structural, not accidental, and each will recur under schedule pressure.

1. **Do not let a stated constraint be silently overridden.** The old contract banned
   databases, routers, and frontend frameworks. All three arrived; the file was never updated.
2. **Do not present architecture the code does not execute.** `ArrivalsService` existed, was
   wired, and was bypassed for months.
3. **Do not ship UI for logic that does not exist.** The H3 grid toggle.
4. **Do not let documentation drift from the code.** `graphify.html` claimed Redux; the code
   uses Zustand. `ESSENCE.md` claims OSRM was stripped out; `api/osrm.ts` is live.
5. **Do not write a spec from another document.** §7.2 once specified a `heading` field that
   exists in no model, no column, and no payload — because it was written from audit prose
   instead of from `message.go`.
6. **Do not document a contradiction and then proceed anyway.** `BASELINE.md` §6 listed ten
   doc/code conflicts, including the `heading` one, and the next commit planned work against
   the doc regardless. Finding the problem is not the same as stopping.
7. **Do not inflate the pitch.** Not a transit app, not a platform, not an Uber competitor.
8. **Do not start the interesting phase before finishing the boring one.** Infrastructure is
   more fun than handler tests. Handler tests come first.
9. **Do not render a value the system cannot produce.** The `HEADING` readout was hardcoded to
   `0` in both producers and displayed a fabricated number for every device. The "no new UI
   control without logic" rule did not catch it because the control was not new.
10. **Do not trust a check that defines its own scope.** The first citation gate reported "all
    line numbers accurate" while silently skipping seven citations. Extract the checklist
    mechanically; let the agent verify, not enumerate.

---

## 10. What this record is for

The individual defects are ordinary. Handlers that skip a service layer, a case-mismatched
string constant, a toggle wired to nothing — none of that is unusual in a codebase this age.

What is worth showing is the **second-order pattern**: every one of them survived because a
document was trusted in place of the code, and several survived *review* because the check
that was supposed to catch them tested something narrower than the claim. The DEFECT-1 verify
command grepped one directory. The claim-checker confirmed a line held an import rather than
that it held *the* import. A coverage number was quoted as stable without being re-run.

The fixes that mattered were therefore not code fixes. They were: extract verification
checklists mechanically rather than letting the checker choose its own scope; classify
staleness by *when* a claim became false, not just *that* it is false; and require that every
factual claim in the contract trace to a file and a line.

That is why this file exists, and why the numbers in it are the minimum observed rather than
the best observed.
