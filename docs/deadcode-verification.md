# Dead-code audit verification

**Source under test:** the 19-claim ponytail-audit report delivered in this session
(unranked here as "the report"). **Verified against:** commit `dcc3e44`, 2026-09-05.
**Method:** every claim re-derived from `grep -rnE '\bSymbol\b'` across the whole repo — Go,
TypeScript, tests, docs, `Makefile`, `scripts/`, migrations — never from re-reading the
report's own reasoning. Where a grep and a prior claim disagree, the grep is recorded as the
finding. No code was changed to produce this document.

**Note on scope drift while working:** `infra/docker-compose.yml` was edited on disk by
another process during this session (the `postgres` service's bind-mount of
`../backend/migrations` into `/docker-entrypoint-initdb.d` was removed). That file is
unrelated to any of the 19 claims below and was not touched by this verification; it is
recorded here only because the harness flagged the external change and asked that anything
that looked wrong be called out — it does not look wrong (it removes the DEFECT-7 double-apply
mechanism CLAUDE.md §4 already describes as a known issue) and is out of scope for this
document.

---

## 1. Verdict table

| # | Symbol | Claim | Verdict | Category | Est. lines |
|---|---|---|---|---|---|
| 1 | 5 WS message branches | unreachable, backend sends only `LOCATION_UPDATE` | **TRUE** | FREE | ~70 |
| 2 | `GetNearbyPredictions` | zero callers | **TRUE** | FREE | 27 |
| 3 | `IsAtZoneWithHysteresis` | zero callers | **TRUE** (callers), but tagged `Phase4Reserved` in source | **SCOPE CALL** | 37 |
| 4 | `DetectEntry`+`DetectExit` | zero callers | **TRUE** (callers), tagged `Phase4Reserved` | **SCOPE CALL** | 24 |
| 5 | `ZoneRepository.GetAll` | unreachable, D8 | **TRUE** | **SCOPE CALL** (D8: pending decision, not free) | 25 |
| 6 | `transformDevice`+`transformBus` | zero callers | **TRUE** | FREE | 22 |
| 7 | `CalculateETAWithTraffic` | zero callers, listed as free deletion | **TRUE** (callers), but tagged `Phase4Reserved` — **report misclassified this one** | **SCOPE CALL** | 14 |
| 8 | `database.LocationRepository.GetNeighborHexes` | dup. of services version | **TRUE** | FREE | 21 |
| 9 | `clsx`,`date-fns`,`lucide-react` | zero imports | **TRUE** | FREE | 3 deps |
| 10 | `KeyPrefixSession/Metric/Geo` | unused | **TRUE** | FREE | 3 |
| 11 | `cfg.LogLevel`,`cfg.RedisPoolSize` | dead, never read | **TRUE** | **SCOPE CALL** (D5: "needs an owner decision," not free) | 2 |
| 12 | leftover `pattern` schema keys | zero consumers | **TRUE** | FREE | 2 |
| 13 | `CacheStore` interface | one impl, self-assertion only | **TRUE** | FREE (see caveat) | 13 |
| 14 | `NewLocationRepository`+`NewGeofencingService` | superseded, zero callers | **PARTIALLY TRUE** — `NewLocationRepository` free; `NewGeofencingService` is called by a live, unrelated test | **FREE** (`NewLocationRepository` only) / requires a 1-line test edit for the other | 4 + 1 test line |
| 15 | `models.Device` | zero references | **TRUE** | **SCOPE CALL** — explicit CLAUDE.md §6 decision | 7 |
| 16 | triplicated lat/lng/radius parsing | 3 copies | **PARTIALLY TRUE** — 2 full copies + 1 partial (no radius block) | BEHAVIOR-preserving refactor | ~30 net |
| 17 | `DeviceCache` map round-trip | typed→map→typed | **TRUE** | BEHAVIOR-preserving refactor | ~15 net |
| 18 | hand-rolled `Recovery` vs `middleware.Recover()` | drop-in swap | **TRUE that it's hand-rolled; FALSE that it's a drop-in swap** | BEHAVIOR (observably different output) | -12/+1 |
| 19 | hand-rolled `Logging` vs `middleware.LoggerWithConfig` | drop-in swap | **TRUE that it's hand-rolled; FALSE that it's a drop-in swap** | BEHAVIOR (observably different output) | -13/+~6 |
| — | `apiConfig.ts` `/api/nearby/*` path drift | (the report's own final claim, already checked) | **FALSE** | — | — |

---

## 2. The free list

Genuinely unreferenced, reverses no documented decision, changes no behavior if removed.

### 2.1 Five dead WebSocket branches
```
$ grep -n "'connected'\|\"connected\"" backend/ -r --include='*.go'
(no output)
```
Backend defines exactly one message type:
```
$ grep -rn 'MsgType' backend/internal/hub/message.go
20:	MsgTypeLocationUpdate = "LOCATION_UPDATE"
```
Frontend confirmed to send nothing outbound over the socket (no dead-branch-only risk from an
inbound/outbound asymmetry):
```
$ grep -rn '\.send(' frontend/src/ --include='*.ts' --include='*.tsx'
(no output)
```
`onopen` in `websocketClient.ts:28` sets connection status directly
(`setConnectionStatus('connected')`) without depending on a server-sent `connected` message,
so `handleConnected` (`messageHandler.ts:54-59`) is dead too, alongside the four branches the
source already self-labels "Dead:" (`arrival_update` :92-115, `route_update` :117-122,
`heartbeat` :124-127, `error` :129-136). **Files:** `messageHandler.ts:19-51,54-136`,
`wsConfig.ts:12-31` (the `messageTypes` entries feeding the switch — `connected`,
`disconnected`, `heartbeat`, `heartbeat_ack`, `arrival_update`, `route_update`,
`system_message`, `error`; `location_update` stays).

### 2.2 `GeofenceService.GetNearbyPredictions`
```
$ grep -rnE '\bGetNearbyPredictions\b' . --include='*.go'
internal/services/geofence.go:84:// GetNearbyPredictions returns devices approaching any zone near a point
internal/services/geofence.go:85:func (s *GeofenceService) GetNearbyPredictions(...) ...
internal/services/arrivals_test.go:293:	predictions, err := svc.GetNearbyPredictions(...)
internal/services/arrivals_test.go:295:	t.Fatalf("GetNearbyPredictions failed: %v", err)
```
No `handlers/interfaces.go` entry, no route registration in `main.go`. **Unlike** claim 7
immediately below it in the source file, this function carries **no** `Phase4Reserved` tag —
verified by reading `geofence.go` in full; the tag sits above `CalculateETAWithTraffic`
(line 133) and `DetectEntryEvent` (line 151), not above this one (line 84). **File:**
`services/geofence.go:84-111`. Its test, `TestGeofenceService_GetNearbyPredictions`
(`arrivals_test.go:267-303`), dies with it.

### 2.3 `transformDevice` + `transformBus`
```
$ grep -rnE '\btransformDevice\b|\btransformBus\b' frontend/src/ backend/
frontend/src/services/api/transformers.ts:73:export const transformDevice = ...
frontend/src/services/api/transformers.ts:94:export const transformBus = transformDevice;
```
Zero call sites anywhere. **File:** `transformers.ts:73-94`.

### 2.4 `database.LocationRepository.GetNeighborHexes`
```
$ grep -rnE '\bGetNeighborHexes\b' internal/ cmd/ --include='*.go'
internal/database/locations.go:190:// GetNeighborHexes returns the H3 k-ring...
internal/database/locations.go:191:func (r *LocationRepository) GetNeighborHexes(...) ...
internal/services/geofence.go:117:	neighborHexes := s.geoService.GetNeighborHexes(zoneLat, zoneLng, 3)
internal/services/geofencing.go:108:// GetNeighborHexes returns H3 hexes within k distance of a point
internal/services/geofencing.go:109:func (s *GeofencingService) GetNeighborHexes(...) ...
internal/services/geofencing.go:144:		hexes := s.GetNeighborHexes(lat, lng, k)
internal/services/geofencing_method_test.go:190-213 (4 call sites)
```
Two distinct implementations exist. The **`database`**-layer one (`locations.go:190-210`) has
callers **only** in its own declaration and doc comment — the four real call sites (in
`geofence.go`, `geofencing.go`, and the test) are all against the **`services`**-layer
implementation (`geofencing.go:109-127`), a separately-written function using
`s.resolution` rather than `r.resolution` and computed independently (not a thin wrapper
around the database one). Confirmed a true, unreferenced duplicate. **File:**
`database/locations.go:188-210` (including its doc comment on 190).

### 2.5 `clsx`, `date-fns`, `lucide-react`
```
$ grep -rn "clsx" frontend/src/ --include='*.ts' --include='*.tsx'
(no output)
$ grep -rn "date-fns" frontend/src/ --include='*.ts' --include='*.tsx'
(no output)
$ grep -rn "lucide-react" frontend/src/ --include='*.ts' --include='*.tsx'
(no output)
```
Zero imports of any of the three anywhere in `src/`. **File:** `frontend/package.json`
(dependencies block).

### 2.6 `KeyPrefixSession`, `KeyPrefixMetric`, `KeyPrefixGeo`
```
$ grep -rnE '\bKeyPrefixSession\b|\bKeyPrefixMetric\b|\bKeyPrefixGeo\b' . --include='*.go'
internal/cache/interface.go:39:	KeyPrefixSession = "session:"
internal/cache/interface.go:40:	KeyPrefixMetric  = "metric:"
internal/cache/interface.go:41:	KeyPrefixGeo     = "geo:"
```
Only `KeyPrefixDevice` (line 38) is used, at `redis.go:49`. **File:**
`cache/interface.go:39-41`.

### 2.7 Leftover `pattern` schema keys
```
$ grep -rn 'pattern' frontend/src/config/apiConfig.ts frontend/src/config/wsConfig.ts
apiConfig.ts:31:            description: 'Get all routes and their patterns',
apiConfig.ts:107:            pattern: 'pattern',              // GeoJSON LineString of route path
wsConfig.ts:26:        route_update: 'route_update',       // Route pattern/stops updated
wsConfig.ts:62:            pattern: 'pattern',               // GeoJSON LineString
$ grep -n 'pattern' frontend/src/types/domain.ts
(no output, exit 1)
```
`Route.pattern` no longer exists on the frontend type (confirmed absent from
`types/domain.ts`), and no code dereferences `schema.pattern` anywhere
(`grep -rn 'schema\.pattern' frontend/src/` → no output). Both entries are the residue of a
field removed in an earlier pass. **Files:** `apiConfig.ts:107`, `wsConfig.ts:62` (the `:31`
and `:26` lines are prose comments, not dead code — leave them or fix the wording, but they
are not "leftover keys").

### 2.8 `CacheStore` interface
```
$ grep -rnE '\bCacheStore\b' . --include='*.go'
internal/cache/redis.go:14:// RedisCache implements CacheStore interface using Redis
internal/cache/redis.go:74:// Deprecated: Use CacheStore interface methods instead
internal/cache/redis.go:108:// Ensure RedisCache implements CacheStore interface
internal/cache/redis.go:109:var _ CacheStore = (*RedisCache)(nil)
internal/cache/interface.go:17:// CacheStore defines the interface for cache operations
internal/cache/interface.go:18:type CacheStore interface {
```
Every reference is inside the `cache` package itself: the declaration and one compile-time
assertion. No handler, service, or `main.go` ever declares a variable of type
`cache.CacheStore` or calls through it polymorphically — `services.DeviceCache` (a
*different*, smaller interface, `services/interfaces.go:40-42`) is what's actually consumed
at the service boundary. **Caveat, not a reason to keep it:** `RedisCache.Ping` is a real,
reachable method (`main.go` calls `redisClient` methods directly on the concrete type, not
through this interface) — deleting the *interface* does not touch `Ping` or `Close`
themselves, only the unused abstraction over them. **File:** `cache/interface.go:17-27`, plus
`redis.go:108-109`.

### 2.9 `NewLocationRepository` (the plain constructor)
```
$ grep -rnE '\bNewLocationRepository\b' cmd/ internal/ --include='*.go'
internal/database/locations.go:19:// NewLocationRepository creates a new LocationRepository
internal/database/locations.go:20:func NewLocationRepository(db *pgxpool.Pool) *LocationRepository {
```
Zero callers anywhere, including tests. `main.go:78` calls
`NewLocationRepositoryWithResolution` exclusively. **File:** `database/locations.go:19-25`.

---

## 3. The scope calls

Each of these reverses a decision that is recorded somewhere in the repo, not merely implied.
No recommendation is made either way.

### 3.1 `IsAtZoneWithHysteresis`, `DetectEntry`, `DetectExit`, `CalculateETAWithTraffic` (claims 3, 4, and — newly identified — 7)

**The decision is not just in a document — it is a comment directly above each function:**
```go
// Phase4Reserved: reserved for Phase 4 geofence-event wiring (CLAUDE.md §2).
// Zero callers today. Do not delete — Phase 4 will wire this onto the request path.
```
Verified present, verbatim, at all six of these locations:
```
$ grep -rn 'Phase4Reserved' backend/internal/services/*.go
geofencing.go:56   (above IsAtZone      -- NOT one of the 19 claims; see §7)
geofencing.go:67   (above IsAtZoneWithHysteresis -- claim 3)
geofencing.go:171  (above DetectEntry   -- claim 4)
geofencing.go:189  (above DetectExit    -- claim 4)
geofence.go:133    (above CalculateETAWithTraffic -- claim 7)
geofence.go:151    (above DetectEntryEvent -- NOT one of the 19 claims; see §7)
```
This tag was added deliberately, in its own commit, **before** the Phase 2 rename, citing a
council ruling:
```
$ git log --all --oneline -S'Phase4Reserved'
804f722 docs(services): tag Phase-4-reserved geofence orphans (item 1.4)
6e6852e docs(services): tag Phase-4-reserved geofence orphans (item 1.4)

$ git show 6e6852e --stat
    docs(services): tag Phase-4-reserved geofence orphans (item 1.4)

    Six functions with zero callers today are explicitly reserved for Phase 4.
    Tagged before the deletion pass so they cannot be confused with dead code.
```
The council's own text, `docs/council/03-verdict.md` §1.3 (recovered from commit `3922967`
since `docs/council/` is gitignored):
> **Verdict: evidence-driven convergence from a single agent's novel insight.**
> MAINTAINER's Wave 1 observation — that CONFIRMED-DEAD symbols and
> UNREACHABLE-BY-INTERFACE Phase-4-reserved functions are grep-indistinguishable — was
> unique to that paper. No other agent's Wave 1 paper raised it. Adoption across seven
> agents in Wave 2 was on the argument's merits, not from a shared brief prior.

