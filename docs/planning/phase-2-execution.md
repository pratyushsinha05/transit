# Phase 2 — domain rename: execution spec

**Audience:** an executor with no prior context on this repository. Everything needed is
inline. Where any other document — including `CLAUDE.md` §6 — disagrees with what is written
here, **this document wins**, because every claim below was verified against source on
2026-09-04 and several of §6's rows were not.

**Repo root:** the directory containing `backend/`, `frontend/`, `infra/`.
**Go module:** `transit-backend` (`backend/go.mod`), `go 1.25.0`, toolchain go1.26.0.
**All `go` commands run from `backend/`; all `npm` commands from `frontend/`.**

**The goal:** the repo's identity is a generic telemetry engine with a transit demo skin.
Core identifiers must be generic; only the demo layer stays transit-flavoured. **Zero
behavior change.**

---

## 0. Entry check — actual numbers, and what they correct

### 0.1 Commands and results

```
cd backend
grep -rn 'Bus\|Buses\|Stop\|Arrival' internal/ pkg/ test/ --include='*.go' | wc -l
```

| Scope | Match lines | Files |
|---|---|---|
| `internal/ pkg/ test/` (as the task specified) | **292** | 23 |
| **`cmd/ internal/ pkg/ test/`** (the real scope) | **305** | **24** |
| `frontend/src/` | **443** | **28** |

> **Correction 1 — the entry-check grep omits `cmd/`.** `backend/cmd/server/main.go` holds
> **13** more matches, including all five compile-time interface assertions and every
> constructor call. Renaming without it guarantees a broken build. Use the four-directory
> form above.

Note the shell detail that produced a false zero on the first attempt: in `zsh`, unquoted
`--include=*.go` is glob-expanded and the grep silently matches nothing. **Quote it**:
`--include='*.go'`.

`git status --short` at time of writing: only `?? docs/phase-1c-execution.md` (untracked
planning doc). Working tree otherwise clean.

### 0.2 Every §6.1 row, verified against source

| §6.1 row | Status | Reality |
|---|---|---|
| `models.NearbyBus` → `NearbyDevice`, `models/location.go:27` | ✅ **exact** | `type NearbyBus struct` at `internal/models/location.go:27` |
| `GetBusesInHex` → `GetDevicesInHex` | ⚠️ **correct but dead** | `internal/database/locations.go:79`. **Zero callers** — not in `services.LocationRepository`, not called anywhere. See §7 finding D35 |
| `GetBusesInHexes` → `GetDevicesInHexes` | ✅ | decl `locations.go:114`; also in `services/interfaces.go:28` |
| `GetBusesNearStop` → `GetDevicesNearZone` | ✅ | decl `locations.go:154`; also `services/interfaces.go:29` |
| `FindNearbyBuses` → `FindNearbyDevices` | ✅ | decl `services/geofencing.go:132`; also `handlers/interfaces.go:35` |
| `NearbyBusesTTL` → `NearbyDevicesTTL` | ✅ | `internal/cache/interface.go:36`. **Dead** (declared, never read) |
| `busLat`/`busLng`/`busHex` → `deviceLat`/… | ⚠️ **partly wrong** | `busLat`/`busLng` exist (`services/arrivals.go:115`, `geofencing.go:61,73`). **`busHex` does not exist anywhere** |
| `models.Stop` → `models.Zone`, `stop.go`→`zone.go` | ✅ | `internal/models/stop.go:3`. 42 qualified `models.Stop` references |
| `StopRepository` → `ZoneRepository`, `stops.go`→`zones.go` | ⚠️ **incomplete** | **Two distinct types share this name**: concrete `database.StopRepository` (`database/stops.go:12`) *and* the interface `services.StopRepository` (`services/interfaces.go:12`). §6.1 lists one row; the rename touches both, in different commits |
| `StopHandler` → `ZoneHandler`, `stops.go`→`zones.go` | ✅ | `internal/handlers/stops.go:11` |
| `models.ArrivalEvent` → `models.GeofenceEvent`, `models/trip.go` | ❌ **STALE — DOES NOT EXIST** | `internal/models/trip.go` contains **only** `TripWithLocation`. `grep -rn 'ArrivalEvent'` finds no type — only the *method* `ArrivalsService.DetectArrivalEvent`. **Delete this row.** (`IDEAS.md` D16, which tracked it at `models/trip.go:11`, is likewise stale) |
| `ArrivalsService` → `GeofenceService`, `arrivals.go`→`geofence.go` | ✅ | `internal/services/arrivals.go:10` |
| `ArrivalHandler` → `GeofenceHandler`, `arrivals.go`→`geofence.go` | ✅ | `internal/handlers/arrivals.go:11` |
| `models.Device` unchanged | ✅ | see §3 |
| `Route` / `location_history` unchanged | ✅ | see §3 |

> **Correction 2 — `models.ArrivalEvent` no longer exists.** It was removed before this
> phase (`models/trip.go` now holds only `TripWithLocation`, moved there by the DEFECT-1
> closure). A rename plan that carries this row will send the executor hunting for a type
> that is not there. This is the exact failure mode `CLAUDE.md` §12.5 names.
>
> **Correction 3 — `models.Trip` is also gone.** `grep -rn 'models\.Trip\b'` returns
> nothing. `IDEAS.md` D10 ("`models.Trip` is dead code") is stale — it is not dead, it is
> *absent*.

### 0.3 Identifiers §6.1 never listed but which the rename requires

§6.1 is a partial list. These are real, in-scope, and missing from it:

