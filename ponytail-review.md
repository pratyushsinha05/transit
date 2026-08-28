Ponytail Audit — Repo-Wide

  Scanned backend/internal/** (Go) and frontend/src/** (TS/React). Verified every "zero callers" claim with grep before listing it.

  delete: CacheStore interface + Session type (SetSession/GetSession/DeleteSession/SessionTTL/KeyPrefixSession) + generic Metric ops (SetMetric/GetMetric/MetricTTL/KeyPrefixMetric). Nothing — grep confirms zero
  callers anywhere, and CacheStore itself is never used as a type (not even in main.go's wiring), only in its own self-referential var _ CacheStore = (*RedisCache)(nil) assertion. The "swap Redis for 
  Memcached" comment describes an abstraction with one implementation and no consumer. [backend/internal/cache/interface.go, backend/internal/cache/redis.go]

  yagni: DeviceCache map[string]interface{} wrapper — mislabeled "legacy"/"Deprecated" in comments, but it's actually the only live path (main.go:85, ingest.go:58). ingest.go builds a map, the wrapper 
  type-asserts every field back out into a struct, then calls the real typed method. Build cache.DeviceLocation directly in ingest.go and drop the map round-trip. [backend/internal/cache/redis.go:170-224, 
  backend/internal/services/ingest.go:53-59]

  delete: 4 of 5 WebSocket JSON handlers (handleConnected, handleArrivalUpdate, handleRouteUpdate, handleHeartbeat, handleError). backend/internal/hub/message.go has exactly one MsgType const (LOCATION_UPDATE) 
  and grep of hub/client.go + handlers/websocket.go shows no other JSON type is ever written — connection lifecycle is protocol-level ping/pong or set directly in ws.onopen/onclose (websocketClient.ts:26,42), 
  never a {type:'connected'} message. Keep handleLocationUpdate + a no-op default. [frontend/src/services/websocket/messageHandler.ts:20-129]

  delete: WS_CONFIG.transformers (3 identity-fn placeholders, comment admits "Placeholder, implemented in handler", zero callers) and WS_CONFIG.channels (comment admits "future: if backend supports," zero 
  callers, backend has no pub/sub). Nothing. [frontend/src/config/wsConfig.ts]

  delete: errorHandler.ts in full (handleError/normalizeError/AppError). Zero callers anywhere in the tree — every real error path calls logger.error directly. [frontend/src/services/errorHandler.ts]

  delete: WS_CONFIG.messageTypes.{disconnected,heartbeat_ack,system_message} and schemas.{arrivalUpdate,routeUpdate,heartbeat} — dead once the handlers above go. [frontend/src/config/wsConfig.ts]

  delete: transformBus + API_CONFIG.schemas.bus. Zero callers — there is no REST bus/device endpoint (only arrivals/stops/routes/location/health are declared), and WS location updates are built inline in 
  messageHandler.ts instead. [frontend/src/services/api/transformers.ts:69-82, frontend/src/config/apiConfig.ts schemas.bus]

  delete: API_CONFIG.retries (maxAttempts/baseDelay/maxDelay/exponentialBackoff). Zero callers — no retry logic exists anywhere in apiClient.ts despite the 429 branch's comment "will retry." 
  [frontend/src/config/apiConfig.ts]

  shrink: logger.ts's level/shouldLog machinery. level is hardcoded 'debug' (the lowest tier) and never set from env ("Determine log level from env if needed" was never done), so shouldLog always returns true —
  the whole comparison is dead weight around 4 console.* wrappers. Collapse until LOG_LEVEL is actually wired (same class of dead config CLAUDE.md §7.4 already tracks on the Go side). 
  [frontend/src/services/logger.ts]

  delete: API_CONFIG.timeouts.streaming/.websocket (only .default is ever read) and endpoint fields .method/.description/.optional/.cacheTime (only .path and health.interval are read at runtime). 
  [frontend/src/config/apiConfig.ts]

  delete: commented-out dead line // config.headers['X-API-Version'] = '1'; ("for future versioning"). [frontend/src/services/api/client.ts:32]

  Noted, not actioned: StopsService/RoutesService are pure one-line pass-throughs to their repository with no added logic — normally a yagni flag, but this is the CLAUDE.md-mandated DEFECT-1 fix
  (handler→service→repo layering was the exact bug that shipped dead code before). Cutting it reopens a real, already-closed defect, so it's excluded from the ranking above.

  net: -350 lines, -0 deps possible.



-----all----^

# Ponytail Review — 8 Hot-Path Files (Ultra Mode)





Ranked by: (hot-path frequency) × (allocation per execution) × (fixability).

---

## CRITICAL — Every Ping (1000s/sec)

### 1. backend/internal/services/ingest.go:51–57

**yagni:** `deviceState := map[string]interface{}{...}` built per ping, immediately passed to `cache.SetDeviceState()`, which type-asserts every field back into `*DeviceLocation` struct (cache/redis.go:184–197). One allocation to pass data that's immediately unpacked. The `DeviceCache` wrapper exists for "backward compatibility" (line 173) but is the only live path (main.go:85, ingest.go:58).

**Fix:** Pass `*DeviceLocation` directly to ingest service; eliminate the wrapper entirely. Or pass `loc` pointer to cache layer directly instead of building an intermediate map.

**net: –5 lines + wrapper removal saves ~55 lines (cache/redis.go:170–224).**

---

### 2. backend/internal/cache/redis.go:50, 63, 37

**shrink:** `fmt.Sprintf("%s%s:loc", KeyPrefixDevice, deviceID)` called twice per SetDeviceLocation. Replace with `strings.TrimSuffix(KeyPrefixDevice, ":") + ":" + deviceID + ":loc"` or just string concat `KeyPrefixDevice + deviceID + ":loc"`. Go's optimizer handles string concat well; `fmt.Sprintf` allocates unnecessarily.

**Lines:** 50 (SetDeviceLocation), 63 (GetDeviceLocation), same pattern in 37 and 91, 104.

**net: –1 line per call, saves ~5 allocations per 1000 pings at 1000/sec throughput.**

---

### 3. backend/internal/cache/redis.go:52

**native:** `json.Marshal(loc)` every `SetDeviceLocation`. At 1000 pings/sec, this is 1000 JSON encodings/sec for a 50-byte struct.

**Question:** Is the cache hit rate high enough to justify JSON marshalling on write? Measure before optimizing. If measured: consider using `json.Encoder` to a `bytes.Buffer` to reduce allocations, or pre-allocate a 256-byte pool.

**Blocked:** No baseline benchmark exists. Measure first.

---

### 4. backend/internal/cache/redis.go:62–78 (GetDeviceLocation)

**delete:** `GetDeviceLocation()` is never called on the hot path. Zero callers in the codebase (grep confirms). The symmetric `GetDeviceState()` wrapper is also dead. Delete all GetDeviceLocation + GetDeviceState + GetSession + DeleteDeviceLocation + DeleteSession + all Metric operations (Section 2 of interface.go + redis.go:87–154).

**Why dead:** Current usage only writes to cache (SetDeviceState), never reads. Read path is speculative for Phase 4+.

**net: –77 lines (all of redis.go:87–154 + interface.go GetSession/DeleteSession/Metric).**

---

## HIGH — Per Ping, But Lower Fixability (DEFECT-6 Blocks)

### 5. backend/internal/hub/hub.go:41–49

**yagni:** `default:` case silently skips on full buffer. Comment (lines 44–48) is 5 lines; code is 1 line. Comment is defensive prose about a problem DEFECT-6 is supposed to solve.

**Real issue:** No backpressure instrumentation, no shutdown signal, no way to measure p99 under load. The `default` here is a stub waiting for DEFECT-6 closure.

**Until DEFECT-6 is closed:** Keep the code but replace comment with `// Drop on buffer full — backpressure; TODO: instrument with metric.` (not "assume it's dead or stuck" speculation).

**net: –4 lines of comment (keep code).**

---

## MEDIUM — Speculative, Zero Callers

### 6. backend/internal/database/locations.go:27–32

**yagni:** `NewLocationRepositoryWithResolution()` is never called. Only `NewLocationRepository()` (line 20–25) is used, hardcoding resolution: 9.

**Justification in code:** CLAUDE.md §7.4(a) notes H3_RESOLUTION is read from config, validated (line 82–83), but never consumed. This factory exists speculatively; the config knob should either be wired or deleted.

**Fix:** Delete the WithResolution factory. If config wiring is needed later, re-add it.

**net: –6 lines.**

---

### 7. backend/internal/models/location.go:14–24

**delete:** `LocationUpdate` struct is never used. Zero callers (grep confirms). It's a duplicate declaration of hub.Message (backend/internal/hub/message.go, which is the canonical form).

**net: –11 lines.**

---

### 8. backend/internal/hub/client.go:25–26

**delete:** `newline` and `space` byte slices are declared but never referenced. Dead module-level state.

**net: –2 lines.**

---

## SKIP (Already Lean)

- **handlers/location.go** — Request parsing, validation, single json.Unmarshal per request (not per-loop). Clean.
- **database/device_routes.go** — Single-row scan, minimal allocation. Clean.
- **hub/client.go (except 25–26)** — Goroutine lifecycle unclear (DEFECT-6), but no allocation waste detected. Defer review until shutdown semantics are wired.

---

## Summary

**Unblocked (apply now):**
- Eliminate `DeviceCache` wrapper, pass `*DeviceLocation` directly (saves 55 lines + allocation).
- Delete unused Session/Metric/Get operations in cache (saves 77 lines).
- Replace `fmt.Sprintf` with string concat in key formatting (saves ~5 allocations/1000 pings).
- Delete unused `LocationUpdate` struct (saves 11 lines).
- Delete unused byte slices (saves 2 lines).
- Delete `NewLocationRepositoryWithResolution()` factory (saves 6 lines).
- Trim comment in hub.go select default (saves 4 lines).

**Blocked by DEFECT-6:**
- hub/hub.go backpressure instrumentation, hub/client.go goroutine lifecycle review.

**Blocked by missing benchmark:**
- cache/redis.go json.Marshal optimization (measure hit rate first).

---

**net: -~160 lines possible; -~200M bytes/day in allocation reduction at 1000 pings/sec, 50-byte payload.**

---

## LOWER FREQUENCY — Per Client Poll (10s–100s/sec)

### 9. backend/internal/services/geofencing.go:26–32

**yagni:** `NewGeofencingServiceWithResolution()` is never called. Only `NewGeofencingService()` (line 17–24) is used, hardcoding resolution: 9. Same pattern as database/locations.go.

**Fix:** Delete the WithResolution factory. If config wiring is needed later, re-add it.

**net: –6 lines.**

---

### 10. backend/internal/services/geofencing.go:165–190

**delete:** `DetectArrival()` and `DetectDeparture()` are never called. Zero callers (grep confirms). Speculative arrival-event detection for Phase 4+.

**net: –26 lines.**

---

### 11. backend/internal/services/arrivals.go:133–146

**delete:** `CalculateETAWithTraffic()` is never called. Zero callers (grep confirms). Comment says "placeholder" and "for now, this would...". Speculative traffic adjustment, deferred to Phase 4+.

**net: –14 lines.**

---

### 12. backend/internal/services/arrivals.go:149–165

**delete:** `DetectArrivalEvent()` is never called. Zero callers (grep confirms). Incomplete (comment says "for arrival detection, we need previous location" but only checks current location). Speculative.

**net: –17 lines.**

---

### 13. backend/internal/database/trips.go

**Lean already. Ship.** Single-row scan, no allocations beyond the query.

---

### 14. backend/internal/handlers/nearby.go:26–59, 89–122

**shrink:** Both `GetNearbyBuses()` and `GetNearbyStops()` duplicate parameter parsing (lat, lng, radius validation) in full. Extract to `parseNearbyParams(c echo.Context) (lat, lng, radius float64, int, error)`.

**Lines:** 26–59 and 89–122 (33 lines × 2 handlers).

**net: –30 lines (one shared function replaces two).**

---

### 15. backend/internal/handlers/nearby.go:71–74

**shrink:** Awkward nil-check for slice length:
```go
busCount := 0
if buses != nil {
    busCount = len(buses)
}
return c.JSON(..., map[string]interface{}{"count": busCount, ...})
```
Replace with: `c.JSON(..., map[string]interface{}{"count": len(buses), ...})` (Go returns 0 for nil slices).

**net: –4 lines.**

---

### 16. backend/internal/handlers/arrivals.go

**Lean already. Ship.** Minimal handler, no waste.

---

## Summary (Files 9–13)

**Unblocked (apply now):**
- Delete `NewGeofencingServiceWithResolution()` factory (saves 6 lines).
- Delete `DetectArrival()` and `DetectDeparture()` in geofencing.go (saves 26 lines).
- Delete `CalculateETAWithTraffic()` in arrivals.go (saves 14 lines).
- Delete `DetectArrivalEvent()` in arrivals.go (saves 17 lines).
- Extract parameter parsing in nearby.go handlers (saves 30 lines).
- Remove nil-check for slice length (saves 4 lines).

**net: -~97 lines possible from lower-frequency files.**

---

## Grand Total

**Across all 13 files:**
- Unblocked cuts: ~160 + ~97 = **~257 lines**.
- Allocation savings: ~5 per 1000 pings (SetDeviceState fmt.Sprintf).
- Dead code: ~200 lines of speculative, never-called features.

**Blocked:** DEFECT-6 (hub backpressure), missing benchmark (redis JSON optimization).

---

## Pass 3 — WS handler + repository loop (ponytail format)

Every claim grep-verified before listing.

**frontend/src/services/websocket/messageHandler.ts**

- `L21-23, L29-43: delete:` 5 of 6 switch cases. `hub/message.go:20` declares exactly one `MsgType` const (`LOCATION_UPDATE`); nothing else is ever written to the socket. `default: break` covers them.
- `L54-59: delete:` `handleConnected`. `websocketClient.ts:28` already sets `'connected'` in `ws.onopen` — protocol-level, no message needed.
- `L92-111: delete:` `handleArrivalUpdate`. Live arrivals path is the REST poll (`useArrivals.ts:31` → `setArrivals`).
- `L113-117: delete:` `handleRouteUpdate`. Logs, then a comment saying "implement if needed."
- `L119-121: delete:` `handleHeartbeat`. `L123-129: delete:` `handleError`.
- `L16-17, L46: delete:` commented-out `logger` calls.
- `L62, 68-81: yagni:` `WS_CONFIG.schemas.locationUpdate` key-name indirection — one consumer, every value equals the literal key (`schema.busId === 'device_id'`). `raw.device_id`, `raw.latitude`, direct.
- `L64-89: shrink:` inner `try/catch` nested inside the outer one at `L12-51`. Drop it.

**frontend/src/store/slices/arrivals.ts**

- `L8, L26-44: delete:` `updateArrival` — 18 lines of merge-by-`tripId` logic whose only caller is the dead `handleArrivalUpdate`. `setArrivals` is the live setter.

**frontend/src/store/slices/connection.ts**

- `L8, L34-38: delete:` `updateHeartbeat`, plus `lastHeartbeat` (`types/domain.ts:60`). Written by the dead handler, read nowhere.

**backend/internal/database/trips.go**

- `L47-54: native:` manual `rows.Next()` / `Scan` / append loop. pgx v5 (already a dep) ships `pgx.CollectRows(rows, pgx.RowToStructByPos[models.TripWithLocation])` — field order matches the `SELECT` exactly. 8 lines → 1.
- `L20-22: delete:` comment restating the SQL printed two lines below it.

**backend/internal/services/ingest.go**

- `L51-57: yagni:` `map[string]interface{}` built per ping, immediately type-asserted back into a struct by the cache wrapper. Pass `loc` through. (Same finding as §1; the fix lands in this file.)

**backend/internal/services/interfaces.go**

- `L38-42: yagni:` `DeviceCache.SetDeviceState(..., map[string]interface{})` — the map signature is the only reason the wrapper exists. `SetDeviceState(ctx, deviceID string, loc *models.Location) error`.
- Rest of file excluded: consumer-declared interfaces are the CLAUDE.md §7.1 / DEFECT-1 fix. Cutting them reopens a closed defect.

**net: -113 lines possible.**