CLAUDE.md §2's phase table backs the citation the tag makes: Phase 4's gate is *"Four event
types emitted; `GeofenceService` is on the request path"* — these six functions are exactly
that unwired machinery.

**What is lost by overriding it:** if Phase 4 (geofence events) is ever executed, these are
the functions it wires onto the request path — `IsAtZoneWithHysteresis` for flicker-resistant
zone detection, `DetectEntry`/`DetectExit` for the entry/exit event pair, and
`CalculateETAWithTraffic` for one of Phase 4's stated event payloads. Deleting them now means
re-writing them later from the same design, with no guarantee the re-write matches this one's
already-tested behavior (each carries a passing, non-trivial test — see §6).

**Correction to the original report: claim 7 was misclassified.** It appeared under
"DELETIONS CLAIMED" as if it were free; it is tagged identically to claims 3 and 4 and should
have been grouped with them.

### 3.2 `ZoneRepository.GetAll` (claim 5)

```
$ grep -rnE '\bGetAll\b' internal/database/zones.go internal/handlers/zones.go
internal/database/zones.go:104:func (r *ZoneRepository) GetAll(ctx context.Context) ([]models.Zone, error) {
```
No caller. `ZoneHandler.GetZones` (`handlers/zones.go:19-32`) hard-requires `route_id` and
returns 400 without it — confirmed by reading the handler in full. This matches `IDEAS.md`
D8 exactly (allowing for stale line numbers pre-dating the Phase 2 rename: D8 cites
`database/stops.go:100`, now `database/zones.go:104`):
> **D8 — `GET /api/stops` cannot return all stops.** … `StopRepository.GetAll` … is written,
> correct, and unreachable — no handler calls it. The frontend has wanted an all-stops
> endpoint since it was written … **Needs a decision: add a handler branch for "no
> route_id → GetAll", or a separate endpoint.**