| Identifier | Declared at | Occurrences |
|---|---|---|
| `services.StopsService` + `NewStopsService` | `services/stops.go:10,15` | 11 / 3 |
| `GetStopsByRoute` | `services/stops.go:20` | 6 |
| `handlers.StopsService` (interface) | `handlers/interfaces.go:18` | (in the 11) |
| `handlers.ArrivalService` (interface) | `handlers/interfaces.go:13` | 5 |
| `services.StopRepository` (interface) | `services/interfaces.go:12` | (in the 19) |
| `services.ArrivalPrediction` | `services/arrivals.go:33` | 9 |
| `GetArrivalsForStop` | `services/arrivals.go:45` | 9 |
| `GetNearbyArrivals` | `services/arrivals.go:85` | 4 |
| `DetectArrivalEvent` | `services/arrivals.go:155` | 4 |
| `isApproaching` (method) | `services/arrivals.go:115` | — |
| `IsAtStop` | `services/geofencing.go:61` | 12 |
| `IsAtStopWithHysteresis` | `services/geofencing.go:73` | 7 |
| `FindNearbyStops` | `services/geofencing.go:165` | 7 |
| `DetectArrival` / `DetectDeparture` | `services/geofencing.go:178,193` | — |
| `GetActiveTripsBeforeStop` | `database/trips.go:19`, `services/interfaces.go:20` | 4 |
| `models.CreateStopInput` | `models/route.go:17` | 4 |
| `CreateRouteRequest.Stops` field | `models/route.go:13` | — |
| `CreateRouteResponse.StopCount` / `.StopIDs` | `models/route.go:28,29` | 4 / 3 |
| `database.NewStopRepository` | `database/stops.go:17` | 2 |
| `handlers.NewStopHandler` / `GetStops` | `handlers/stops.go:15,19` | 1 / 2 |
| `handlers.GetNearbyBuses` / `GetNearbyStops` | `handlers/nearby.go:26,89` | — |
| `handlers.NewArrivalHandler` / `GetArrivals` | `handlers/arrivals.go:15,22` | 1 / — |
| `NearbyService` interface methods | `handlers/interfaces.go:35,36` | — |

### 0.4 False positives — grep matches that MUST NOT be renamed

| Location | Match | Why it stays |
|---|---|---|
| `internal/hub/hub.go:140` | `defer ticker.Stop()` | stdlib `time.Ticker.Stop` |
| `internal/hub/client.go:69` | `ticker.Stop()` | stdlib |
| `internal/hub/hub.go:50` | `// Stop reading Broadcast…` | English prose in a D28 comment |
| `internal/services/routes_test.go:61,62` | `Name: "Stop 1"` / `"Stop 2"` | test fixture *data*, permitted demo vocabulary (§6.3) |
| `frontend/src/components/Map/MapCameraHandler.tsx:55` | `e.stopPropagation()` | DOM API |

**`internal/hub/` contains no domain identifiers at all.** Its three matches are all above.
Do not open hub files during this phase.

### 0.5 The claim about new test files — corrected

The task premise was that Phase 1B/1C test files "will contain Bus/Stop/Arrival identifiers
throughout." **Verified false for most of them.**

| New test file | Domain identifiers? |
|---|---|
| `test/integration_test.go` (Phase 1C) | **None.** Written with `deviceID`/`routeID`/`tripID`; touches `routes`, `devices`, `trips`, `location_history` — never `stops` |
| `internal/hub/hub_shutdown_test.go` | None |
| `internal/hub/hub_concurrency_test.go` | None |
| `internal/hub/hub_drop_test.go` | None |
| `internal/hub/message_test.go` | None (see §3 — its `heading` refs are a separate guard) |
| `internal/handlers/location_test.go` | None |
| `internal/handlers/location_bench_test.go` | None |
| `internal/services/ingest_test.go` | None |
| `internal/services/geofencing_test.go` | None |
| **`internal/services/geofencing_method_test.go`** | **73 matches** |
| **`internal/services/arrivals_test.go`** | **54 matches** |
| **`internal/services/stops_test.go`** | **7 matches** |
| **`internal/services/routes_test.go`** | **6 matches** |

So exactly **four** test files are affected, all in `internal/services/`, totalling **140**
match lines — not "~2000 lines throughout." The integration test needs **no** changes, which
is worth knowing because it is the slowest one to re-run.

---

## 1. The complete Go identifier map

Rename these. Old → new, declaration site, total whole-word occurrences across
`cmd/ internal/ pkg/ test/`.

### 1.1 Types owned by `models` and `database` (Commit 1)

| Current | New | Declared | Occ. |
|---|---|---|---|
| `models.NearbyBus` | `models.NearbyDevice` | `internal/models/location.go:27` | 24 |
| `models.Stop` | `models.Zone` | `internal/models/stop.go:3` → **file → `zone.go`** | 42 (qualified) |
| `models.CreateStopInput` | `models.CreateZoneInput` | `internal/models/route.go:17` | 4 |
| `CreateRouteRequest.Stops` | `.Zones` | `internal/models/route.go:13` | — |
| `CreateRouteResponse.StopCount` | `.ZoneCount` | `internal/models/route.go:28` | 4 |
| `CreateRouteResponse.StopIDs` | `.ZoneIDs` | `internal/models/route.go:29` | 3 |
| `database.StopRepository` | `database.ZoneRepository` | `internal/database/stops.go:12` → **file → `zones.go`** | (of 19) |
| `database.NewStopRepository` | `NewZoneRepository` | `internal/database/stops.go:17` | 2 |
| `LocationRepository.GetBusesInHex` | `GetDevicesInHex` | `internal/database/locations.go:79` | 2 (dead) |
| `LocationRepository.GetBusesInHexes` | `GetDevicesInHexes` | `internal/database/locations.go:114` | 5 |
| `LocationRepository.GetBusesNearStop` | `GetDevicesNearZone` | `internal/database/locations.go:154` | 7 |
| `TripRepository.GetActiveTripsBeforeStop` | `GetActiveTripsBeforeZone` | `internal/database/trips.go:19` | 4 |
| `cache.NearbyBusesTTL` | `NearbyDevicesTTL` | `internal/cache/interface.go:36` | 2 (dead) |