D8 frames this as a **pending product decision with two live options**, one of which is
*wiring `GetAll` up*, not deleting it. Deleting it forecloses that option silently. **What is
lost:** the ability to resolve D8 by wiring the existing, correct method — the alternative
(a separate endpoint) would mean writing the same query again.

### 3.3 `cfg.LogLevel`, `cfg.RedisPoolSize` (claim 11)

```
$ grep -n 'LogLevel\|RedisPoolSize' internal/config/config.go
26:	RedisPoolSize int
37:	LogLevel string
59:		RedisPoolSize: getEnvAsInt("REDIS_POOL_SIZE", 50),
70:		LogLevel: getEnvOrDefault("LOG_LEVEL", "debug"),
$ grep -rn 'LogLevel\|RedisPoolSize' cmd/ internal/ --include='*.go' | grep -v config.go
(no output)
```
Confirmed dead — parsed, validated nowhere, never read. `IDEAS.md` D5:
> `cfg.LogLevel` is never read; logging is `log.Printf` plus Echo's default, not structured
> or levelled as §7.1 claims. `cfg.RedisPoolSize` is never read … Real structured, levelled
> logging is bigger than a Phase 1 wiring fix — it's a logging-library decision. **Flagged as
> the one open item at the Phase 0 gate; needs an owner decision (implement vs. soften the
> §7.1/§7.4 claim) before Phase 2.**

CLAUDE.md §7.4 independently documents the same two fields as category "(a) Read, validated,
then ignored" and gives the fix as "wiring `cfg` into the constructors," not deletion. **What
is lost:** D5's "before Phase 2" deadline has already passed (Phase 2 is done, per
`git log`) without the owner decision being made — deleting these fields now would be making
that decision by default, silently, rather than by the explicit choice D5 asked for.

### 3.4 `models.Device` (claim 15)

```
$ grep -rnE '\bmodels\.Device\b' cmd/ internal/ pkg/ test/ --include='*.go'
(no output)
```
Confirmed zero references anywhere, including the struct's own file (which declares `Device`,
not `models.Device`, from inside the package). CLAUDE.md §6, verbatim:
> **Decision: keep `models.Device` unchanged.** Do not rename into it. Do not delete it —
> Phase 4's `stale` event needs a device roster to know which devices should be reporting.

This is the most explicit of the four scope calls — a direct imperative, not an inference
from a pending-decision note. **What is lost:** the device roster Phase 4's `stale` event
(a device that stopped reporting) would need to know which devices exist at all.

---

## 4. The behavior swaps

### 4.1 `Recovery` vs `echo/middleware.Recover()` (claim 18)

**Confirmed hand-rolled** (`middleware/recovery.go:12-31`, quoted in full):
```go
func Recovery(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC RECOVERED] %v\n%s", r, debug.Stack())
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}
				c.Error(echo.NewHTTPError(500, fmt.Sprintf("internal server error: %v", err)))
			}
		}()
		return next(c)
	}
}
```
**Confirmed NOT a drop-in swap**, by reading Echo v4.13.4's actual source
(`$(go env GOMODCACHE)/github.com/labstack/echo/v4@v4.13.4/middleware/recover.go`, fetched
read-only via `go mod download` — no `go.mod`/`go.sum` change, confirmed with `git status`
before and after). Echo's builtin `Recover()`, on panic, calls `c.Error(err)` where `err` is
the plain recovered `error` — **not** wrapped in `echo.NewHTTPError`. This repo's custom
error handler (`middleware/errors.go:9-20`, quoted in full):
```go
func HTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	message := "Internal Server Error"
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		message = result(he.Message)
	}
	c.Logger().Error(err)
	_ = c.JSON(code, map[string]string{"error": message})
}
```
type-asserts on `*echo.HTTPError`. With the current hand-rolled `Recovery`, that assertion
**succeeds** and the client receives `{"error": "internal server error: <panic detail>"}`.
With Echo's builtin, the assertion **fails** (the error isn't wrapped), and the client would
instead receive the generic fallback `{"error": "Internal Server Error"}` — capitalization
different, panic detail **gone entirely**. The log line also changes: stdlib `log.Printf`
writing `"[PANIC RECOVERED] %v\n%s"` to the process's default logger, versus Echo's builtin
writing `"[PANIC RECOVER] %v %s\n"` (note: different text, "RECOVER" not "RECOVERED") through
`c.Logger()` (the gommon logger), a different destination/format entirely.