Local variables in these files, same commit: `stops`→`zones`, `s models.Stop`→`z models.Zone`,
`buses`→`devices`, `b models.NearbyBus`→`d models.NearbyDevice`, `stopIDs`→`zoneIDs`,
`stopID`→`zoneID`, `stopSequence`→`zoneSequence`.

**Consumers whose *references* must update in Commit 1 to keep the build green** (their own
type names change later, in Commit 2): `internal/services/interfaces.go:13,14,15,20,28,29`,
`internal/services/{arrivals,geofencing,stops}.go`, `internal/handlers/interfaces.go:19,35,36`,
`internal/handlers/{stops,nearby,routes,arrivals}.go`, `cmd/server/main.go`, and the four
`internal/services/*_test.go` files. See §5.0 for why this is unavoidable.

### 1.2 Types owned by `services` and `handlers` (Commit 2)

| Current | New | Declared | Occ. |
|---|---|---|---|
| `services.StopRepository` (iface) | `ZoneRepository` | `internal/services/interfaces.go:12` | (of 19) |
| `services.StopsService` | `services.ZonesService` | `internal/services/stops.go:10` → **file → `zones.go`** | 11 |
| `services.NewStopsService` | `NewZonesService` | `internal/services/stops.go:15` | 3 |
| `StopsService.GetStopsByRoute` | `GetZonesByRoute` | `internal/services/stops.go:20` | 6 |
| `services.ArrivalsService` | `services.GeofenceService` | `internal/services/arrivals.go:10` → **file → `geofence.go`** | 13 |
| `services.NewArrivalsService` | `NewGeofenceService` | `internal/services/arrivals.go:18` | 8 |
| `services.ArrivalPrediction` | `services.GeofencePrediction` | `internal/services/arrivals.go:33` | 9 |
| `GetArrivalsForStop` | `GetPredictionsForZone` | `internal/services/arrivals.go:45` | 9 |
| `GetNearbyArrivals` | `GetNearbyPredictions` | `internal/services/arrivals.go:85` | 4 |
| `DetectArrivalEvent` | `DetectEntryEvent` | `internal/services/arrivals.go:155` | 4 |
| `GeofencingService.IsAtStop` | `IsAtZone` | `internal/services/geofencing.go:61` | 12 |
| `IsAtStopWithHysteresis` | `IsAtZoneWithHysteresis` | `internal/services/geofencing.go:73` | 7 |
| `FindNearbyBuses` | `FindNearbyDevices` | `internal/services/geofencing.go:132` | 9 |
| `FindNearbyStops` | `FindNearbyZones` | `internal/services/geofencing.go:165` | 7 |
| `DetectArrival` | `DetectEntry` | `internal/services/geofencing.go:178` | — |
| `DetectDeparture` | `DetectExit` | `internal/services/geofencing.go:193` | — |
| `handlers.ArrivalService` (iface) | `GeofenceService` | `internal/handlers/interfaces.go:13` | 5 |
| `handlers.StopsService` (iface) | `ZonesService` | `internal/handlers/interfaces.go:18` | (of 11) |
| `handlers.StopHandler` | `handlers.ZoneHandler` | `internal/handlers/stops.go:11` → **file → `zones.go`** | 5 |
| `handlers.NewStopHandler` | `NewZoneHandler` | `internal/handlers/stops.go:15` | 1 |
| `StopHandler.GetStops` | `ZoneHandler.GetZones` | `internal/handlers/stops.go:19` | 2 |
| `handlers.ArrivalHandler` | `handlers.GeofenceHandler` | `internal/handlers/arrivals.go:11` → **file → `geofence.go`** | 5 |
| `handlers.NewArrivalHandler` | `NewGeofenceHandler` | `internal/handlers/arrivals.go:15` | 1 |
| `ArrivalHandler.GetArrivals` | `GeofenceHandler.GetPredictions` | `internal/handlers/arrivals.go:22` | — |
| `NearbyHandler.GetNearbyBuses` | `GetNearbyDevices` | `internal/handlers/nearby.go:26` | — |
| `NearbyHandler.GetNearbyStops` | `GetNearbyZones` | `internal/handlers/nearby.go:89` | — |
| params `busLat`,`busLng` | `deviceLat`,`deviceLng` | `services/arrivals.go:115`, `geofencing.go:61,73` | — |
| params `stopLat`,`stopLng`,`wasAtStop` | `zoneLat`,`zoneLng`,`wasAtZone` | same | — |

`busHex` from §6.1 does not exist — nothing to do.

### 1.3 Test files (rename in the same commit as the code they test)

| File | Matches | Commit |
|---|---|---|
| `internal/services/geofencing_method_test.go` | 73 | 1 (mock types) + 2 (service methods) |
| `internal/services/arrivals_test.go` | 54 | 1 + 2 |
| `internal/services/stops_test.go` | 7 | 2 |
| `internal/services/routes_test.go` | 6 | 1 |

Mock types to rename: `mockStopRepo`→`mockZoneRepo`, its fields `getByIDFn`/`getByRouteIDFn`/
`getNearbyFn` (names unchanged, signatures reference `models.Zone`), `getBusesInHexesFn`→
`getDevicesInHexesFn`, `getBusesNearStopFn`→`getDevicesNearZoneFn`, and the methods
implementing `LocationRepository`.

Test **function names** (`TestStopsService_GetStopsByRoute` →
`TestZonesService_GetZonesByRoute`, etc.) rename with their subjects.

---

## 2. The complete frontend map

### 2.1 §6.2's file renames — all twelve verified present