**What asserts on this today:** nothing. Confirmed —
`grep -rn 'PANIC RECOVERED\|Recovery\|recover(' internal/*/*_test.go test/*.go` finds no test
that triggers a panic through the middleware chain; `test/integration_test.go:205` wires
`middleware.Recovery` into its Echo instance but never exercises the panic path. **So no test
would break** from this swap — but the client-visible error body and the log format both
change silently in production, which is the exact failure mode the swap needs to be
re-verified against before landing, not assumed safe because tests stay green.

### 4.2 `Logging` vs `echo/middleware.LoggerWithConfig` (claim 19)

**Confirmed hand-rolled** (`middleware/logging.go:10-30`, quoted in full):
```go
func Logging(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		if err != nil {
			c.Error(err)
		}
		stop := time.Now()
		latency := stop.Sub(start)
		req := c.Request()
		res := c.Response()
		log.Printf(`{"method":"%s","path":"%s","status":%d,"latency_ms":%d}`,
			req.Method, req.URL.Path, res.Status, latency.Milliseconds())
		return err
	}
}
```
Echo's `middleware.Logger()`/`LoggerWithConfig` is template-tag-driven (confirmed reading
`logger.go` in the same module version — tags like `method`, `uri`, `status`, `latency`,
etc.) and does **not** default to this exact JSON shape or field set (`path` vs Echo's `uri`,
`latency_ms` as milliseconds vs Echo's nanosecond-by-default `latency` tag) — matching the
current output requires an explicit custom `Format` string, which is configuration work, not
a bare swap. **What asserts on this today:** nothing —
`grep -rn 'latency_ms\|"method":"%s"' internal/ test/` finds only the declaration itself, no
test parses or matches this log line. **What would need re-verification:** any external
log-parsing expectation (not present in this repo, but the kind of thing a real deploy would
have) built around the current field names and units.

---

## 5. False and partial claims

### 5.1 FALSE — `apiConfig.ts` `/api/nearby/*` path drift (the report's own closing claim)
```
$ grep -n 'nearby' frontend/src/config/apiConfig.ts
(no output)
```
```
$ grep -rn 'nearby' frontend/src/
frontend/src/hooks/useZones.ts:25:            // If we want "all stops" we might need a different endpoint (e.g. nearby).
frontend/src/services/api/zones.ts:13:        // Backend requires route_id for now, or might support nearby.
```
`apiConfig.ts` has **zero** `nearby` references — not the old path, not the new one. The
frontend never calls `/api/nearby/devices` or `/api/nearby/stops` at all; the two comments
above are speculative "we might need this someday" notes, unrelated to any path having moved.
The claim that the frontend "still points at the old path" is false — there is no frontend
code pointing anywhere on this route family for the claim to be stale about.

### 5.2 PARTIALLY TRUE — claim 7 (`CalculateETAWithTraffic`)
Correct that it has zero production callers. Incorrect in placement: it was listed under
"DELETIONS CLAIMED" as free-to-delete, but carries the identical `Phase4Reserved` tag as
claims 3 and 4, which the same report correctly treated as scope calls. See §3.1.

### 5.3 PARTIALLY TRUE — claim 14 (`NewLocationRepository` + `NewGeofencingService`)
`NewLocationRepository` is genuinely free (§2.9). `NewGeofencingService` is not
callsite-free — `TestGeofencingService_CalculateHex`
(`geofencing_method_test.go:76-77`) calls it directly:
```go
func TestGeofencingService_CalculateHex(t *testing.T) {
	svc := NewGeofencingService(nil, nil)
```
This test exercises `CalculateHex`/`GetHexResolution`, both of which are live, used methods
— it merely uses the plain constructor for convenience. Deleting `NewGeofencingService`
requires a one-line edit to this unrelated, currently-passing test
(`NewGeofencingServiceWithResolution(nil, nil, 9)`), which is not "free" in the sense the
report's own FREE category defines (§2's standard: "no decision reverses, no behavior
changes" — this one requires a test edit even though it reverses no decision).

### 5.4 PARTIALLY TRUE — claim 16 (triplicated parsing)
Two of the three cited ranges (`GetNearbyDevices`, `GetNearbyZones`) are confirmed
byte-for-byte identical parsing blocks (lat/lng/radius, 28 lines each). The third
(`GetHexInfo`) duplicates only the lat/lng portion (2 of the 3 sub-blocks) — it has no
`radius` parameter or radius-parsing logic at all. "Triplicated" overstates the third
instance; the underlying observation (real, extractable duplication) still holds.

### 5.5 TRUE-but-mislabeled — claims 18, 19
Both correctly identify hand-rolled code with a stdlib/framework equivalent (the report's own
"BEHAVIOR SWAPS CLAIMED" heading already gets this right, distinct from the "DELETIONS
CLAIMED" section). Flagged here only because the phrase "drop-in swap" undersells the
verified risk — see §4.

### 5.6 Claim IDEAS.md D22 — does not exist
The task brief that commissioned this verification cited "IDEAS.md D22" as prior evidence of
an indirect-reachability trap. `grep -n 'D22' IDEAS.md` returns no such entry — the repo's
`IDEAS.md` D-numbers run D4–D40 with no D22. Recorded because it is itself an unverified
claim that entered this process the same way the 19 claims did, and the same standard
applies: it does not check out.

---

## 6. Test and coverage impact — free list only

```
$ go test ./internal/services/... -cover
ok  	transit-backend/internal/services	(cached)	coverage: 94.0% of statements
```

Only claim 2 (`GetNearbyPredictions`) from the free list lives in `internal/services`; the
rest of the free list is in `cache`, `database`, or the frontend, none of which are covered
by this package's percentage.

**`TestGeofenceService_GetNearbyPredictions`** (`arrivals_test.go:267-303`) is the only test
that dies. Measured precisely via `go tool cover -func` against a fresh `-coverprofile`, not
estimated:

| | statements |
|---|---|
| package total | 134 |
| package covered | 126 (94.0%) |
| `GetNearbyPredictions` total | 14 |
| `GetNearbyPredictions` covered | 12 (85.7%, matches `go tool cover -func`) |
| **package total after removal** | 120 |
| **package covered after removal** | 114 |
| **package coverage after removal** | **95.0%** |

**Correction to this session's own earlier framing:** the audit's caveat said deleting the
scope-call candidates "lowers `internal/services` coverage from its current 94%." Measured
with `go tool cover -func` against a fresh `-coverprofile` and cross-checked line-by-line
against the source (not estimated), the answer depends on exactly which set of functions is
meant, and the two readings of "the scope-call candidates" move in **opposite directions**:

**Set A — exactly the symbols named in claims 2, 3, 4, and 7** (`GetNearbyPredictions`,
`IsAtZoneWithHysteresis`, `DetectEntry`, `DetectExit`, `CalculateETAWithTraffic` — 5
functions; does not include `IsAtZone` or `DetectEntryEvent`, which are not among the 19
claims — see §7):

| | statements |
|---|---|
| package total / covered | 134 / 126 (94.0%) |
| Set A total / covered | 45 / 41 |
| **after removing Set A: total / covered** | 89 / 85 |
| **after removing Set A: coverage** | **95.5%** |

**Set B — all six functions actually carrying the `Phase4Reserved` source tag**
(`IsAtZone`, `IsAtZoneWithHysteresis`, `DetectEntry`, `DetectExit`,
`CalculateETAWithTraffic`, `DetectEntryEvent` — the full tagged set, including the two new
findings in §7):

| | statements |
|---|---|
| package total / covered | 134 / 126 (94.0%) |
| Set B total / covered | 39 / 37 |
| **after removing Set B: total / covered** | 95 / 89 |
| **after removing Set B: coverage** | **93.7%** |

The direction flips because `IsAtZone`, `DetectEntry`, `DetectExit`, `DetectEntryEvent`, and
`CalculateETAWithTraffic` are each **100.0%** covered on their own (every branch is exercised
by their dedicated tests), while `GetNearbyPredictions` (85.7%) and `IsAtZoneWithHysteresis`
(88.9%) are the two below-package-average functions. Set A happens to remove both of the
weak ones, pulling the ratio up; Set B removes four fully-covered functions while leaving the
package's two weakest functions in place (`GetNearbyPredictions` stays, since it's not
tagged), pulling the ratio down slightly. **Neither computation supports treating "coverage
will drop" as a settled fact** — it depends entirely on which functions are actually removed,
and the swing in either direction is under 1 percentage point regardless. This does not make
deletion free — §3.1 still applies to every tagged function regardless of its effect on the
coverage number — but the coverage-drop claim should not be cited as an independent reason
for caution; the `Phase4Reserved` tag and the council decision behind it already are one.

`internal/handlers`, `internal/hub`, `internal/database`, `internal/cache`,
`internal/config`, and the frontend are unaffected by this package's number and were not
separately measured, since none of the free-list items outside `services` carry a passing
test (`transformDevice`/`transformBus` and the database-layer `GetNeighborHexes` have zero
test references, confirmed by the same greps in §2).

---

## 7. New findings (not on the list, not further investigated)

- **`IsAtZone`** (`geofencing.go:59-65`) carries the same `Phase4Reserved` tag
  (`geofencing.go:56-57`) but was not one of the 19 claims. Its only callers are
  `DetectEntry`/`DetectExit` (themselves tagged, zero-caller) and its own test — an
  indirect-reachability chain worth noting given the task's explicit concern about exactly
  this pattern (rule 4).
- **`GeofenceService.DetectEntryEvent`** (`geofence.go:154-171`) carries the same tag
  (`geofence.go:151-152`) and was likewise not among the 19 claims.
- **`wsConfig.ts`'s `transformers` and `channels` blocks** (`wsConfig.ts:73-86`) —
  `normalizeLocationUpdate`/`normalizeArrivalUpdate`/`normalizeRouteUpdate` are all
  identity-function placeholders (`(raw: any) => raw`), and `channels` documents a
  pub/sub scheme the code comment itself says the backend doesn't implement
  ("Currently all messages broadcast to all clients"). Neither was grepped for callers;
  noted only as a candidate for a future pass.

---

## 8. Error rate

Of 19 claims: **14 TRUE**, **0 FALSE**, **5 PARTIALLY TRUE** (claims 7, 14, 16, and — by the
report's own correct-but-underselling framing — 18 and 19). The one claim outside the
numbered 19 that this session was asked to re-check independently (the `apiConfig.ts` nearby-path
drift) was **FALSE**.

Read narrowly (is the symbol referenced or not), the report's factual hit rate is high: every
"zero callers" grep checked out. Read as delivered — where a claim was filed (FREE vs. SCOPE
CALL vs. BEHAVIOR) — the error rate is more serious: **one claim (7) was placed in the wrong
category entirely** (a tagged, do-not-delete function listed as a free deletion), and the
report's own risk framing around test coverage (§6) pointed in the wrong direction. A report
that gets "does X exist" right nineteen-for-nineteen but "is X safe to act on" wrong on a
scope call is not a safe input to a deletion pass without this kind of independent
verification — which is the reason this document exists.