| Current | New | Status |
|---|---|---|
| `src/components/Map/BusMarkers.tsx` | `Map/DeviceMarkers.tsx` | ✅ exists, 27 matches |
| `src/components/Map/StopMarkers.tsx` | `Map/ZoneMarkers.tsx` | ✅ 31 |
| `src/components/Sidebar/ArrivalCard.tsx` | `Sidebar/GeofenceEventCard.tsx` | ✅ 13 |
| `src/components/Sidebar/ArrivalsList.tsx` | `Sidebar/GeofenceEventList.tsx` | ✅ 22 |
| `src/components/Sidebar/StopDetails.tsx` | `Sidebar/ZoneDetails.tsx` | ✅ 12 |
| `src/store/slices/busLocations.ts` | `store/slices/devices.ts` | ✅ 11 |
| `src/store/slices/arrivals.ts` | `store/slices/geofenceEvents.ts` | ✅ 25 |
| `src/store/slices/stops.ts` | `store/slices/zones.ts` | ✅ 9 |
| `src/hooks/useStops.ts` | `hooks/useZones.ts` | ✅ 21 |
| `src/hooks/useArrivals.ts` | `hooks/useGeofenceEvents.ts` | ✅ 20 |
| `src/services/api/stops.ts` | `services/api/zones.ts` | ✅ 7 |
| `src/services/api/arrivals.ts` | `services/api/geofenceEvents.ts` | ✅ 9 |
| `types/domain.ts` `BusLocation` → `DeviceLocation` | | ✅ `domain.ts:41` |
| `layerVisibility.buses`/`.stops` → `.devices`/`.zones` | | ✅ `ui.ts:36,38,61` — and confirms DEFECT-3's `grid` key is gone |

> **Correction 4 — §6.2 lists only the twelve *renamed* files. Sixteen more files contain
> references that must change and are named nowhere in §6.**

| File | Matches | What it holds |
|---|---|---|
| `src/store/slices/ui.ts` | **39** | `selectedStopId`, `followedBusId`, `routeCreatorStops`, `addCreatorStop`, `removeCreatorStop`, `updateCreatorStopName`, `updateCreatorStopPosition`, `reorderCreatorStops`, `clearCreatorStops`, `layerVisibility` |
| `src/components/Sidebar/RouteCreatorPanel.tsx` | 30 | creator-stop UI |
| `src/services/api/transformers.ts` | 23 | `transformArrival`, `transformStop`, `transformBus` |
| `src/config/apiConfig.ts` | 21 | endpoint paths + field-map schema (**see §3 — paths are wire contract**) |
| `src/components/Sidebar/Sidebar.tsx` | 21 | composition |
| `src/components/Map/MapCameraHandler.tsx` | 18 | +1 false positive (`stopPropagation`) |
| `src/components/Map/RouteCreatorMarkers.tsx` | 17 | |
| `src/services/websocket/messageHandler.ts` | 14 | |
| `src/config/wsConfig.ts` | 12 | |
| `src/types/domain.ts` | 11 | `Arrival`, `Stop`, `BusLocation`, `RouteCreatorStop`, `Route.stops`, `CreateRoutePayload.stops`, `CreateRouteResponse.stop_count`/`stop_ids` |
| `src/store/slices/stops.ts` | 9 | |
| `src/components/Sidebar/OperationsPanel.tsx` | 8 | the two surviving layer toggles |
| `src/store/index.ts` | 7 | `createBusLocationsSlice`, `BusLocationsSlice`, `createStopsSlice`, `StopsSlice`, `createArrivalsSlice`, `ArrivalsSlice` |
| `src/styles/globals.css` | 7 | `.bus-blip*` (×5), `.stop-crosshair-marker` (×2) |
| `src/components/Map/MapContainer.tsx` | 4 | |
| `src/components/Map/MapClickHandler.tsx` | 3 | |
| `src/services/api/createRoute.ts` | 1 | |

### 2.2 Frontend type/symbol renames

| Current | New | Where |
|---|---|---|
| `interface Stop` | `interface Zone` | `types/domain.ts:22` |
| `interface Arrival` | `interface GeofenceEvent` | `types/domain.ts:10` |
| `interface BusLocation` | `interface DeviceLocation` | `types/domain.ts:41` |
| `interface RouteCreatorStop` | `RouteCreatorZone` | `types/domain.ts:66` |
| `Stop.arrivals` | `.events` | `types/domain.ts:29` |
| `Route.stops` | `.zones` | `types/domain.ts:37` |
| `BusLocationsSlice` / `createBusLocationsSlice` | `DevicesSlice` / `createDevicesSlice` | `store/index.ts:7`, `slices/busLocations.ts` |
| `StopsSlice` / `createStopsSlice` | `ZonesSlice` / `createZonesSlice` | `store/index.ts:9` |
| `ArrivalsSlice` / `createArrivalsSlice` | `GeofenceEventsSlice` / `createGeofenceEventsSlice` | `store/index.ts:8` |
| `selectedStopId` / `setSelectedStopId` | `selectedZoneId` / `setSelectedZoneId` | `ui.ts:15,16,51,77` |
| `followedBusId` / `setFollowedBusId` | `followedDeviceId` / `setFollowedDeviceId` | `ui.ts:17,18,52,78` |
| `routeCreatorStops` | `routeCreatorZones` | `ui.ts` ×11 |
| `addCreatorStop`/`removeCreatorStop`/`updateCreatorStopName`/`updateCreatorStopPosition`/`reorderCreatorStops`/`clearCreatorStops` | `…Zone`/`…Zones` | `ui.ts:28-33, 94-130` |
| `layerVisibility.stops`/`.buses` | `.zones`/`.devices` | `ui.ts:36,38,61` |
| CSS `.bus-blip*` | `.device-blip*` | `globals.css:62,67,73,88,101` + consumers |
| CSS `.stop-crosshair-marker` | `.zone-crosshair-marker` | `globals.css:117,122` + consumers |

---

## 3. MUST NOT RENAME — each re-verified

| Item | Verified how | Verdict |
|---|---|---|
| **`models.Device`** | `internal/models/device.go:3`; `grep -rn 'models\.Device' cmd/ internal/ pkg/ test/ --include='*.go'` → **no matches**. No `type Bus` exists anywhere, so nothing collides | **Keep, untouched.** Phase 4's `stale` event needs a device roster. Do not rename *into* it either |
| **`Route` / `route_id`** | `models/route.go`, `routes` table, `Message.RouteID` | Already generic. **Unchanged** |
| **`location_history`** | `migrations/001:32` | Already generic. **Unchanged** |
| **`hex_res9`** | column `003:7`; field `models.Location.HexRes9` (`location.go:11`); JSON tag on the *wire envelope* is `h3_hex` (`hub/message.go:15`) per §7.2 | **Unchanged.** One name internally, one on the wire |
| **`migrations/004_seed_data.sql`** | `bus-001`…`bus-005`, "Bus Alpha", stop names, "Downtown Express" | **Unchanged** — §6.3 permits transit vocabulary in demo fixtures |
| **`hub/message_test.go` `heading` refs** | asserts the key is *absent* (DEFECT-2 guard) | **Unchanged.** Do not "tidy" the word `heading` out of it |
| **`ticker.Stop()` ×2, the `// Stop reading Broadcast` comment** | §0.4 | **Unchanged** — stdlib and prose |
| **`e.stopPropagation()`** | `MapCameraHandler.tsx:55` | **Unchanged** — DOM API |

### 3.1 Also must not change — the wire contract (not in §6, but implied by "zero behavior change")

Renaming these would be a *behavior* change, which §6.3 forbids. They stay **exactly** as-is:

- **HTTP route paths**: `/api/stops`, `/api/arrivals`, `/api/nearby/buses`, `/api/nearby/stops`
  (`cmd/server/main.go:138,141,144,145`) and their frontend mirrors in `config/apiConfig.ts`.
- **Query params**: `stop_id` (`handlers/arrivals.go:23`), `route_id`.
- **Every JSON struct tag**: `json:"stops"` (`models/route.go:13`), `json:"stop_count"`,
  `json:"stop_ids"` (`route.go:28,29`), and all of `models.Stop`'s tags. Rename the Go
  *field*, keep the *tag*. `CreateRouteResponse.StopCount int \`json:"stop_count"\`` becomes
  `ZoneCount int \`json:"stop_count"\``.
- **Frontend `apiConfig.ts` schema values** (the right-hand side strings like
  `id: 'trip_id'`, `stop_id: 'stop_id'`) — these name backend fields. Keys may be renamed;
  values may not.

If a future phase wants to rename the public API, that is its own decision with its own
migration story. It is **not** this phase.

---

## 4. The SQL migration

### 4.1 The number is 005, not 006

§6.3 says *"Add `migrations/006_rename_stops_zones.sql` (005 is taken by the DEFECT-2 work if
it created one — check first)."* **Checked:**

```
$ ls backend/migrations/
001_create_tables.sql  002_add_indexes.sql  003_add_h3_postgis.sql
004_seed_data.sql      migrations.go
$ git log --all --diff-filter=A --name-only -- 'backend/migrations/00*' | grep '005\|006'
(no output)
```

DEFECT-2 created **no** migration; the DEFECT-7 fix edited `004` in place. **The file is
`backend/migrations/005_rename_stops_zones.sql`.**

`migrations.go` is `//go:embed *.sql`, so a new file is picked up automatically — no
registration step.

### 4.2 The migration content

```sql
-- Migration 005: rename stops -> zones (Phase 2 domain rename).
-- Mechanical. No data change, no column type change.
-- Guarded so it is idempotent and safe on a database where 001-004 have
-- already run and on a fresh one where they run in sequence first.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema = 'public' AND table_name = 'stops')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables
                       WHERE table_schema = 'public' AND table_name = 'zones')
    THEN
        ALTER TABLE stops RENAME TO zones;
    END IF;
END $$;

-- Index names do not follow the table automatically.
ALTER INDEX IF EXISTS idx_stops_geom RENAME TO idx_zones_geom;
```

Notes the executor needs:

- **`trips.current_stop` stays.** It is an `INT` sequence pointer, not a table reference, and
  renaming it would require touching `004`'s INSERT column list — forbidden by §6.3. Record
  as a follow-up, do not do it here.
- **The FK `stops.route_id REFERENCES routes(id)` survives** an `ALTER TABLE … RENAME`;
  constraints follow the table.
- **PostGIS needs nothing.** `geometry_columns` has been a view over the catalog since
  PostGIS 2.0, so the rename is reflected automatically. Do **not** hand-edit it.
- **Migration order on a fresh DB is safe**: `001` creates `stops`, `004` seeds `stops`,
  `005` renames. `RunMigrations` applies in sorted filename order (`fs.ReadDir`), so `004`
  always precedes `005`.
- **`004`'s verification block** (`SELECT COUNT(*) INTO stop_count FROM stops;` at
  `004:139`) runs during `004`, before the rename. Untouched.

### 4.3 Every Go query string the migration invalidates

**These must change in the SAME commit as the migration.** Each is a runtime-only
dependency: the Go compiler cannot see inside a SQL string, so the build stays green while
production breaks. This is the highest-risk step in the phase.

| File:line | Statement | Change |
|---|---|---|
| `internal/database/stops.go:25` (→ `zones.go`) | `FROM stops` — `GetByRouteID` | `FROM zones` |
| `internal/database/stops.go:56` | `FROM stops` — `GetByID` | `FROM zones` |
| `internal/database/stops.go:73` | `FROM stops` — `GetNearby` | `FROM zones` |
| `internal/database/stops.go:107` | `FROM stops` — `GetAll` | `FROM zones` |
| `internal/database/routes.go:65` | `INSERT INTO stops (id, route_id, name, latitude, longitude, sequence_number, geom)` | `INSERT INTO zones (…)` |

**Five statements. That is the complete list** — verified by
`grep -rn 'stops' backend/internal/database/*.go`, discarding comments and Go-local
identifiers. No other package embeds SQL: `trips.go` joins `devices` and `location_history`
only; `locations.go` joins `devices`.

**Column names are unchanged** (`id, name, latitude, longitude, sequence_number, route_id,
geom`), so only the table token moves in each string.

---

## 5. The four commits

### 5.0 Read this first — §6.3's split does not compile as literally written

§6.3 says commit 1 is "Go models + database" and to "verify the build between each." Go is
statically typed: the moment `models.Stop` becomes `models.Zone`, **every** package
referencing it fails to compile — `services`, `handlers`, `cmd/server`, and four test files.
A commit containing only `models/` and `database/` changes **cannot** build.

**Resolution, which preserves §6.3's four commits and keeps each one green:**

> Commit *N* renames the identifiers **owned by** its layer, and updates **references** to
> those identifiers everywhere in the tree, including in layers whose own type names are
> renamed later.

So Commit 1 turns `models.Stop` into `models.Zone` *and* rewrites `services.StopRepository`'s
method signatures to say `models.Zone` — while `services.StopRepository` keeps its own name
until Commit 2. Every commit compiles; no commit is a partial rename of a single identifier.

---

### Commit 1 — Go models + database layer

**Files changed**
- `internal/models/stop.go` → **`internal/models/zone.go`** (`Stop`→`Zone`)
- `internal/models/location.go` (`NearbyBus`→`NearbyDevice`)
- `internal/models/route.go` (`CreateStopInput`→`CreateZoneInput`, `.Stops`→`.Zones`,
  `.StopCount`→`.ZoneCount`, `.StopIDs`→`.ZoneIDs`; **all `json:` tags unchanged**)
- `internal/database/stops.go` → **`internal/database/zones.go`** (`StopRepository`→
  `ZoneRepository`, `NewStopRepository`→`NewZoneRepository`; **SQL strings still say
  `stops`** — the table is renamed in Commit 3)
- `internal/database/locations.go` (`GetBusesInHex`/`InHexes`/`NearStop` →
  `GetDevicesInHex`/`InHexes`/`NearZone`)
- `internal/database/trips.go` (`GetActiveTripsBeforeStop`→`GetActiveTripsBeforeZone`)
- `internal/database/routes.go` (`req.Stops`→`req.Zones`, `stopIDs`→`zoneIDs`)
- `internal/cache/interface.go` (`NearbyBusesTTL`→`NearbyDevicesTTL`)
- **Reference-only updates**: `internal/services/interfaces.go`,
  `internal/services/{arrivals,geofencing,stops}.go`, `internal/handlers/interfaces.go`,
  `internal/handlers/{stops,nearby,routes,arrivals}.go`, `cmd/server/main.go`,
  `internal/services/{geofencing_method,arrivals,stops,routes}_test.go`

**Pre-flight**
```bash
cd backend
git status --short                      # clean except untracked docs
go build ./... && go vet ./... && test -z "$(gofmt -l .)"
go test ./... -race -count=1
```

**Verification (must pass before Commit 2)**
```bash
cd backend
go build ./...                                  # exit 0
go vet ./...                                    # exit 0
test -z "$(gofmt -l .)"                         # exit 0
go test ./... -race -count=1                    # exit 0, same pass/fail set as pre-flight
go vet -tags=integration ./...                  # exit 0
grep -rn 'NearbyBus\|CreateStopInput\|StopCount\|StopIDs' cmd/ internal/ pkg/ test/ --include='*.go'
                                                # → no output
grep -rn 'models\.Stop\b' cmd/ internal/ pkg/ test/ --include='*.go'
                                                # → no output
bash ../scripts/gate.sh --check=layering        # exit 0
```

---

### Commit 2 — Go services + handlers

**Files changed**
- `internal/services/stops.go` → **`internal/services/zones.go`**
- `internal/services/arrivals.go` → **`internal/services/geofence.go`**
- `internal/services/geofencing.go` (methods per §1.2; **filename unchanged** — it is already
  generic)
- `internal/services/interfaces.go` (`StopRepository`→`ZoneRepository`)
- `internal/handlers/stops.go` → **`internal/handlers/zones.go`**
- `internal/handlers/arrivals.go` → **`internal/handlers/geofence.go`**
- `internal/handlers/interfaces.go`, `internal/handlers/nearby.go`
- `cmd/server/main.go` (all five compile-time assertions at `:31-41`, constructors at
  `:79,89,90,104,105`, **route registrations at `:138,141,144,145` change only the handler
  method name, never the path string**)
- `internal/services/{stops,arrivals,geofencing_method}_test.go` (rename test funcs + mocks)

**Pre-flight:** Commit 1's verification block, all green.

**Verification (must pass before Commit 3)**
```bash
cd backend
go build ./... && go vet ./... && test -z "$(gofmt -l .)"
go test ./... -race -count=1
go vet -tags=integration ./...
grep -rn 'Bus\|Buses\|Arrival' cmd/ internal/ pkg/ test/ --include='*.go'
   # → only: nearby.go's /api/nearby/buses path string + its doc comment,
   #         apiConfig-facing comments, and "arrivals" route path/query-param strings
grep -rn '\bStop\b' cmd/ internal/ pkg/ test/ --include='*.go'
   # → EXACTLY the 5 false positives from §0.4 (2x ticker.Stop, 1 comment, 2 test strings)
bash ../scripts/gate.sh --check=layering
curl -s localhost:8080/api/stops >/dev/null    # only if a server is up; path must still work
```

---

### Commit 3 — SQL migration + the Go query strings it invalidates

**Files changed**
- **new** `backend/migrations/005_rename_stops_zones.sql` (§4.2)
- `internal/database/zones.go` — 4 × `FROM stops` → `FROM zones` (was `:25,56,73,107`)
- `internal/database/routes.go` — 1 × `INSERT INTO stops` → `INSERT INTO zones` (was `:65`)

**Nothing else.** Do not edit `001`–`004`.

**Pre-flight**
```bash
cd backend
docker version                          # daemon must be up — this commit's check needs it
go build ./...
grep -rn 'FROM stops\|INTO stops' internal/ --include='*.go' | wc -l   # → 5
```

**Verification — a build is NOT sufficient here**
```bash
cd backend
go build ./... && go vet ./... && test -z "$(gofmt -l .)"
go test ./... -race -count=1

# The one that actually matters: real Postgres, real migrations, real queries.
go test -tags=integration ./test/... -v -count=1 -timeout 120s     # exit 0

grep -rn 'stops' internal/ --include='*.go'
   # → comments only; zero SQL strings
```

> **Why the integration test is the gate.** A stale `FROM stops` compiles perfectly and fails
> only at runtime. `backend/test/integration_test.go` spins up the real TimescaleDB
> container, applies `001`→`005` through the real `RunMigrations`, and exercises the stack.
> It does **not** currently touch the zones table, so **extend it in this commit**: after the
> existing assertions, add a `SELECT count(*) FROM zones` and assert it returns the 15
> seeded rows. Without that, this commit has no automated proof at all.

---

### Commit 4 — Frontend

**Files changed:** the 12 renames from §2.1, the 16 reference-only files from §2.1, and the
symbol renames from §2.2. `apiConfig.ts`/`wsConfig.ts` change **keys and comments only** —
every path and schema *value* string stays byte-identical (§3.1).

**Pre-flight**
```bash
cd frontend
npm run build        # baseline — currently passes: "✓ built in 798ms", 167 modules
npm run lint
```

**Verification (phase exit)**
```bash
cd frontend
npm run build        # exit 0; module count still 167
npm run lint         # exit 0, no new warnings
grep -rn 'Bus\|bus\|Arrival\|arrival' src/
   # → only: apiConfig/wsConfig path+schema VALUES, user-facing UI copy, and
   #         MapCameraHandler's stopPropagation
grep -rln 'BusLocation\|ArrivalsSlice\|StopsSlice\|selectedStopId\|followedBusId\|routeCreatorStops' src/
   # → no output
git status --short   # renames staged as R, not D+A (use `git mv`)
```

---

## 6. The primary tripwire — test files

Phase 1B/1C added test files that reference these identifiers. **§0.5 corrects the scale**:
four files, 140 match lines, all in `internal/services/`. The integration test needs no
rename at all.

> ### If a test's ASSERTIONS change, the rename was not mechanical. Stop and revert.
>
> Renaming a symbol changes **names**: identifiers, mock type names, `t.Run` labels, test
> function names. It must **never** change:
> - an expected value (`if got != 2` stays `!= 2`)
> - a comparison operator or a boundary
> - the number of assertions, or which branch a case exercises
> - a table-driven case's input coordinates or expected output
> - `DroppedMessages` counts, timeout constants, buffer sizes
>
> The mechanical check, run between every commit:
> ```bash
> cd backend
> go test ./... -race -count=1 2>&1 | tee /tmp/after.txt
> diff <(grep -E '^(ok|FAIL|---)' /tmp/before.txt) <(grep -E '^(ok|FAIL|---)' /tmp/after.txt)
> ```
> Capture `/tmp/before.txt` in pre-flight. **The set of passing test names must differ only
> where a test was deliberately renamed.** A test that starts *passing*, or one whose
> subtests change count, is as much a failure signal as one that breaks.

Two specific traps in these files:

1. **`geofencing_method_test.go:112-176`** computes real H3 cells and asserts hex-boundary
   behavior (`IsAtStop`, `IsAtStopWithHysteresis`, neighbour cells). The lat/lng constants
   `28.61395, 77.20905` etc. are **load-bearing** — they were chosen to land in specific
   hexes. Renaming `stopLat`→`zoneLat` must not touch a single digit.
2. **`routes_test.go:60-79`** contains fixture strings `"Stop 1"`, `"Stop 2"` and asserts
   `res.StopCount != 2`. The *field* becomes `ZoneCount`; the *literal `2`* and the fixture
   strings stay (§0.4).

---

## 7. Rename mechanism

### 7.1 Neither preferred tool is installed

```
$ which gopls gorename
gopls not found
gorename not found
```

**Install `gopls` first** — it is the only tool that renames Go identifiers correctly across
packages while leaving strings, comments, and unrelated same-named symbols alone:

```bash
go install golang.org/x/tools/gopls@latest
export PATH="$PATH:$(go env GOPATH)/bin"
gopls version
```

Then, per identifier, from `backend/`:

```bash
gopls rename -w ./internal/models/stop.go:3:6 Zone
#                └ file:line:col of the DECLARATION       └ new name
```

`gopls rename` is type-aware: it will refuse an unsafe rename (collision, shadowing) rather
than silently corrupting the tree. Renaming `models.Stop` updates all 42 references across
every package in one operation. **Do the file `git mv` separately, after the symbol rename.**

For the frontend, use TypeScript's rename-symbol via the language server (VS Code F2, or
`typescript-language-server` if driving headlessly). `tsc -b` then proves the result.

### 7.2 Fallback if `gopls` cannot be installed

§6.3 forbids blind tree-wide `sed -i` — it hits strings, comments, and seed data. The
fallback is **targeted, one identifier at a time, with an explicit exclusion list and a build
between each**:

```bash
# ONE identifier. Note the word boundaries and the exclusions.
cd backend
grep -rl 'GetBusesInHexes' cmd/ internal/ pkg/ test/ --include='*.go' \
  | grep -v 'internal/hub/'            \
  | xargs sed -i '' 's/\bGetBusesInHexes\b/GetDevicesInHexes/g'
go build ./... && go test ./... -race -count=1     # AFTER EVERY SINGLE IDENTIFIER
```

**Permanent exclusion list for every `sed` invocation:**

| Exclude | Reason |
|---|---|
| `backend/migrations/004_seed_data.sql` | demo fixtures (§3) |
| `backend/migrations/001`–`003` | never edit an applied migration |
| `backend/internal/hub/` | only stdlib `ticker.Stop()` + prose (§0.4) |
| `backend/internal/hub/message_test.go` | DEFECT-2 `heading` guard |
| any line containing `json:"` | wire tags (§3.1) |
| any line containing `/api/` | route paths (§3.1) |
| `frontend/src/config/apiConfig.ts` schema **values** | backend field names (§3.1) |

**Ordering rule for `sed`:** always rename the **longest** identifier first.
`GetBusesInHexes` before `GetBusesInHex`, `IsAtStopWithHysteresis` before `IsAtStop`,
`StopsService` before `Stop`. Reversing this corrupts the longer name.

`\b` word boundaries are mandatory. On macOS, `sed -i ''` (with the empty argument);
on GNU/Linux, `sed -i`.

---

## 8. Full-phase exit criteria

1. All four commits landed, in order, each with its verification block green **at the time it
   was committed** (not merely at the end).
2. From `backend/`:
   ```bash
   grep -rn 'Bus\|Buses\|Arrival' cmd/ internal/ pkg/ test/ --include='*.go'
   grep -rn '\bStop\b\|Stops\b' cmd/ internal/ pkg/ test/ --include='*.go'
   ```
   returns **only**: the five §0.4 false positives, the `/api/nearby/buses` and
   `/api/stops` / `/api/arrivals` route-path strings with `stop_id`, and comments that
   deliberately reference the old vocabulary or the wire contract.
3. `go build ./...`, `go vet ./...`, `go vet -tags=integration ./...`,
   `gofmt -l .` (empty), `go test ./... -race -count=1` — all clean.
4. `go test -tags=integration ./test/... -count=1 -timeout 120s` passes **against the renamed
   table**, including the new `zones` row-count assertion from Commit 3.
5. `cd frontend && npm run build && npm run lint` — clean, 167 modules.
6. `bash scripts/gate.sh --check=layering` — exit 0.
7. **Behavior is unchanged**: the same HTTP paths, the same JSON field names, the same
   WebSocket envelope (§7.2 of `CLAUDE.md`), the same test assertions.
8. `CLAUDE.md` §6 updated in the final commit to reflect what was actually done — including
   deleting the `models.ArrivalEvent` row, correcting `006`→`005`, adding the identifiers
   from §0.3, and noting `cmd/` in the verify command. Leaving §6 as-is after executing
   against a corrected plan would recreate the exact drift this phase's own process is
   supposed to prevent.

---

## 9. New findings — append to `IDEAS.md`, do not fix here

**D35 — `LocationRepository.GetBusesInHex` is dead code.**
`backend/internal/database/locations.go:79`. Two occurrences total: the declaration and its
doc comment. It is absent from `services.LocationRepository` (`services/interfaces.go:25-30`,
which declares only `GetBusesInHexes` and `GetBusesNearStop`) and is called nowhere. Phase 2
renames it to `GetDevicesInHex` mechanically rather than deleting it, because deletion is a
scope decision, not a rename. Decide separately whether the single-hex variant should exist
at all.

**D36 — `cache.NearbyBusesTTL` is dead.**
`backend/internal/cache/interface.go:36`. Declared, never read — the only TTL actually used
is `DeviceLocationTTL` (`redis.go:57`). Same treatment as D35: renamed, not deleted.

**D37 — `CLAUDE.md` §6 was stale in five distinct ways before Phase 2 began.**
Recorded because the *pattern* matters more than the individual errors: (a) the
`models.ArrivalEvent` row names a type that no longer exists; (b) `busHex` in the locals row
never existed; (c) the `006` migration number was a guess that the check disproved — it is
`005`; (d) §6.1 omitted ~22 in-scope identifiers, including all of `StopsService`,
`ArrivalPrediction`, and `IsAtStop`; (e) §6.2 lists only renamed files and omits the 16
frontend files needing reference updates, one of which (`ui.ts`, 39 matches) has more matches
than any file it does list. Also, §6.3's stated verify command greps `backend/internal
backend/pkg` and omits `backend/cmd`, where main.go holds 13 matches.

**D38 — `IDEAS.md` D10 and D16 are stale.**
D10 says `models.Trip` is dead code; D16 says `models.ArrivalEvent` is dead at
`models/trip.go:11`. Both types have since been **removed** — `models/trip.go` now contains
only `TripWithLocation`. "Dead" and "absent" need different follow-ups; these entries should
be closed, not carried.

**D39 — `trips.current_stop` keeps transit vocabulary after Phase 2.**
`migrations/001_create_tables.sql:51`. Renaming it would mean editing `004_seed_data.sql`'s
INSERT column list, which §6.3 forbids, so Phase 2 leaves it. After this phase the schema is
mixed: a `zones` table alongside a `current_stop` column. Resolve in whichever phase next
touches `trips` (§5.5 currently scopes `trips.go` out).
