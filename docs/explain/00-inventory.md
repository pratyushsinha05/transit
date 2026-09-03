# 00-inventory.md — Repository inventory

**Source of record:** this document was assembled **exclusively** from the nine zone
reports in `docs/explain/raw/` (`zone-1.md`, `zone-2.md`, `zone-3.md`, `zone-4.md`,
`zone-4-recheck.md`, `zone-5.md`, `zone-6.md`, `zone-7.md`, `zone-gap.md`). No source
file was opened while writing it. That is deliberate: every claim below must be
traceable through a citable intermediate rather than through a second, unrecorded read.

**Citation convention.** Every claim carries `path:line` followed by the zone report it
came from, e.g. `backend/cmd/server/main.go:95` [z1]. Keys: `[z1]`…`[z7]` = `zone-N.md`,
`[z4r]` = `zone-4-recheck.md`, `[zg]` = `zone-gap.md`.

**Consequence of the method.** Where two zone reports disagree about a line number, this
document records both and does not adjudicate. Those disagreements are listed in §7.8.
Where a zone report recorded no consumer for a symbol, this document says exactly that —
"no zone report records a caller" — which is weaker than "unreachable" and is not
upgraded to it.

---

## 1. File census

### 1.1 Repository root

| File | Lines | Contents |
|---|---|---|
| `Makefile` | 718 [z6][zg] | ~70 targets: dependency checks, Docker lifecycle, DB migrate/seed/reset, test/fmt/vet/lint, curl-based API smoke targets, cleanup. `.DEFAULT_GOAL := help` at `Makefile:679` [zg] |
| `deploy.sh` | 121 [z6] | 3 phases: prerequisite check (`deploy.sh:25-41`), `make check-deps` + `make docker-up` (`:47-65`), `npm install` + `npm run build` (`:70-96`), then `npm run preview` (`:120`) [z6] |
| `cleanup.sh` | 75 [z6] | `make clean-all` or manual compose teardown (`cleanup.sh:25-35`); removes `frontend/dist` and `frontend/node_modules` (`:42-61`) [z6] |
| `.env.example` | 72 [z7] | 22 env vars with defaults; H3 resolution reference table in comments at lines 37–44 [z7] |
| `.gitignore` | 7 [z6] | Python cache, binary output [z6] |
| `CLAUDE.md` | 706 [z7] | Project contract. Section outline at `zone-7.md` §14 [z7] |
| `BASELINE.md` | 496 [z7] | Phase 0 ground-truth record [z7] |
| `IDEAS.md` | 107 [z7] | Deferred items D4–D13, C7, C8 [z7] |
| `README.md` | 102 [z7] | Deploy/Makefile/frontend instructions [z7] |
| `=` | 7 bytes [z7] | Stray file, literal filename `=`, sole content `31.3.2` [z7] |

### 1.2 `backend/` (top level)

| File | Lines | Contents |
|---|---|---|
| `backend/Dockerfile` | 80 [z6][zg] | 2 stages. Builder `golang:1.23-alpine` (`:4`), `CGO_ENABLED=1` build (`:28-34`); runtime `alpine:latest` (`:39`), non-root `appuser` (`:50-51,69`), `EXPOSE 8080` (`:72`), healthcheck wget `/health` (`:75-76`), `CMD ["./server"]` (`:79`) [zg] |
| `backend/.dockerignore` | 36 [z7] / 37 [zg] | `.env*`, build artifacts, IDE/OS files, Go binaries, temp files [zg] |
| `backend/go.mod` | 31 [z7][zg] | `module transit-backend` (`:1`), `go 1.23.0` (`:3`), 6 direct requires (`:6-11`), indirect block (`:15-30`) [z7][zg] |
| `backend/go.sum` | 63 [z7][zg] | Generated lockfile [z7] |
| `backend/healthcheck.sh` | 36 [z6] | wget (`:20`) → curl (`:23`) → nc (`:27`) fallback chain against `${HOST}:${PORT}${ENDPOINT}` [z6] |

### 1.3 `backend/cmd/server/`

| File | Lines | Contents |
|---|---|---|
| `main.go` | 179 [z1] | Composition root. 11 compile-time interface assertions (`:31-41`), config→DB→migrations→Redis→repos→cache→services→hub→ingest→handlers→Echo→routes→start→graceful shutdown (`:44-178`) [z1] |

### 1.4 `backend/internal/cache/`

| File | Lines | Contents |
|---|---|---|
| `interface.go` | 71 [z3] | `DeviceLocation` struct (`:9-15`), `Session` struct (`:18-22`), `CacheStore` interface with 10 methods (`:26-46`), 4 TTL constants (`:52,55,58,61`), 4 key-prefix constants (`:66-69`) [z3] |
| `redis.go` | 228 [z3] | `RedisCache` (`:16-18`) implementing all of `CacheStore`; `New(cfg)` (`:21-44`); `DeviceCache` wrapper (`:174-176`) with `SetDeviceState` (`:184-205`) / `GetDeviceState` (`:208-224`), commented as legacy/deprecated at `:172-173`; assertion `var _ CacheStore = (*RedisCache)(nil)` (`:227`) [z3] |

### 1.5 `backend/internal/config/`

| File | Lines | Contents |
|---|---|---|
| `config.go` | 106 [z1] | `Config` struct, 16 fields (`:12-38`); `LoadConfig()` (`:41-87`); `getEnvOrDefault` (`:90-95`); `getEnvAsInt` (`:98-105`). Validates `DBName` (`:74-75`), `DBUser` (`:77-78`), `H3Resolution` 0–15 (`:82-84`) [z1] |

### 1.6 `backend/internal/database/`

| File | Lines | Contents |
|---|---|---|
| `db.go` | 106 [z2] | `New(cfg)` pool constructor (`:17-51`); `RunMigrations(ctx, pool, fs.FS)` (`:55-105`) with `schema_migrations` ledger (`:56-61`) [z2] |
| `device_routes.go` | 45 [z2] | `DeviceRouteRepository` (`:18-20`), `GetActiveRouteID` (`:29-44`) [z2] |
| `locations.go` | 247 [z2] | `LocationRepository` (`:14-17`); `Insert` (`:47-75`), `GetBusesInHex` (`:79-111`), `GetBusesInHexes` (`:114-150`), `GetBusesNearStop` (`:154-196`), `GetLatestLocation` (`:199-224`), `GetNeighborHexes` (`:227-246`), `CalculateHex` (`:37-44`) [z2] |
| `routes.go` | 91 [z2] | `RouteRepository` (`:11-13`); `GetAll` (`:19-36`), `Create` (`:40-90`, transactional) [z2] |
| `stops.go` | 128 [z2] | `StopRepository` (`:12-14`); `GetByRouteID` (`:22-50`), `GetByID` (`:53-65`), `GetNearby` (`:68-101`), `GetAll` (`:104-127`) [z2] |
| `trips.go` | 65 [z2] | `TripRepository` (`:9-11`), `TripWithLocation` (`:17-24`), `GetActiveTripsBeforeStop` (`:26-64`) [z2] |

### 1.7 `backend/internal/handlers/`

| File | Lines | Contents |
|---|---|---|
| `interfaces.go` | 40 [z3] | 5 consumer-side service interfaces: `ArrivalService` (`:13-15`), `StopsService` (`:18-20`), `RoutesService` (`:23-26`), `IngestService` (`:29-31`), `NearbyService` (`:34-39`) [z3] |
| `arrivals.go` | 36 [z3] | `ArrivalHandler{service ArrivalService}` (`:11-13`), `GetArrivals` (`:22-35`) [z3] |
| `location.go` | 62 [z3] | `LocationHandler{service IngestService}` (`:15-17`), `IngestLocation` (`:27-61`) with range validation (`:34-46`) [z3] |
| `nearby.go` | 184 [z3] | `NearbyHandler{geoService NearbyService}` (`:15-17`); `GetNearbyBuses` (`:26-85`), `GetNearbyStops` (`:89-142`), `GetHexInfo` (`:146-183`) [z3] |
| `routes.go` | 66 [z3] | `RouteHandler{service RoutesService}` (`:13-15`); `GetRoutes` (`:24-31`), `CreateRoute` (`:35-65`) [z3] |
| `stops.go` | 33 [z3] | `StopHandler{service StopsService}` (`:11-13`), `GetStops` (`:19-32`) [z3] |
| `websocket.go` | 46 [z3] | `WebSocketHandler{hub *hub.Hub}` (`:21-23`) — **concrete type, not an interface**; `upgrader` with `CheckOrigin` always true (`:13-19`); `HandleWS` (`:29-45`) [z3] |

### 1.8 `backend/internal/hub/`

| File | Lines | Contents |
|---|---|---|
| `hub.go` | 55 [z3] | `Hub` struct, 5 fields (`:7-13`); `New()` (`:15-22`) — `Broadcast` buffered 256 (`:17`), `Register`/`Unregister` unbuffered (`:18-19`); `Run()` (`:24-54`) — infinite `for{select{}}`, **no ctx/done case**; silent-drop `default:` on full client buffer (`:43-49`) [z3] |
| `client.go` | 94 [z3] | `Client` struct (`:29-37`); timing constants `writeWait=10s` (`:12`), `pongWait=60s` (`:15`), `pingPeriod` (`:18`), `maxMessageSize=512` (`:21`); `ReadPump` (`:43-60`), `WritePump` (`:66-93`) — both infinite loops with no ctx/done [z3] |
| `message.go` | 22 [z3] | `Message` struct, 9 fields, **no `omitempty` on any field** (`:7-17`); `MsgTypeLocationUpdate = "LOCATION_UPDATE"` (`:20`) [z3] |
| `message_test.go` | 85 [z3] | `TestMessage_JSONShape` (`:12-62`) — asserts 9 keys, no `data` wrapper (`:48-49`), no `heading` key (`:51-52`), `speed` present and `0` (`:55-61`); `TestMessage_ZeroCoordinatesNotOmitted` (`:64-84`) [z3] |

### 1.9 `backend/internal/middleware/`

| File | Lines | Contents |
|---|---|---|
| `cors.go` | 14 [z1] | `CORS()` (`:8-14`): `AllowOrigins: ["*"]`, methods GET/POST/OPTIONS, header Content-Type (`:10-12`) [z1] |
| `errors.go` | 27 [z1] | `HTTPErrorHandler` (`:9-20`) → JSON `{"error": …}` (`:19`); `result()` helper (`:22-27`) [z1] |
| `logging.go` | 30 [z1] | `Logging` (`:10-30`); `log.Printf` of method/path/status/latency_ms (`:25`) [z1] |
| `recovery.go` | 32 [z1] | `Recovery` (`:12-31`); recovers panic, logs `debug.Stack()` (`:17`), emits 500 (`:25`) [z1] |

### 1.10 `backend/internal/models/`

| File | Lines | Contents |
|---|---|---|
| `device.go` | 8 [z4][z4r] | `Device{ID, Name, Status}` (`:3-7`) [z4r] |
| `location.go` | 36 [z4][z4r] | `Location` (`:4-12`), `LocationUpdate` (`:15-24`), `NearbyBus` (`:27-35`) [z4r] |
| `route.go` | 31 [z4][z4r] | `Route` (`:3-7`), `CreateRouteRequest` (`:10-14`), `CreateStopInput` (`:17-21`), `CreateRouteResponse` (`:24-30`) [z4r] |
| `stop.go` | 10 [z4][z4r] | `Stop{ID, Name, Latitude, Longitude, Sequence}` (`:3-9`) [z4r] |
| `trip.go` | 17 [z4][z4r] | `Trip` (`:3-9`), `ArrivalEvent` (`:11-16`) [z4r] |

### 1.11 `backend/internal/services/`

| File | Lines | Contents |
|---|---|---|
| `interfaces.go` | 51 [z4][z4r] | 6 consumer-side repository interfaces: `StopRepository` (`:13-17`), `TripRepository` (`:20-22`), `LocationRepository` (`:26-31`), `RouteRepository` (`:34-37`), `DeviceCache` (`:41-43`), `DeviceRouteRepository` (`:48-50`). Imports `transit-backend/internal/database` (`:6`) for `database.TripWithLocation` [z4r] |
| `arrivals.go` | 166 [z4][z4r] | `ArrivalsService` (`:10-15`) — `geoService` field is concrete `*GeofencingService` (`:14`); `ArrivalPrediction` (`:33-42`); `GetArrivalsForStop` (`:44-82`), `GetNearbyArrivals` (`:84-111`), `isApproaching` (`:113-131`), `CalculateETAWithTraffic` (`:133-146`), `DetectArrivalEvent` (`:148-165`) [z4r] |
| `geofencing.go` | 224 [z4][z4r] | `GeofencingService` (`:11-15`); 13 functions/methods including `CalculateHex` (`:35-44`), `CalculateHexAtResolution` (`:46-54`), `IsAtStop` (`:56-62`), `IsAtStopWithHysteresis` (`:64-100`), `GetNeighborHexes` (`:102-121`), `FindNearbyBuses` (`:123-156`), `FindNearbyStops` (`:158-163`), `DetectArrival` (`:165-178`), `DetectDeparture` (`:180-190`), `GetHexResolution` (`:192-195`), `HexEdgeLengthMeters` (`:197-223`) [z4r] |
| `ingest.go` | 82 [z4][z4r] | `IngestService` (`:17-22`) — `hub` field is concrete `*hub.Hub` (`:21`); `IngestLocation` (`:29-81`) [z4r] |
| `routes.go` | 28 [z4][z4r] | `RoutesService` (`:10-12`); `GetRoutes` (`:19-22`), `CreateRoute` (`:24-27`) [z4r] |
| `stops.go` | 23 [z4][z4r] | `StopsService` (`:10-12`); `GetStopsByRoute` (`:19-22`) [z4r] |
| `geofencing_test.go` | 148 [z4][z4r] | 6 tests, all against the two pure functions `CalculateHexAtResolution` and `HexEdgeLengthMeters` (`:7,65,89,102,117,130`) [z4r] |

### 1.12 `backend/pkg/geo/`

| File | Lines | Contents |
|---|---|---|
| `distance.go` | 38 [z4][z4r] | Constants `EarthRadiusKm=6371.0`, `MinSpeedKmh=1.0`, `DefaultSpeed=20.0` (`:5-7` per [z4r]; `:6-8` per [z4]); `Haversine` (`:12-24`), `CalculateETA` (`:26-37`) [z4r] |
| `distance_test.go` | 45 [z4][z4r] | `TestHaversine` (`:7-21`), `TestCalculateETA` (`:23-44`) incl. `DefaultSpeed` fallback branch (`:39-41`) [z4r] |

### 1.13 `backend/migrations/`

| File | Lines | Contents |
|---|---|---|
| `migrations.go` | 7 [z2] | `//go:embed *.sql` → `var Files embed.FS` (`:5-6`) [z2] |
| `001_create_tables.sql` | 54 [z2] | Extensions `timescaledb` (`:2`), `postgis` (`:3`); tables `devices` (`:6-11`), `routes` (`:14-18`), `stops` (`:21-29`), `location_history` (`:32-40`), `trips` (`:46-53`); `create_hypertable` (`:43`) [z2] |
| `002_add_indexes.sql` | 6 [z2] | `idx_stops_geom` GIST (`:2`), `idx_location_history_device_time` (`:5`) [z2] |
| `003_add_h3_postgis.sql` | 119 [z2] | `ADD COLUMN hex_res9` (`:7`); 5 indexes (`:28-41,62-63`); `UPDATE stops SET geom` (`:57-59`); compression settings (`:76-79`) + 7-day policy (`:87`); `update_location_geom()` function (`:97-105`); `trg_location_geom` trigger (`:108-118`) [z2] |
| `004_seed_data.sql` | 140 [z2] | Seeds 5 devices (`:12-18`), 3 routes (`:23-27`), 15 stops (`:33-42,47-56,61-70`), stops geom update (`:75-77`), 4 trips (`:82-87`), 11 `location_history` rows (`:94-99,102-105,108-111`), verification DO block (`:122-139`) [z2] |

### 1.14 `infra/`

| File | Lines | Contents |
|---|---|---|
| `docker-compose.yml` | 126 [z6] | 3 services — `postgres` `timescale/timescaledb-ha:pg15-latest` (`:12-32`), `redis` `redis:7-alpine` (`:36-57`), `backend` (`:62-109`); volumes (`:114-118`); `transit-network` bridge (`:123-125`) [z6] |

### 1.15 `frontend/` (top level)

| File | Lines | Contents |
|---|---|---|
| `index.html` | 18 [z7] / 19 [zg] | Favicon `/vite.svg` (`:5`), Space Mono webfont (`:10-12`), `<div id="root">` (`:15`), single module script `/src/main.tsx` (`:16`) [z7][zg] |
| `package.json` | 42 [z6] | Scripts `dev`/`build`/`lint`/`preview` (`:7-10`); 9 dependencies (`:13-21`); 16 devDependencies (`:24-39`) [z6] |
| `package-lock.json` | — [z6] | Generated, `lockfileVersion: 3` (`:4`) [z6] |
| `eslint.config.js` | 23 [z6] | Flat config, ignores `dist` (`:9`), extends js + tseslint + react-hooks + react-refresh (`:13-16`) [z6] |
| `postcss.config.js` | 6 [z6] | `tailwindcss` (`:3`), `autoprefixer` (`:4`) [z6] |
| `tailwind.config.js` | 60 [z6] | `hud` palette (`:9-23`), Space Mono font stack (`:25-26`), 4 keyframes (`:32-48`) + animations (`:50-55`) [z6] |
| `tsconfig.json` | 7 [z6] | Project references only (`:3-6`) [z6] |
| `tsconfig.app.json` | 32 [z6] | `strict: true` (`:20`), `noUnusedLocals`/`noUnusedParameters` (`:21-22`), path alias `@/*` (`:27-29`) [z6] |
| `tsconfig.node.json` | 26 [z6] | Covers `vite.config.ts` (`:25`) [z6] |
| `vite.config.ts` | 13 [z6] | React plugin (`:7`), `@` → `./src` alias (`:8-11`) [z6] |
| `.gitignore` | 25 [z6][zg] | logs, `node_modules`, `dist`, IDE/OS artifacts [zg] |
| `README.md` | 73 [z7] | Unmodified Vite + React + TS template README; no project-specific claim [z7] |

### 1.16 `frontend/src/` (root files and assets)

| File | Lines | Contents |
|---|---|---|
| `main.tsx` | 9 [z7] | `createRoot(getElementById('root')!)` → `<StrictMode><App /></StrictMode>` (`:5-9`) [z7] |
| `App.tsx` | 42 [z7] | Static layout, no router, no providers: `Sidebar` (`:16`), `MapContainer` (`:20`), `ConnectionStatus` (`:25`), `OperationsPanel` (`:31`), `TelemetryTray` (`:36`); imports `./styles/globals.css` (`:6`), **not** `./App.css` [z7] |
| `App.css` | 42 [z7] / 43 [zg] | Vite starter boilerplate: `#root`, `.logo`, `.card`, `.read-the-docs`, `logo-spin` keyframes [z7][zg] |
| `index.css` | 12 [z7] / 13 [zg] | `:root { color-scheme: dark }` (`:2-3`), `body` reset (`:6`) [z7][zg] |
| `styles/globals.css` | 245 [z7] / 246 [zg] | `@tailwind` directives (`:1-3`), 10 `--hud-*` custom properties (`:9-31`), 30 selector blocks + 2 keyframes (`radar-ping` `:105`, `scanline-sweep` `:158`); base64 SVG `feTurbulence` noise in `.noise-overlay` (`:215-224`) [z7] |
| `public/vite.svg` | 1497 bytes [zg] | Favicon, referenced from `index.html:5` [z7][zg] |
| `assets/react.svg` | 4126 bytes [zg] | Vite template asset; `grep` for `react.svg` under `frontend/src/` returned no matches [z7] |
| `utils/` | — [z6] | Directory exists and is **empty** [z6] |

### 1.17 `frontend/src/components/`

| File | Lines | Contents |
|---|---|---|
| `Footer/TelemetryTray.tsx` | 91 [z5] | Clock `setInterval` (`:23`), reads `connection` (`:11`) [z5] |
| `Header/ConnectionStatus.tsx` | 51 [z5] | Calls `useHealthCheck()` (`:10`) and `useWebSocket()` (`:11`) [z5] |
| `Map/MapContainer.tsx` | 65 [z5] | `<LeafletMap zoom={13}>` (`:36`), CartoDB dark tiles (`:42-47`), mounts 6 child components (`:50-61`). **Reads no store state at all** [z5] |
| `Map/BusMarkers.tsx` | 108 [z5] | `createBusBlipIcon()` (`:12-25`); reads `buses` (`:28`), `layerVisibility` (`:29`), `hiddenRoutes` (`:30`); `setFollowedBusId` on click (`:89`) [z5] |
| `Map/StopMarkers.tsx` | 115 [z5] | `StopPopupContent` inline component (`:32`) calling `useArrivals(stopId)` (`:33`); reads `layerVisibility` (`:88`); `setSelectedStopId` on click (`:100`) [z5] |
| `Map/RoutePolyline.tsx` | 72 [z5] | `getCoordinates()` (`:12-18`); three stacked `<Polyline>` layers (`:34-65`) [z5] |
| `Map/RouteCreatorMarkers.tsx` | 133 [z5] | OSRM fetch effect (`:49-77`) calling `getRouteGeometry` (`:65`); draggable numbered markers (`:116-129`) [z5] |
| `Map/MapCameraHandler.tsx` | 83 [z5] | Follow-camera effect (`:22-33`); drag cancels follow (`:36-47`) [z5] |
| `Map/MapClickHandler.tsx` | 23 [z5] | `useMapEvents` click → `addCreatorStop` (`:13-16`) [z5] |
| `Sidebar/Sidebar.tsx` | 180 [z5] | Reads `selectedStopId`, `connection`, `buses`, `routeCreatorMode` (`:12-18`); mounts `ArrivalsList`, `RouteCreatorPanel` [z5] |
| `Sidebar/OperationsPanel.tsx` | 103 [z5] | **Two** layer toggles only: `stops` (`:40-46`) and `buses` (`:49-56`); route visibility list (`:71-90`) [z5] |
| `Sidebar/RouteCreatorPanel.tsx` | 237 [z5] | Route creator form; `createRoute(...)` submit (`:37`) [z5] |
| `Sidebar/ArrivalsList.tsx` | 97 [z5] | `useArrivals(stopId)` (`:15`), renders `ArrivalCard` [z5] |
| `Sidebar/ArrivalCard.tsx` | 70 [z5] | Presentational; takes `arrival: Arrival` prop (`:13-15`) [z5] |
| `Sidebar/StopDetails.tsx` | 45 [z5] | Presentational; takes `stop: Stop` prop (`:8-10`) [z5] |

### 1.18 `frontend/src/hooks/`, `store/`, `types/`

| File | Lines | Contents |
|---|---|---|
| `hooks/useArrivals.ts` | 50 [z5] | `fetchArrivals(stopId)` (`:30`) → `setArrivals` (`:31`) [z5] |
| `hooks/useHealthCheck.ts` | 38 [z5] | Polls `checkHealth()` on `API_CONFIG.endpoints.health.interval` default 30000 (`:15-24`) [z5] |
| `hooks/useRoutes.ts` | 50 [z5] | `fetchRoutes()` (`:23`) with early-exit cache `routes.length > 0` [z5] |
| `hooks/useStops.ts` | 57 [z5] | `fetchStops('route-101')` — **hardcoded route id** (`:28`) [z5] |
| `hooks/useWebSocket.ts` | 25 [z5] | `wsClient.connect()` / `.disconnect()` on mount/unmount (`:13-21`) [z5] |
| `store/index.ts` | 22 [z5] | Composes 5 slices into `useStore` (`:15`) [z5] |
| `store/slices/busLocations.ts` | 20 [z5] | `buses: Map<string, BusLocation>` (`:16`); `setBuses` (`:17`), `updateBus` (`:18`) [z5] |
| `store/slices/arrivals.ts` | 47 [z5] | `arrivals.byStopId: Map` (`:18-20`); `setArrivals` (`:21-26`), `updateArrival` (`:27-45`) [z5] |
| `store/slices/stops.ts` | 22 [z5] | `stops: Stop[]` (`:17`), `routes: Route[]` (`:18`) [z5] |
| `store/slices/connection.ts` | 42 [z5] | `connection` (`:18-21`), `health` (`:22-26`); `setConnectionStatus`, `updateHeartbeat`, `setHealth` (`:27-40`) [z5] |
| `store/slices/ui.ts` | 149 [z5] | `notifications` (`:12`), `selectedStopId` (`:15`), `followedBusId` (`:17`), route-creator state (`:21-24`), `layerVisibility: { stops, buses }` (`:36`), `hiddenRoutes` (`:37`); 16 actions (`:64-147`) [z5] |
| `types/domain.ts` | 91 [z6] | `Arrival` (`:10-20`), `Stop` (`:22-30`), `Route` (`:32-39`), `BusLocation` (`:41-50`), `HealthStatus` (`:52-56`), `ConnectionState` (`:58-63`), `RouteCreatorStop` (`:67-72`), `CreateRoutePayload` (`:74-82`), `CreateRouteResponse` (`:84-90`) [z6] |

### 1.19 `frontend/src/config/` and `frontend/src/services/`

| File | Lines | Contents |
|---|---|---|
| `config/apiConfig.ts` | 130 [z6] | `VITE_API_BASE_URL` / `VITE_WS_URL` (`:8-9`); 5 endpoint descriptors (`:12-51`); response schemas (`:53-106`); timeouts/retries (`:116-128`) [z6] |
| `config/wsConfig.ts` | 87 [z6] | `VITE_WS_URL` default `ws://localhost:8080/ws` (`:7`); 9 message-type literals (`:14-30`) incl. uppercase `'LOCATION_UPDATE'` (`:20`); 4 payload schemas (`:41-69`); `reconnectInterval: 3000` (`:8`), `heartbeatInterval: 30000` (`:9`) [z6] |
| `services/api/client.ts` | 84 [z6] | Axios instance (`:19-26`); request interceptor (`:29-37`), response interceptor with 400/404/429/500/503 handling (`:40-83`) [z6] |
| `services/api/arrivals.ts` | 30 [z6] | `fetchArrivals(stopId)` → GET with `{stop_id}` (`:10-15`), maps `transformArrival` (`:24`) [z6] |
| `services/api/routes.ts` | 23 [z6] | `fetchRoutes()` → maps `transformRoute` (`:15`) [z6] |
| `services/api/stops.ts` | 32 [z6] | `fetchStops(routeId?)` — optional param (`:10,19`), maps `transformStop` (`:24`) [z6] |
| `services/api/createRoute.ts` | 19 [z6] | `createRoute(payload)` → `POST /api/routes` (`:12`) [z6] |
| `services/api/health.ts` | 25 [z6] | `checkHealth()` → GET `/health` (`:13`) [z6] |
| `services/api/osrm.ts` | 41 [z6] | `OSRM_BASE_URL = 'https://router.project-osrm.org/route/v1/driving'` (`:9`); `getRouteGeometry()` (`:16-38`), falls back to straight lines on error (`:36`) [z6] |
| `services/api/transformers.ts` | 120 [z6] | `transformArrival` (`:10-37`), `transformStop` (`:39-67`), `transformBus` (`:69-94`), `transformRoute` (`:96-119`) [z6] |
| `services/websocket/websocketClient.ts` | 83 [z6] | Singleton `WebSocketClient` (`:11-81`); `connect()` (`:16-59`), `disconnect()` (`:61-67`), `scheduleReconnect()` (`:69-79`); exported instance (`:82`) [z6] |
| `services/websocket/messageHandler.ts` | 132 [z6] | `handleMessage` switch over 6 cases + default (`:20-48`); `handleLocationUpdate` (`:61-92`), `handleArrivalUpdate` (`:94-113`), `handleRouteUpdate` (`:115-119`), `handleHeartbeat` (`:121-123`), `handleError` (`:125-131`) [z6] |
| `services/logger.ts` | 50 [z6] | `Logger` class, level fixed to `'debug'` (`:11`); exported instance (`:49`) [z6] |
| `services/errorHandler.ts` | 56 [z6] | `AppError` interface (`:8`), `handleError` (`:16`), `normalizeError` (`:28`) — contains code literal typo `'UNKNOWN_ObJECT'` (`:49`) [z6] |

### 1.20 `docs/` (prior-audit output — not source)

| File | Lines | Contents |
|---|---|---|
| `docs/audit/ALGORITHM.md` | 31 [z7] | Haversine/ETA/k-ring description; asserts algorithm is orphaned (`:31`) [z7] |
| `docs/audit/ARCHITECTURE.md` | 38 [z7] | Stack + data-flow narrative; 3 "surprises" (`:36-38`) [z7] |
| `docs/audit/ESSENCE.md` | 18 [z7] | Prose characterisation; asserts OSRM stripped out (`:6`) [z7] |
| `docs/audit/OPERATIONS.md` | 37 [z7] | Scripts, Makefile, env vars, test posture [z7] |
| `docs/audit/STATE.md` | 7 [z7] | Audit-tool progress marker; "CONTRADICTIONS OPEN: 0" (`:5`) [z7] |
| `docs/audit/VERIFICATION_LOG.md` | 49 [z7] | Audit run log dated 2026-07-03 (`:4`) [z7] |
| `docs/audit/reports/00-entrypoints.txt` | 1 [z7][zg] | `./backend/cmd/server/main.go` [zg] |
| `docs/audit/reports/00-file-tree.txt` | 114 [z7][zg] | Stale file tree — see §7.7 [zg] |
| `docs/audit/reports/00-language-stats.txt` | 19 [z7][zg] | Per-extension counts [zg] |
| `docs/audit/reports/00-largest-files.txt` | 26 [z7][zg] | Top-25 by line count [zg] |
| `docs/audit/reports/00-orientation.md` | 17 [z7] | Entry-point orientation notes [z7] |
| `docs/audit/reports/01-plan.md` | 75 [z7] | 7-zone audit plan [z7] |
| `docs/audit/reports/02-zone-1..7-*.md` | 44/86/88/57/57/48/39 [z7] | Per-zone prior audit findings [z7] |
| `docs/backend_lld.md` | 221 [z7] | HLD/LLD document, last modified Mar 27 2026 [z7] |
| `docs/component_documentation.md` | 2167 [z7] | 3-part component doc, generated 2025-12-13, last modified Mar 27 2026 [z7] |

### 1.21 Agent/tooling state (non-source)

| File | Contents |
|---|---|
| `.claude/agents/claim-checker.md` | 15 lines; agent definition, model `haiku` [z7] |
| `.claude/agents/file-reader.md` | 20 lines; agent definition, model `haiku` [z7] |
| `.claude/agents/fresh-eyes.md` | 15 lines; agent definition, model `sonnet` [z7] |
| `.claude-flow/neural/patterns.json` | 12370 lines, generated [z7] |
| `.claude-flow/neural/stats.json` | 5 lines, generated [z7] |
| `.claude-flow/policy/state.json` | 4572 lines, generated [z7] |
| `backend/.claude-flow/*` (3 files) | 7582 / 5 / 2092 lines, generated [z7] |
| `frontend/.claude-flow/*` (3 files) | 2395 / 5 / 1132 lines, generated [z7] |

---

## 2. Entry points

### 2.1 Go binary

`backend/cmd/server/main.go` — `main()` at `:44-178` [z1]. Confirmed as the sole Go entry
point by `docs/audit/reports/00-entrypoints.txt:1` [zg].

Execution order as recorded [z1]:

| Step | Line | Action |
|---|---|---|
| 1 | `main.go:48` | `config.LoadConfig()` |
| 2 | `main.go:56` | `database.New(cfg)`; `defer dbPool.Close()` at `:60` |
| 3 | `main.go:64` | `database.RunMigrations(ctx, dbPool, migrations.Files)` |
| 4 | `main.go:70` | `cache.New(cfg)`; `defer redisClient.Close()` at `:74` |
| 5 | `main.go:78-82` | 5 repositories constructed |
| 6 | `main.go:85` | `cache.NewDeviceCache(redisClient)` |
| 7 | `main.go:88-91` | 4 services constructed |
| 8 | `main.go:94-95` | `hub.New()`; **`go wsHub.Run()` — goroutine 1, fire-and-forget** |
| 9 | `main.go:98` | `services.NewIngestService(locRepo, deviceRouteRepo, deviceCache, wsHub)` |
| 10 | `main.go:101-106` | 6 handlers constructed |
| 11 | `main.go:109-116` | Echo instance; middleware order Recovery → CORS → Logging |
| 12 | `main.go:121-153` | Route registration (§3) |
| 13 | `main.go:156-162` | **`go func(){ e.Start(addr) }()` — goroutine 2** |
| 14 | `main.go:165-167` | Block on SIGINT/SIGTERM |
| 15 | `main.go:170-177` | `e.Shutdown(ctx)` with 10s timeout; **hub is never shut down** |

The binary also embeds its own migrations: `backend/migrations/migrations.go:5-6`
`//go:embed *.sql` [z2], applied through the `schema_migrations` ledger at
`backend/internal/database/db.go:56-61,79,96-97` [z2].

### 2.2 Single-page application

`frontend/index.html:16` `<script type="module" src="/src/main.tsx">` → `frontend/src/main.tsx:5-9`
mounts `<App />` into `<div id="root">` (`index.html:15`) inside `<StrictMode>` [z7].
`frontend/src/App.tsx:8-42` is the sole layout; no router and no context providers [z7].

WebSocket connection is opened by `frontend/src/hooks/useWebSocket.ts:13-21` calling
`wsClient.connect()` [z5], which constructs `new WebSocket(WS_CONFIG.url)` at
`frontend/src/services/websocket/websocketClient.ts:23`, default `ws://localhost:8080/ws`
(`frontend/src/config/wsConfig.ts:7`) [z6]. `useWebSocket()` is invoked from
`frontend/src/components/Header/ConnectionStatus.tsx:11` [z5] — i.e. the app's only live
socket is opened as a side effect of rendering the header status widget.

### 2.3 Containers

| Container | Entry |
|---|---|
| `postgres` | `timescale/timescaledb-ha:pg15-latest` (`infra/docker-compose.yml:13`); `../backend/migrations` mounted read-only into `/docker-entrypoint-initdb.d` (`:23`) [z6] |
| `redis` | `redis:7-alpine` (`:38`) with `--maxmemory 256mb --maxmemory-policy allkeys-lru --appendonly yes` (`:40-44`) [z6] |
| `backend` | Built from `../backend/Dockerfile` (`:64-65`); `CMD ["./server"]` (`backend/Dockerfile:79`) [z6][zg]; `depends_on` both services healthy (`:96-100`) [z6] |

### 2.4 Developer/operator entry points

`./deploy.sh` (full stack) [z6]; `make docker-up` (`Makefile:321`), `make dev`
(`Makefile:572`, `go run ./cmd/server`), `make build` (`Makefile:294`), `make run`
(`Makefile:567`) [zg]; `npm run dev` / `npm run build` / `npm run preview`
(`frontend/package.json:7-10`) [z6].

---

## 3. HTTP routes

All registrations are in `backend/cmd/server/main.go` [z1]. Handler → service →
repository → SQL chains are assembled from [z3] (handlers), [z4]/[z4r] (services),
[z2] (repositories and SQL).

### 3.1 `GET /health`

- **Registered:** `main.go:121` — inline closure, no handler type [z1]
- **Service / repository / SQL:** none. Returns `{"status":"healthy","time":"<RFC3339>"}` [z1]

### 3.2 `POST /api/location`

- **Registered:** `main.go:132` → `locHandler.IngestLocation` [z1]
- **Handler:** `handlers/location.go:27-61`; binds `models.Location` (`:28`); validates
  `DeviceID` non-empty (`:34`), latitude ∈ [-90,90] and longitude ∈ [-180,180] (`:37`),
  speed ∈ [0,350] (`:40`); defaults `Timestamp` to now when 0 (`:45-46`). Returns 400
  (`:30,35,38,41`), 500 (`:53`), 200 (`:57`) [z3]
- **Service:** `IngestService.IngestLocation` — `services/ingest.go:29-81` [z4r]
- **Repositories:**
  - `LocationRepository.Insert` — `database/locations.go:47-75` [z2]
  - `DeviceRouteRepository.GetActiveRouteID` — `database/device_routes.go:29-44` [z2]
  - `DeviceCache.SetDeviceState` — `cache/redis.go:184-205` [z3]
- **SQL:**
  ```sql
  -- database/locations.go:60-63
  INSERT INTO location_history (time, device_id, latitude, longitude, speed, accuracy, hex_res9)
  VALUES ($1, $2, $3, $4, $5, $6, $7)
  ```
  ```sql
  -- database/device_routes.go:31-34
  SELECT route_id FROM trips
  WHERE device_id = $1 AND status = 'IN_PROGRESS'
  ORDER BY started_at DESC LIMIT 1
  ```
- **Side effect:** `IngestService` holds `hub *hub.Hub` (`services/ingest.go:21`) [z4r] and
  broadcasts on it; `hub.Hub.Run()` fans out at `hub/hub.go:38-51` [z3]

### 3.3 `GET /api/routes`

- **Registered:** `main.go:135` → `routeHandler.GetRoutes` [z1]
- **Handler:** `handlers/routes.go:24-31`; 500 (`:28`), 200 (`:30`) [z3]
- **Service:** `RoutesService.GetRoutes` — `services/routes.go:19-22` [z4r]
- **Repository:** `RouteRepository.GetAll` — `database/routes.go:19-36` [z2]
- **SQL:** `SELECT id, name, description FROM routes` — `database/routes.go:20` [z2]

### 3.4 `POST /api/routes`

- **Registered:** `main.go:136` → `routeHandler.CreateRoute` [z1]
- **Handler:** `handlers/routes.go:35-65`; binds `models.CreateRouteRequest` (`:36`);
  requires `Name` (`:42`), ≥2 stops (`:45`), valid coords per stop (`:51`); 400
  (`:38,43,46,52`), 500 (`:61`), 201 (`:64`) [z3]
- **Service:** `RoutesService.CreateRoute` — `services/routes.go:24-27` [z4r]
- **Repository:** `RouteRepository.Create` — `database/routes.go:40-90`, single transaction [z2]
- **SQL:**
  ```sql
  -- database/routes.go:49-53
  INSERT INTO routes (id, name, description) VALUES (
      'route-' || substr(md5(random()::text), 1, 8), $1, $2
  ) RETURNING id
  ```
  ```sql
  -- database/routes.go:64-70
  INSERT INTO stops (id, route_id, name, latitude, longitude, sequence_number, geom)
  VALUES ('stop-' || substr(md5(random()::text), 1, 8), $1, $2, $3::numeric, $4::numeric, $5,
          ST_SetSRID(ST_MakePoint($4::numeric, $3::numeric), 4326)) RETURNING id
  ```

### 3.5 `GET /api/stops`

- **Registered:** `main.go:137` → `stopHandler.GetStops` [z1]
- **Handler:** `handlers/stops.go:19-32`; reads `route_id` (`:20`), **hard-requires it** —
  400 at `:22`; 500 (`:28`), 200 (`:31`) [z3]
- **Service:** `StopsService.GetStopsByRoute` — `services/stops.go:19-22` [z4r]
- **Repository:** `StopRepository.GetByRouteID` — `database/stops.go:22-50` [z2]
- **SQL:**
  ```sql
  -- database/stops.go:23-28
  SELECT id, name, latitude, longitude, sequence_number
  FROM stops WHERE route_id = $1 ORDER BY sequence_number
  ```
- **Note:** `StopRepository.GetAll` (`database/stops.go:104-127`) [z2] is not a member of
  the `services.StopRepository` interface (`services/interfaces.go:13-17`) [z4r] and no
  zone report records a caller — see §6.

### 3.6 `GET /api/arrivals`

- **Registered:** `main.go:140` → `arrivalHandler.GetArrivals` [z1]
- **Handler:** `handlers/arrivals.go:22-35`; reads `stop_id` (`:23`), 400 if absent (`:25`);
  500 (`:31`), 200 (`:34`) [z3]
- **Service:** `ArrivalsService.GetArrivalsForStop` — `services/arrivals.go:44-82` [z4r];
  returns `[]ArrivalPrediction` (`services/arrivals.go:33-42`) [z4r]
- **Repositories:**
  - `StopRepository.GetByID` — `database/stops.go:53-65` [z2]
  - `TripRepository.GetActiveTripsBeforeStop` — `database/trips.go:26-64` [z2]
- **In-service computation:** `geo.Haversine` / `geo.CalculateETA`
  (`pkg/geo/distance.go:12-24`, `:26-37`) [z4r]; `isApproaching` uses
  `geoService.GetNeighborHexes(stopLat, stopLng, 3)` at `services/arrivals.go:116` [z4r]
  (recorded as `:117` by [z4] — see §7.8)
- **SQL:**
  ```sql
  -- database/stops.go:54-58
  SELECT id, name, latitude, longitude, sequence_number FROM stops WHERE id = $1
  ```
  ```sql
  -- database/trips.go:30-46
  SELECT t.id, t.device_id, d.name, lh.latitude, lh.longitude, lh.speed
  FROM trips t
  JOIN devices d ON t.device_id = d.id
  LEFT JOIN LATERAL (
      SELECT latitude, longitude, speed FROM location_history
      WHERE device_id = t.device_id ORDER BY time DESC LIMIT 1
  ) lh ON true
  WHERE t.status = 'IN_PROGRESS' AND t.current_stop < $1 AND lh.latitude IS NOT NULL
  ```

### 3.7 `GET /api/nearby/buses`

- **Registered:** `main.go:143` → `nearbyHandler.GetNearbyBuses` [z1]
- **Handler:** `handlers/nearby.go:26-85`; parses `lat` (`:28`), `lng` (`:29`), `radius`
  (`:30`, default 500, cap 10000); 400 (`:33,40,47`), 500 (`:65`), 200 (`:76`) [z3]
- **Service:** `GeofencingService.FindNearbyBuses` — `services/geofencing.go:123-156` [z4r].
  Branch at `geofencing.go:129`: `radiusMeters <= 500` → H3 k-ring path, else direct
  PostGIS. `k := (radiusMeters / 175) + 1` (`:132`), capped `k = 5` (`:134`) [z4r]
- **Repositories:**
  - `LocationRepository.GetBusesInHexes(ctx, hexes, 5)` — `database/locations.go:114-150`,
    called at `services/geofencing.go:144` with `maxAgeMinutes = 5` [z2][z4r]
  - `LocationRepository.GetBusesNearStop(ctx, lat, lng, radiusMeters, 5)` —
    `database/locations.go:154-196`, called at `services/geofencing.go:155` [z2][z4r]
- **SQL:**
  ```sql
  -- database/locations.go:119-132
  SELECT DISTINCT ON (lh.device_id)
      lh.device_id, COALESCE(d.name, lh.device_id) as device_name,
      lh.latitude, lh.longitude, lh.speed, lh.hex_res9
  FROM location_history lh
  LEFT JOIN devices d ON lh.device_id = d.id
  WHERE lh.hex_res9 = ANY($1) AND lh.time > NOW() - ($2 || ' minutes')::INTERVAL
  ORDER BY lh.device_id, lh.time DESC
  ```
  ```sql
  -- database/locations.go:157-178
  SELECT DISTINCT ON (lh.device_id)
      lh.device_id, COALESCE(d.name, lh.device_id) as device_name,
      lh.latitude, lh.longitude, lh.speed,
      ST_Distance(lh.geom::geography,
                  ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography) as distance_meters,
      lh.hex_res9
  FROM location_history lh
  LEFT JOIN devices d ON lh.device_id = d.id
  WHERE ST_DWithin(lh.geom::geography,
                   ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3)
    AND lh.time > NOW() - ($4 || ' minutes')::INTERVAL
  ORDER BY lh.device_id, lh.time DESC
  ```

### 3.8 `GET /api/nearby/stops`

- **Registered:** `main.go:144` → `nearbyHandler.GetNearbyStops` [z1]
- **Handler:** `handlers/nearby.go:89-142`; same three query params (`:91-93`); 400
  (`:96,103,110`), 500 (`:128`), 200 (`:133`) [z3]
- **Service:** `GeofencingService.FindNearbyStops` — `services/geofencing.go:158-163` [z4r]
- **Repository:** `StopRepository.GetNearby` — `database/stops.go:68-101` [z2]
- **SQL:**
  ```sql
  -- database/stops.go:71-83
  SELECT id, name, latitude, longitude, sequence_number
  FROM stops
  WHERE ST_DWithin(geom::geography,
                   ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3)
  ORDER BY ST_Distance(geom::geography,
                       ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography)
  ```

### 3.9 `GET /api/geo/hex`

- **Registered:** `main.go:147` → `nearbyHandler.GetHexInfo` [z1]
- **Handler:** `handlers/nearby.go:146-183`; parses `lat` (`:147`), `lng` (`:148`); 400
  (`:151,158,165`), 200 (`:174`) [z3]
- **Service:** `NearbyService.CalculateHex` / `.GetHexResolution`
  (`handlers/interfaces.go:37-38`) [z3], implemented at `services/geofencing.go:35-44` and
  `:192-195` [z4r]
- **Repository / SQL:** none — pure H3 computation

### 3.10 `GET /ws`

- **Registered:** `main.go:150` → `wsHandler.HandleWS` [z1]
- **Handler:** `handlers/websocket.go:29-45`; upgrader `ReadBufferSize`/`WriteBufferSize`
  1024 and `CheckOrigin` returning true unconditionally (`:13-19`) [z3]
- **Service:** none — the handler holds the concrete `*hub.Hub` (`handlers/websocket.go:21-23`) [z3]
- **Goroutines:** per-client `ReadPump` (`hub/client.go:43-60`) and `WritePump`
  (`hub/client.go:66-93`), both infinite loops with no ctx/done case [z3]

### 3.11 Static files

- `main.go:153` — `e.Static("/", "public")` [z1]. Note `backend/Dockerfile:63` shows the
  `COPY --from=builder /app/public /app/public` line **commented out** [zg].

### 3.12 Outbound HTTP (frontend, not a server route)

- `GET https://router.project-osrm.org/route/v1/driving/{coords}?overview=full&geometries=geojson`
  — `frontend/src/services/api/osrm.ts:9,21`, called from
  `frontend/src/components/Map/RouteCreatorMarkers.tsx:65` [z6][z5]. Falls back to straight
  lines on error (`osrm.ts:36`) [z6]

---

## 4. Database tables

Schema is defined by `backend/migrations/001_create_tables.sql` and altered by
`003_add_h3_postgis.sql` [z2]. Extensions `timescaledb` (`001:2`) and `postgis` (`001:3`) [z2].

### 4.1 `devices`

| Column | Type | Source |
|---|---|---|
| `id` | `TEXT PRIMARY KEY` | `001_create_tables.sql:6-11` [z2] |
| `name` | `TEXT` | same |
| `status` | `TEXT` | same |
| `created_at` | `TIMESTAMP DEFAULT NOW()` | same |

- **Indexes:** primary key only; no zone report records another index on `devices`
- **Triggers:** none recorded
- **Written by:** `004_seed_data.sql:12-18` (5 rows, `ON CONFLICT (id) DO UPDATE`) [z2].
  No Go code writes this table in any zone report.
- **Read by:** `database/locations.go:78,113,158` (`LEFT JOIN devices d`) [z2][z7-BASELINE],
  `database/trips.go:34` (`JOIN devices d`) [z2]. Rows are scanned into `models.NearbyBus`
  and `database.TripWithLocation`, never into `models.Device` [z7-BASELINE:130-132]

### 4.2 `routes`

| Column | Type | Source |
|---|---|---|
| `id` | `TEXT PRIMARY KEY` | `001_create_tables.sql:14-18` [z2] |
| `name` | `TEXT` | same |
| `description` | `TEXT` | same |

- **Indexes:** primary key only
- **Written by:** `database/routes.go:49-53` (`RouteRepository.Create`) [z2];
  `004_seed_data.sql:23-27` (3 rows, `ON CONFLICT`) [z2]
- **Read by:** `database/routes.go:20` (`GetAll`) [z2]; referenced as FK from `stops` and `trips`

### 4.3 `stops`

| Column | Type | Source |
|---|---|---|
| `id` | `TEXT PRIMARY KEY` | `001_create_tables.sql:21-29` [z2] |
| `route_id` | `TEXT REFERENCES routes(id)` | same |
| `name` | `TEXT` | same |
| `latitude` | `DECIMAL(10, 8)` | same |
| `longitude` | `DECIMAL(11, 8)` | same |
| `sequence_number` | `INT` | same |
| `geom` | `GEOMETRY(POINT, 4326)` | same |

- **Indexes:** `idx_stops_geom` GIST on `geom` — created twice, at
  `002_add_indexes.sql:2` and again at `003_add_h3_postgis.sql:62-63`, both `IF NOT EXISTS` [z2]
- **Triggers:** none. `geom` is backfilled by `UPDATE` statements
  (`003_add_h3_postgis.sql:57-59`, `004_seed_data.sql:75-77`) [z2] and set inline on insert
  by `database/routes.go:64-70` [z2]
- **Written by:** `database/routes.go:64-70` [z2]; `004_seed_data.sql:33-42,47-56,61-70`
  (15 rows, `ON CONFLICT`) [z2]
- **Read by:** `database/stops.go:23-28` (`GetByRouteID`), `:54-58` (`GetByID`), `:71-83`
  (`GetNearby`), `:105-109` (`GetAll`) [z2]

### 4.4 `location_history` (TimescaleDB hypertable)

| Column | Type | Source |
|---|---|---|
| `time` | `TIMESTAMP NOT NULL` | `001_create_tables.sql:32-40` [z2] |
| `device_id` | `TEXT REFERENCES devices(id)` | same |
| `latitude` | `DECIMAL(10, 8)` | same |
| `longitude` | `DECIMAL(11, 8)` | same |
| `speed` | `FLOAT` | same |
| `accuracy` | `FLOAT` | same |
| `metadata` | `JSONB` | same |
| `hex_res9` | `TEXT` | added by `003_add_h3_postgis.sql:7` [z2] |
| `geom` | geometry | populated by trigger `003:97-105`; no `ADD COLUMN` for `geom` on this table is recorded in any zone report — see §7.6 |

- **Hypertable:** `create_hypertable('location_history', 'time', if_not_exists => TRUE)` —
  `001_create_tables.sql:43` [z2]
- **Indexes:**
  - `idx_location_history_device_time` on `(device_id, time DESC)` — `002_add_indexes.sql:5`
    and again `003_add_h3_postgis.sql:40-41` [z2]
  - `idx_location_history_hex_res9` on `(hex_res9)` — `003:28-29` [z2]
  - `idx_location_history_geom` GIST on `(geom)` — `003:32-33` [z2]
  - `idx_location_history_hex_time` on `(hex_res9, time DESC)` — `003:36-37` [z2]
- **Triggers:** `trg_location_geom` `BEFORE INSERT FOR EACH ROW EXECUTE FUNCTION
  update_location_geom()` — `003:108-118`; function at `003:97-105` sets
  `NEW.geom := ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)` [z2]
- **Compression:** `timescaledb.compress` with `compress_segmentby = 'device_id'`
  (`003:76-79`); `add_compression_policy('location_history', INTERVAL '7 days')` (`003:87`) [z2]
- **Written by:** `database/locations.go:60-63` (`Insert`) [z2]; `004_seed_data.sql:94-99`,
  `:102-105`, `:108-111` — **11 rows across three statements, none carrying `ON CONFLICT`** [z2]
- **Read by:** `database/locations.go:80-93` (`GetBusesInHex`), `:119-132` (`GetBusesInHexes`),
  `:157-178` (`GetBusesNearStop`), `:200-207` (`GetLatestLocation`) [z2];
  `database/trips.go:30-46` (LATERAL subquery) [z2]

### 4.5 `trips`

| Column | Type | Source |
|---|---|---|
| `id` | `TEXT PRIMARY KEY` | `001_create_tables.sql:46-53` [z2] |
| `route_id` | `TEXT REFERENCES routes(id)` | same |
| `device_id` | `TEXT REFERENCES devices(id)` | same |
| `started_at` | `TIMESTAMP` | same |
| `current_stop` | `INT` | same |
| `status` | `TEXT` | same |

- **Indexes:** primary key only
- **Triggers:** none recorded
- **Written by:** `004_seed_data.sql:82-87` only (4 rows, `ON CONFLICT`) [z2]. No Go code
  writes this table in any zone report — `trips.go` [z2] and `device_routes.go` [z2] contain
  `SELECT` statements only.
- **Read by:** `database/trips.go:30-46` (`GetActiveTripsBeforeStop`) [z2];
  `database/device_routes.go:31-34` (`GetActiveRouteID`) [z2]

### 4.6 `schema_migrations`

| Column | Type | Source |
|---|---|---|
| `version` | `TEXT PRIMARY KEY` | `database/db.go:56-61` [z2] |
| `applied_at` | `TIMESTAMP DEFAULT NOW()` | same |

- Created at runtime by `RunMigrations`, not by any `.sql` file [z2]
- **Read by:** `db.go:79` — `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)` [z2]
- **Written by:** `db.go:96-97` — `INSERT INTO schema_migrations (version) VALUES ($1)` [z2]
- Migration files are discovered via `fs.ReadDir` and applied in alphabetical order
  (`db.go:66-74`) [z2]

### 4.7 Double-application hazard

`infra/docker-compose.yml:23` mounts `../backend/migrations` into
`/docker-entrypoint-initdb.d` [z6], so Postgres executes every `.sql` at first init; the Go
binary then replays all four through its own empty ledger (`database/db.go:57`) [z2][z7].
Because `004_seed_data.sql:94-99,102-105,108-111` carry no `ON CONFLICT` [z2], the 11 seed
`location_history` rows are inserted twice on a fresh volume.

---

## 5. Environment variables

"Consumed" below means: a zone report records a line of code that reads the value **after**
`LoadConfig` returns it.

### 5.1 Backend (Go)

| Variable | Declared | Read at | Actually consumed? |
|---|---|---|---|
| `DB_HOST` | `.env.example:10` [z7], `docker-compose.yml:76` [z6] | `config.go:47` [z1] | **Yes** — `database/db.go` `New()` `:17-51` [z2] |
| `DB_PORT` | `.env.example:11` [z7], `compose:77` [z6] | `config.go:48` [z1] | **Yes** — `db.go:17-51` [z2] |
| `DB_NAME` | `.env.example:12` [z7], `compose:78` [z6] | `config.go:49` [z1] | **Yes** — validated `config.go:74-75` [z1]; used `db.go:17-51` [z2] |
| `DB_USER` | `.env.example:13` [z7], `compose:79` [z6] | `config.go:50` [z1] | **Yes** — validated `config.go:77-78` [z1]; used `db.go:17-51` [z2] |
| `DB_PASSWORD` | `.env.example:14` [z7], `compose:80` [z6] | `config.go:51` [z1] | **Yes** — `db.go:17-51` [z2] |
| `DB_MAX_CONNS` | `.env.example:18` [z7], `compose:81` [z6] | `config.go:52` [z1] | **Yes** — `cfg.DBMaxConns` consumed in `db.go` `New()` [z2] |
| `DB_MIN_CONNS` | `.env.example:19` [z7], `compose:82` [z6] | `config.go:53` [z1] | **Yes** — `cfg.DBMinConns` consumed in `db.go` `New()` [z2] |
| `REDIS_HOST` | `.env.example:24` [z7], `compose:84` [z6] | `config.go:56` [z1] | **Yes** — `cache/redis.go:22` [z3] |
| `REDIS_PORT` | `.env.example:25` [z7], `compose:85` [z6] | `config.go:57` [z1] | **Yes** — `cache/redis.go:22` [z3] |
| `REDIS_PASSWORD` | `.env.example:26` [z7], `compose:86` [z6] | `config.go:58` [z1] | **Yes** — `cache/redis.go:26` [z3] |
| `REDIS_POOL_SIZE` | `.env.example:29` [z7], `compose:87` [z6] | `config.go:59` [z1] | **NO.** [z3] records only `cfg.RedisHost`, `cfg.RedisPort`, `cfg.RedisPassword` being read in `redis.go`; `cfg.RedisPoolSize` has no recorded reader [z1 summary][z3] |
| `SERVER_HOST` | `.env.example:34` [z7], `compose:89` [z6] | `config.go:63` [z1] | **Yes** — `main.go:157` address construction [z1] |
| `SERVER_PORT` | `.env.example:35` [z7], `compose:90` [z6] | `config.go:62` [z1] | **Yes** — `main.go:157` [z1]. Also read by `backend/healthcheck.sh:13` [z6] |
| `H3_RESOLUTION` | `.env.example:45` [z7], `compose:92` [z6] | `config.go:66`, validated 0–15 at `:82-84` [z1] | **NO.** No file records reading `cfg.H3Resolution`. Resolution 9 is hard-coded at `services/geofencing.go:22` [z4r] and `database/locations.go:23` [z2]. `NewGeofencingServiceWithResolution` (`geofencing.go:26-33`) [z4r] and `NewLocationRepositoryWithResolution` (`locations.go:28-33`) [z2] exist and have no recorded caller — `main.go:78,88` uses the non-parameterised constructors [z1] |
| `ENV` | `.env.example:51` [z7], `compose:94` [z6] | `config.go:69` [z1] | **NO.** No zone report records any reader of `cfg.Env` |
| `LOG_LEVEL` | `.env.example:54` (`debug`) [z7], `compose:95` (`info`) [z6] | `config.go:70` [z1] | **NO.** No reader recorded; logging is `log.Printf` at `middleware/logging.go:25`, `middleware/recovery.go:17`, `middleware/errors.go:18` [z1] and `services/ingest.go` imports `log` (`:6`) [z4r] |
| `WS_READ_BUFFER_SIZE` | `.env.example:60` [z7] | **nowhere** — not a `Config` field (`config.go:12-38`) [z1], not in `docker-compose.yml:74-95` [z6] | **NO.** `handlers/websocket.go:13-19` hard-codes 1024 [z3] |
| `WS_WRITE_BUFFER_SIZE` | `.env.example:61` [z7] | **nowhere** [z1][z6] | **NO.** hard-coded 1024 at `handlers/websocket.go:13-19` [z3] |
| `WS_PING_INTERVAL` | `.env.example:64` [z7] | **nowhere** [z1][z6] | **NO.** `pingPeriod` derived from `pongWait` at `hub/client.go:18` [z3] |
| `VERSION` | `.env.example:70` [z7], `compose:67` [z6] | Docker build arg `backend/Dockerfile:13` [zg] | **Build-time only** — `-X main.Version` ldflag (`Dockerfile:28-34`) [zg]; `Makefile:48` derives from `git describe` [zg] |
| `BUILD_TIME` | `.env.example:71` [z7], `compose:68` [z6] | `Dockerfile:14` [zg] | **Build-time only** — `Makefile:49` [zg] |
| `GIT_HASH` | `.env.example:72` [z7], `compose:69` [z6] | `Dockerfile:15` [zg] | **Build-time only** — `Makefile:50` [zg] |

**Summary: five variables are read-then-ignored** — `REDIS_POOL_SIZE`, `H3_RESOLUTION`,
`ENV`, `LOG_LEVEL` (all parsed into `Config` and never consumed) and the three `WS_*`
variables (declared in `.env.example` but never even parsed). `H3_RESOLUTION` is the most
misleading of these: it is *validated* at `config.go:82-84` [z1], which reads as evidence
that it matters.

### 5.2 Container-only

| Variable | Declared | Consumer |
|---|---|---|
| `POSTGRES_USER` | `docker-compose.yml:16` [z6] | Postgres image init |
| `POSTGRES_PASSWORD` | `docker-compose.yml:17` [z6] | Postgres image init |
| `POSTGRES_DB` | `docker-compose.yml:18` [z6] | Postgres image init |

### 5.3 Shell script

| Variable | Read at | Default |
|---|---|---|
| `HEALTH_HOST` | `backend/healthcheck.sh:12` [z6] | `localhost` |
| `SERVER_PORT` | `backend/healthcheck.sh:13` [z6] | `8080` |
| `HEALTH_ENDPOINT` | `backend/healthcheck.sh:14` [z6] | `/health` |
| `HEALTH_TIMEOUT` | `backend/healthcheck.sh:15` [z6] | `5` |

Note: `healthcheck.sh` assigns `HEALTH_HOST` at `:12` but the commands at `:20`, `:23`, `:27`
reference `${HOST}` [z6].

### 5.4 Frontend (Vite)

| Variable | Read at | Fallback |
|---|---|---|
| `VITE_API_BASE_URL` | `frontend/src/config/apiConfig.ts:8` [z6] | recorded as present, value not transcribed |
| `VITE_WS_URL` | `frontend/src/config/apiConfig.ts:9` and `frontend/src/config/wsConfig.ts:7` [z6] | `ws://localhost:8080/ws` (`wsConfig.ts:7`) [z6] |

Neither is declared in `.env.example` [z7] nor in `infra/docker-compose.yml` [z6].

---

## 6. Dead code

Three confidence levels are used, because the zone reports support three different
strengths of claim:

- **CONFIRMED-DEAD** — a zone report records a grep proving zero references.
- **NO-RECORDED-CALLER** — the symbol is defined, and no zone report records any call site.
  This is the honest ceiling for most entries: the zone reports are per-file extractions,
  so absence of a recorded caller is strong but not conclusive.
- **UNREACHABLE-BY-INTERFACE** — the symbol exists on a concrete type but is absent from the
  consumer-declared interface through which that type is injected, so no consumer *can*
  call it through the wiring recorded in `main.go`.

### 6.1 Go — models

| Symbol | Status | Evidence |
|---|---|---|
| `models.Device` (`models/device.go:3-7`) | **CONFIRMED-DEAD** | `grep -rn "models.Device" --include="*.go" backend/` returns no output — `BASELINE.md:123-125` as recorded in [z7] |
| `models.Trip` (`models/trip.go:3-9`) | **CONFIRMED-DEAD** | `grep -rn "models\.Trip\b"` zero hits — `BASELINE.md:183-190`, `IDEAS.md:77-81` as recorded in [z7] |
| `models.ArrivalEvent` (`models/trip.go:11-16`) | **NO-RECORDED-CALLER** | `handlers/arrivals.go` returns whatever `ArrivalService.GetArrivalsForStop` yields, and that interface returns `[]services.ArrivalPrediction` (`handlers/interfaces.go:14`) [z3]. No zone report records `models.ArrivalEvent` being constructed or returned anywhere. See §7.4(g) — `BASELINE.md:185` claims it is live |
| `models.LocationUpdate` (`models/location.go:15-24`) | **NO-RECORDED-CALLER** | Declared [z4r]; the wire type is `hub.Message` (`hub/message.go:7-17`) [z3]. No zone report records a use |

### 6.2 Go — cache

| Symbol | Status | Evidence |
|---|---|---|
| `cache.Session` (`cache/interface.go:18-22`) | **NO-RECORDED-CALLER** | [z3] explicitly lists it under "Unreferenced Declarations" |
| `RedisCache.SetSession` / `.GetSession` / `.DeleteSession` (`redis.go:90-99, 103-120, 123-126`) | **UNREACHABLE-BY-INTERFACE** | `services.DeviceCache` declares only `SetDeviceState` (`services/interfaces.go:41-43`) [z4r] |
| `RedisCache.SetMetric` / `.GetMetric` (`redis.go:131-140, 144-156`) | **UNREACHABLE-BY-INTERFACE** | same |
| `RedisCache.SetDeviceLocation` / `.GetDeviceLocation` / `.DeleteDeviceLocation` (`redis.go:49-58, 62-79, 82-85`) | **UNREACHABLE-BY-INTERFACE** | same. Note these are the purpose-built device-location methods; the *legacy* `DeviceCache.SetDeviceState` (`redis.go:184-205`, commented deprecated at `:172-173`) [z3] is the one actually wired at `main.go:85,98` [z1] |
| `RedisCache.Ping` (`redis.go:161-163`) | **NO-RECORDED-CALLER** | Declared in `CacheStore` (`interface.go:42`) [z3]; no zone report records a call. (`redis.go:21-44` `New()` is recorded as connecting, but no ping call site is cited) |
| `DeviceCache.GetDeviceState` (`redis.go:208-224`) | **UNREACHABLE-BY-INTERFACE** | `services.DeviceCache` declares only `SetDeviceState` [z4r] |
| `SessionTTL`, `MetricTTL`, `NearbyBusesTTL` (`interface.go:55,58,61`) | **NO-RECORDED-CALLER** | [z3] lists all three under unreferenced |
| `KeyPrefixSession`, `KeyPrefixMetric`, `KeyPrefixGeo` (`interface.go:67,68,69`) | **NO-RECORDED-CALLER** | [z3] |
| `CacheStore` interface (`interface.go:26-46`) | **ASSERTION-ONLY** | Its only recorded use is the compile-time assertion `var _ CacheStore = (*RedisCache)(nil)` at `redis.go:227` [z3]. `main.go:40` asserts `services.DeviceCache` instead [z1] |

### 6.3 Go — database

| Symbol | Status | Evidence |
|---|---|---|
| `StopRepository.GetAll` (`database/stops.go:104-127`) | **UNREACHABLE-BY-INTERFACE** | Absent from `services.StopRepository` (`services/interfaces.go:13-17`, which declares only `GetByID`, `GetByRouteID`, `GetNearby`) [z4r]. Confirmed independently by `BASELINE.md:419-423` [z7] |
| `LocationRepository.GetBusesInHex` singular (`locations.go:79-111`) | **UNREACHABLE-BY-INTERFACE** | `services.LocationRepository` declares `GetBusesInHexes` (plural) but not the singular form (`services/interfaces.go:26-31`) [z4r] |
| `LocationRepository.GetNeighborHexes` (`locations.go:227-246`) | **UNREACHABLE-BY-INTERFACE** | Absent from `services.LocationRepository` [z4r]. The live k-ring helper is `GeofencingService.GetNeighborHexes` (`services/geofencing.go:102-121`) [z4r] |
| `NewLocationRepositoryWithResolution` (`locations.go:28-33`) | **NO-RECORDED-CALLER** | `main.go:78` calls `NewLocationRepository` [z1]. Corroborated by `BASELINE.md:373-377` [z7] |

### 6.4 Go — services

| Symbol | Status | Evidence |
|---|---|---|
| `ArrivalsService.GetNearbyArrivals` (`services/arrivals.go:84-111`) | **UNREACHABLE-BY-INTERFACE** | `handlers.ArrivalService` declares only `GetArrivalsForStop` (`handlers/interfaces.go:13-15`) [z3] |
| `ArrivalsService.CalculateETAWithTraffic` (`arrivals.go:133-146`) | **UNREACHABLE-BY-INTERFACE** | same. This is the only place the 1.2× traffic multiplier (`arrivals.go:143`) [z4r] appears |
| `ArrivalsService.DetectArrivalEvent` (`arrivals.go:148-165`) | **UNREACHABLE-BY-INTERFACE** | same |
| `GeofencingService.IsAtStop` (`geofencing.go:56-62`) | **UNREACHABLE-BY-INTERFACE** | `handlers.NearbyService` declares only `FindNearbyBuses`, `FindNearbyStops`, `CalculateHex`, `GetHexResolution` (`handlers/interfaces.go:34-39`) [z3]; `ArrivalsService` holds `*GeofencingService` concretely (`arrivals.go:14`) [z4r] but no zone report records it calling `IsAtStop` |
| `GeofencingService.IsAtStopWithHysteresis` (`geofencing.go:64-100`) | **NO-RECORDED-CALLER** | same; contains the only 1-ring `h3.GridDisk(stopCell, 1)` at `geofencing.go:87` [z4r] |
| `GeofencingService.DetectArrival` (`geofencing.go:165-178`) | **NO-RECORDED-CALLER** | [z4r] |
| `GeofencingService.DetectDeparture` (`geofencing.go:180-190`) | **NO-RECORDED-CALLER** | [z4r] |
| `GeofencingService.HexEdgeLengthMeters` (`geofencing.go:197-223`) | **TEST-ONLY** | Exercised by `geofencing_test.go:65-87` [z4r]; no production call site recorded |
| `NewGeofencingServiceWithResolution` (`geofencing.go:26-33`) | **NO-RECORDED-CALLER** | `main.go:88` calls `NewGeofencingService` [z1]; corroborated by `BASELINE.md:373-377` [z7] |
| `ArrivalPrediction.HexRes9` (`arrivals.go:40`) | **FLAGGED** | [z4r] lists it as assigned but with no consumer visible in that file. Note the frontend *does* map it: `transformers.ts:23` reads `raw[schema.hexRes9]` [z6] |

### 6.5 Frontend — files with no importer

| Symbol | Status | Evidence |
|---|---|---|
| `frontend/src/App.css` | **CONFIRMED-DEAD** | [z7] records a grep confirming no `.ts`/`.tsx` under `frontend/src/` imports it; `App.tsx:6` imports `./styles/globals.css` only [z7] |
| `frontend/src/assets/react.svg` | **CONFIRMED-DEAD** | `grep -rn "react.svg" frontend/src/` returned no matches [z7] |
| `frontend/src/components/Sidebar/StopDetails.tsx` | **NO-RECORDED-CALLER** | [z5] records its imports and props but no component in the 26-file zone-5 set imports it. `Sidebar.tsx:7-9` imports `ArrivalsList` and `RouteCreatorPanel` only [z5]; `StopMarkers.tsx:32` uses its own inline `StopPopupContent` [z5] |
| `frontend/src/services/errorHandler.ts` (`handleError`, `AppError`) | **NO-RECORDED-CALLER** | No file in [z5] or [z6] records importing `errorHandler`; every recorded error path calls `logger.error` directly [z5][z6] |
| `frontend/src/utils/` | **EMPTY DIRECTORY** | [z6]. `h3Helpers.ts` is listed by `docs/audit/reports/00-file-tree.txt` but is not on disk [zg] |

### 6.6 Frontend — symbols with no consumer

| Symbol | Status | Evidence |
|---|---|---|
| `transformBus` (`transformers.ts:69-94`) | **NO-RECORDED-CALLER** | `fetchArrivals`→`transformArrival`, `fetchStops`→`transformStop`, `fetchRoutes`→`transformRoute` are all recorded [z6]; no API module calls `transformBus`. It is also the only reader of `raw[schema.heading]` (`transformers.ts:79`) [z6] |
| `handleArrivalUpdate` (`messageHandler.ts:94-113`) | **DEAD PATH** | Reached only by `case WS_CONFIG.messageTypes.arrival_update` (`messageHandler.ts:29`) [z6]; `hub/message.go:20` defines exactly one message type, `"LOCATION_UPDATE"` [z3]. Corroborated by `IDEAS.md:56-63` (D12) [z7] |
| `handleRouteUpdate` (`messageHandler.ts:115-119`) | **DEAD PATH** | `route_update` is never emitted — `hub/message.go:20` [z3] |
| `handleHeartbeat` (`messageHandler.ts:121-123`) | **DEAD PATH** | `heartbeat` is never emitted [z3] |
| `handleConnected` (`messageHandler.ts:21`) | **DEAD PATH** | `connected` is never emitted [z3] |
| `handleError` (`messageHandler.ts:125-131`) | **DEAD PATH** | `error` type is never emitted [z3] |
| `WS_CONFIG.messageTypes` `connected`/`disconnected`/`heartbeat`/`heartbeat_ack`/`arrival_update`/`route_update`/`system_message`/`error` (`wsConfig.ts:14-17,23,26,29-30`) | **DEAD** | 8 of the 9 declared message types have no backend emitter [z6][z3] |
| `WS_CONFIG.schemas.arrivalUpdate` / `.routeUpdate` / `.heartbeat` (`wsConfig.ts:51-69`) | **DEAD** | Consumed only by the dead handlers above [z6] |
| `WS_CONFIG.heartbeatInterval` (`wsConfig.ts:9`) | **NO-RECORDED-CALLER** | `websocketClient.ts` [z6] records use of `WS_CONFIG.url` (`:23`) and `WS_CONFIG.reconnectInterval` (`:72`) only |
| `BusLocation.heading` (`domain.ts:41-50`) | **PERMANENT GHOST** | Required non-optional field [z6]; `hub/message.go:7-17` has no `heading` [z3] and `message_test.go:51-52` asserts the key is absent [z3]. Corroborated by `IDEAS.md:65-75` (D13) [z7] |
| `ui.notifications` + `addNotification` + `removeNotification` (`ui.ts:12,64-76`) | **NO-RECORDED-CALLER** | No component in the [z5] set reads `notifications` or calls either action |
| `busLocations.setBuses` (`busLocations.ts:17`) | **NO-RECORDED-CALLER** | Only `updateBus` is recorded as being called, from `messageHandler.ts` [z6] |
| `arrivals.updateArrival` (`arrivals.ts:27-45`) | **DEAD PATH** | Called only from `handleArrivalUpdate` (`messageHandler.ts:107`-region) [z6], itself dead |
| `connection.setConnectionStatus` / `.updateHeartbeat` (`connection.ts:27-39`) | **PARTIALLY DEAD** | `updateHeartbeat` is called only from the dead `handleHeartbeat` [z6]; `setConnectionStatus` has recorded use in `websocketClient.ts` lifecycle handlers [z6] |
| `fetchStops()` no-arg branch (`stops.ts:10`, optional `routeId`) | **NO-RECORDED-CALLER** | `useStops.ts:28` always passes the hardcoded `'route-101'` [z5]; the optional-param branch has no caller. Note `GET /api/stops` would 400 on it anyway (`handlers/stops.go:22`) [z3] |

### 6.7 Non-code artifacts that are stale or stray

| Item | Status | Evidence |
|---|---|---|
| `=` (repo root) | **STRAY** | 7 bytes, content `31.3.2`, no extension [z7] |
| `docs/audit/reports/00-file-tree.txt` | **STALE** | Lists `frontend/src/utils/h3Helpers.ts` which is not on disk; omits ≥10 backend files that exist including `device_routes.go`, `handlers/interfaces.go`, `hub/message_test.go`, `services/ingest.go`, `services/interfaces.go`, `services/routes.go`, `services/stops.go` [zg] |
| `backend/Dockerfile:63` | **COMMENTED OUT** | `COPY --from=builder /app/public /app/public` is disabled [zg], while `main.go:153` serves `e.Static("/", "public")` [z1] |

---

## 7. Contradictions between the zone reports and `CLAUDE.md`, `BASELINE.md`, `docs/audit/`

Listed, not edited. Each entry gives the document's claim, the zone-report finding, and the
kind of divergence.

### 7.1 `CLAUDE.md` — line-number citations that no longer resolve

| # | `CLAUDE.md` claim | Zone-report finding |
|---|---|---|
| a | §3.1: H3 resolution 9 hard-coded at `services/geofencing.go:24` and `database/locations.go:22` (`CLAUDE.md:127`) [z7] | `resolution: 9` is at `services/geofencing.go:22` [z4r] and `database/locations.go:23` [z2]. Both citations are off, in opposite directions |
| b | §3.1: "Pool size hard-coded 50 in `cache/redis.go:26`" (`CLAUDE.md:128`) [z7] | [z3] records `cache/redis.go:26` as the line reading `cfg.RedisPassword`. **No zone report records a `PoolSize` literal anywhere in `redis.go`.** The conclusion (`REDIS_POOL_SIZE` unused) is corroborated; the cited line is not |
| c | §4 DEFECT-6: hub started at `go wsHub.Run()`, client goroutines at `handlers/websocket.go:41-42` (`CLAUDE.md:285-289`) [z7] | Hub start is `main.go:95` [z1]; `handlers/websocket.go` is 46 lines with `HandleWS` at `:29-45` [z3], so `:41-42` is plausible but unconfirmed by any zone report |
| d | §4 DEFECT-3: delete `grid` from `store/slices/ui.ts` lines 36, 38, 61 (`CLAUDE.md:259-263`) [z7] | `ui.ts:36` is `layerVisibility: { stops: boolean; buses: boolean }` — **no `grid` key exists** [z5] |
| e | §3.4: k-ring radius 3 in `backend/internal/services/arrivals.go` (`CLAUDE.md:198-201`) [z7] | Confirmed, at `arrivals.go:116` [z4r] / `:117` [z4]. The two zone passes disagree by one line |

### 7.2 `CLAUDE.md` — claims about state that the zone reports show has changed

| # | `CLAUDE.md` claim | Zone-report finding |
|---|---|---|
| f | §4 DEFECT-3 is written as **open**, with instructions to delete the toggle and `h3Helpers.ts` (`CLAUDE.md:253-265`) [z7] | `OperationsPanel.tsx` has exactly two toggles, `stops` (`:40-46`) and `buses` (`:49-56`) — no grid toggle [z5]; `frontend/src/utils/` is an empty directory [z6]. The defect appears already closed while §4 still prescribes the fix |
| g | §2 phase gate: "DEFECT-1 through 7 all closed" is the Phase 1 exit condition (`CLAUDE.md:96` region) [z7] | **DEFECT-7 is still open**: `004_seed_data.sql:94-99`, `:102-105`, `:108-111` have no `ON CONFLICT` [z2] |
| h | §8.1 baseline table: `internal/hub` at 0% coverage (`CLAUDE.md:563-571`) [z7] | `backend/internal/hub/message_test.go` exists — 85 lines, 2 test functions [z3]. The stated baseline is stale |
| i | §3.3 data-flow step 4: "**Handler** pushes update to Redis (hot path) and into the hub channel" (`CLAUDE.md:173-179`) [z7] | The handler delegates: `handlers/location.go:27-61` calls `IngestService.IngestLocation` [z3], and `services/ingest.go:29-81` owns the cache write and hub push [z4r]. The described flow describes the pre-DEFECT-1 shape |
| j | §3.1 attributes the false OSRM claim to `repo-analysis/ESSENCE.md` (`CLAUDE.md:138-140`) [z7] | The file on disk is `docs/audit/ESSENCE.md` [z7]. `repo-analysis/` is not present in any zone report's file set |

### 7.3 `CLAUDE.md` — internal-rule violations the zone reports surface

| # | Rule | Zone-report finding |
|---|---|---|
| k | §7.1: "Interfaces are declared by the consumer"; §3.2: "A service **must** depend on a repository interface, not a concrete type" [z7] | `ArrivalsService.geoService` is concrete `*GeofencingService` (`services/arrivals.go:14`) [z4][z4r]; `IngestService.hub` is concrete `*hub.Hub` (`services/ingest.go:21`) [z4r]; `WebSocketHandler.hub` is concrete `*hub.Hub` (`handlers/websocket.go:21-23`) [z3]. [z4] flags the first of these explicitly as a layering violation |
| l | §3.2: layering is `handlers → services → repositories → database`, and `internal/services` should depend on interfaces it declares [z7] | `backend/internal/services/interfaces.go:6` imports `transit-backend/internal/database` to reference the concrete `database.TripWithLocation` in the `TripRepository` interface signature (`interfaces.go:21`) [z4r]. The services package therefore has a compile-time dependency on the database package |
| m | §7.4: "Every config value is declared in `infra/docker-compose.yml` with a default. Adding a config value without wiring it is a bug" [z7] | `WS_READ_BUFFER_SIZE`, `WS_WRITE_BUFFER_SIZE`, `WS_PING_INTERVAL` are declared in `.env.example:60,61,64` [z7] but appear in neither `docker-compose.yml:74-95` [z6] nor `Config` (`config.go:12-38`) [z1] |
| n | §7.4 names exactly three dead config values (`H3_RESOLUTION`, `LOG_LEVEL`, `REDIS_POOL_SIZE`) [z7] | `ENV` (`config.go:69`) [z1] is a fourth with no recorded reader, and the three `WS_*` vars are a fifth through seventh (§5.1) |
| o | §7.2: "No `heading` — it does not exist anywhere in this codebase" [z7] | Backend-side true (`hub/message.go:7-17`) [z3]. **Frontend-side false**: `domain.ts:41-50` declares `heading: number` as a required field [z6] and `transformers.ts:79` reads `raw[schema.heading]` [z6], so `API_CONFIG.schemas.bus` also names it [z6] |
| p | §3.1 stack table: "Hot state \| Redis \| Latest known location per device" [z7] | The wired path is `DeviceCache.SetDeviceState` (`redis.go:184-205`), which `redis.go:172-173` comments as a deprecated backward-compatibility shim [z3]. The purpose-built `SetDeviceLocation`/`GetDeviceLocation` on `RedisCache` (`redis.go:49-58, 62-79`) are not in the `services.DeviceCache` interface [z4r] and have no recorded caller |
| q | §3.1: "`H3_RESOLUTION` is read and validated in `config.go` but **never consumed** — see DEFECT-7 note below" [z7] | The cross-reference is wrong: DEFECT-7 is the migration double-apply defect (`CLAUDE.md:295-303`) [z7]. The relevant tracked item is D4 (`IDEAS.md:35-39`, `BASELINE.md:371-377`) [z7] |
| r | §10: "Any new Make target must be documented here in the same commit"; §10 lists 6 operations [z7] | `Makefile` declares 60+ `.PHONY` targets and ~70 targets total [zg] |

### 7.4 `BASELINE.md` — present-tense claims the zone reports contradict

`BASELINE.md` is explicitly a Phase-0 snapshot at SHA `c53488d` [z7], so most of these are
staleness rather than error. They are listed because the document states them in the present
tense with no "as of" qualifier at the point of claim.

| # | `BASELINE.md` claim | Zone-report finding |
|---|---|---|
| a | `:244-266` — `grep -rn "database\." backend/internal/handlers/` returns 8 matches across `stops.go`, `location.go`, `arrivals.go`, `routes.go`; five handlers hold concrete repository types [z7] | Every handler now holds an interface: `ArrivalHandler{service ArrivalService}` (`arrivals.go:11-13`), `LocationHandler{service IngestService}` (`location.go:15-17`), `NearbyHandler{geoService NearbyService}` (`nearby.go:15-17`), `RouteHandler{service RoutesService}` (`routes.go:13-15`), `StopHandler{service StopsService}` (`stops.go:11-13`) [z3]. [z3] states no handler imports `database` |
| b | `:268-273` — `main.go:68` constructs `arrivalsService`, `main.go:69` discards it with `_ = arrivalsService`, `main.go:80` builds `ArrivalHandler` from repositories [z7] | `main.go:89` constructs it and `main.go:104` passes it into `NewArrivalHandler(arrivalsService)` [z1] |
| c | `:290-304` — `hub.Message` carries a `Payload` field and `omitempty` on every numeric [z7] | `hub/message.go:7-17` has 9 fields, no `Payload`, and no `omitempty` on any field [z3] |
| d | `:306-308,326-328` — frontend uses lowercase `'location_update'`, so no location message is ever processed [z7] | `wsConfig.ts:20` is `'LOCATION_UPDATE'` uppercase [z6] |
| e | `:339-349` — `layerVisibility` defaults to `{ stops: true, buses: true, grid: false }`; `grid` at `ui.ts:36,38,61` [z7] | `ui.ts:36` is `{ stops: true, buses: true }` [z5] |
| f | `:354-363` — `h3Helpers.ts` exists at `src/utils/h3Helpers.ts:31,52,67` [z7] | `frontend/src/utils/` is empty [z6]; the file is listed only by the stale audit tree [zg] |
| g | `:185` — `models.ArrivalEvent` is "live as `GET /api/arrivals` response type"; `:444` — only `models.Device` and `models.Trip` are dead [z7] | The `/api/arrivals` response type is `services.ArrivalPrediction` (`handlers/interfaces.go:14`, `services/arrivals.go:33-42`) [z3][z4r]. No zone report records any use of `models.ArrivalEvent`, making it a third dead struct |
| h | `:427-435` (D9) — `transformArrival` throws "Missing required fields: id or busId" on every row because `API_CONFIG.schemas.arrival` maps `id, bus_id, eta, status, route_number, route_id, timestamp` [z7] | `transformers.ts:16-24` maps `schema.id`, `schema.tripId`, `schema.deviceId`, `schema.deviceName`, `schema.etaMinutes`, `schema.distanceKm`, `schema.currentSpeed`, `schema.hexRes9`, `schema.isApproaching` — i.e. the `ArrivalPrediction` field set [z6]. D9 appears closed; `BASELINE.md` still asserts it |
| i | `:192-197` — `GET /api/arrivals` registered at `main.go:110`; `GetActiveTripsBeforeStop` called from `handlers/arrivals.go:38` and `services/arrivals.go:54` [z7] | Registered at `main.go:140` [z1]; `handlers/arrivals.go` is 36 lines total and its only recorded call is `h.service.GetArrivalsForStop` [z3] |
| j | `:403-410` — hub started at `main.go:73`, graceful shutdown at `main.go:145` [z7] | `main.go:95` and `main.go:173` [z1]. The defect itself still holds: `hub.go:24-54` has no ctx/done case [z3] |
| k | `:419-423` — `StopRepository.GetAll` at `database/stops.go:100` [z7] | `GetAll` spans `database/stops.go:104-127` [z2] |
| l | `:65-74` — `internal/hub` at 0.0% coverage [z7] | `hub/message_test.go` exists with 2 tests [z3] |
| m | `:8` — toolchain Go 1.26.0 [z7] | Four different Go versions are stated across the repo: `go.mod:3` `go 1.23.0` [z7]; `Makefile:23` `GO_VERSION := 1.23` [zg]; `Makefile:112-127` `check-go` verifies ≥1.21 [zg]; `README.md:9-12` "Go 1.21+" [z7]; `backend/Dockerfile:4` `golang:1.23-alpine` [zg] |

### 7.5 `docs/audit/` — claims the zone reports contradict

| # | Document claim | Zone-report finding |
|---|---|---|
| a | `ESSENCE.md:6` — the system "strips away complex routing engines (like OSRM)" [z7] | `frontend/src/services/api/osrm.ts:9` targets `https://router.project-osrm.org/route/v1/driving` and is imported at `RouteCreatorMarkers.tsx:5`, called at `:65` [z6][z5] |
| b | `ARCHITECTURE.md:37`, `02-zone-6-frontend-ui.md:18`, `VERIFICATION_LOG.md:38` — backend `Message` struct drops `route_id` and `h3_hex` [z7] | `hub/message.go:7-17` includes `RouteID` (`json:"route_id"`) and `H3Hex` (`json:"h3_hex"`) [z3] |
| c | `ARCHITECTURE.md:36`, `ALGORITHM.md:31`, `ESSENCE.md:15`, `02-zone-2-architecture.md:45`, `02-zone-3-algorithm.md:38` — `ArrivalHandler` bypasses `ArrivalsService` and hits repositories directly [z7] | `ArrivalHandler` holds `service ArrivalService` (`handlers/arrivals.go:11-13`) and calls it at `:22-35` [z3]; `main.go:104` injects `arrivalsService` [z1] |
| d | `ARCHITECTURE.md:38`, `02-zone-7-frontend-components.md:34`, `VERIFICATION_LOG.md:42` — the Operations Panel toggles an "H3 Spatial Grid" layer that does not render [z7] | `OperationsPanel.tsx` has two toggles only, `stops` and `buses` (`:40-46,49-56`) [z5] |
| e | `OPERATIONS.md:32` — `H3_RESOLUTION` "sets global geospatial grid precision" [z7] | `cfg.H3Resolution` has no recorded reader; 9 is hard-coded at `geofencing.go:22` [z4r] and `locations.go:23` [z2] |
| f | `OPERATIONS.md:33` — `LOG_LEVEL` "adjusts stdout verbosity" [z7] | `cfg.LogLevel` has no recorded reader [z1] |
| g | `OPERATIONS.md:37` — `distance_test.go`/`geofencing_test.go` "achieve 100% coverage on pure math functions in `pkg/geo`" [z7] | `geofencing_test.go` is in `backend/internal/services/`, not `pkg/geo` [z4][z4r] |
| h | `02-zone-2-architecture.md:4,32-41` — `main.go:24-155`; config at `:28`, DB `:36`, migrations `:44`, Redis `:50`, repos/services `:58-82`, Echo `:85-92`, routes `:97-129`, server `:132-154` [z7] | `main()` spans `:44-178`; config `:48`, DB `:56`, migrations `:64`, Redis `:70`, repos `:78-82`, services `:88-91`, Echo `:109-116`, routes `:121-153`, server `:156-177` [z1]. Every citation in that document is stale |
| i | `02-zone-3-algorithm.md:88` — H3 resolution 9 at `geofencing.go:23`, k-ring 3 at `arrivals.go:118` [z7] | `geofencing.go:22` [z4r] and `arrivals.go:116` [z4r] / `:117` [z4] |
| j | `02-zone-1-operations.md:17` — lists `tailwindcss` and `vite` among frontend deps at `package.json:12-22` [z7] | Both are devDependencies at `package.json:24-39`; the `dependencies` block is `:13-21` [z6] |
| k | `STATE.md:5` — "CONTRADICTIONS OPEN: 0" [z7] | This section lists 30+; [z7] itself flags the claim as self-contradicted by its own sibling documents |
| l | `00-file-tree.txt` — 114 files including `frontend/src/utils/h3Helpers.ts` [z7] | h3Helpers.ts absent; ≥10 existing backend files omitted, incl. `device_routes.go`, `handlers/interfaces.go`, `hub/message_test.go`, `services/ingest.go`, `services/interfaces.go`, `services/routes.go`, `services/stops.go` [zg] |
| m | `00-largest-files.txt` — `main.go` 154 lines, `locations.go` 246, `stops.go` 127, `client.go` 93, `handlers/location.go` 92, `routes.go` 90 [z7][zg] | `main.go` 179 [z1], `locations.go` 247 [z2], `stops.go` 128 [z2], `client.go` 94 [z3], `handlers/location.go` 62 [z3], `database/routes.go` 91 [z2]. The `handlers/location.go` figure is off by 30 lines |
| n | `00-language-stats.txt` — 33 `.go` files [z7][zg] | The zone reports enumerate 35 `.go` files under `backend/` (§1.3–§1.13), before counting `migrations.go` |

### 7.6 `docs/backend_lld.md` and `docs/component_documentation.md`

Both predate the current tree (`backend_lld.md` and `component_documentation.md` last
modified Mar 27 2026; the latter generated 2025-12-13) [z7].

| # | Document claim | Zone-report finding |
|---|---|---|
| a | `backend_lld.md:175-181` — `stops` has column `sequence INT` [z7] | `001_create_tables.sql:21-29` declares `sequence_number INT` [z2] |
| b | `backend_lld.md:160-169` — `location_history` columns list omits `hex_res9` and `geom` [z7] | `003_add_h3_postgis.sql:7` adds `hex_res9`; the `trg_location_geom` trigger (`003:108-118`) populates `geom` [z2] |
| c | `backend_lld.md:62-66` — Redis key `device:{device_id}`, TTL 1 hour [z7] | `cache/interface.go:52` declares `DeviceLocationTTL = 5 * time.Minute` [z3] |
| d | `backend_lld.md:200-202` — build stage `golang:1.21-alpine` [z7] | `backend/Dockerfile:4` is `golang:1.23-alpine` [zg] |
| e | `backend_lld.md:192-197` — `LOG_LEVEL` optional, default `"info"`; `DB_HOST` required with no default [z7] | `.env.example:54` sets `LOG_LEVEL=debug` [z7] while `docker-compose.yml:95` sets `info` [z6]; `config.go:47` reads `DB_HOST` through `getEnvOrDefault` [z1] |
| f | `backend_lld.md:67-70` — broadcast is "Non-blocking: If channel is full, the handler waits (or could drop, but currently blocks)" [z7] | Self-contradictory as written; `hub/hub.go:43-49` is a `default:` branch that silently drops [z3] |
| g | `component_documentation.md:1105-1107` — pool `MaxConns = 10`, `MinConns = 2` [z7] | `docker-compose.yml:81-82` sets `DB_MAX_CONNS: 50`, `DB_MIN_CONNS: 10` [z6]; `.env.example:18-19` the same [z7] |
| h | `component_documentation.md:1144-1160` — `Insert` SQL has 6 columns, no `hex_res9` [z7] | `database/locations.go:60-63` inserts 7 columns including `hex_res9` [z2] |
| i | `component_documentation.md:1729-1741` — `hub.Message` has a `Payload` field and `omitempty` on all numerics [z7] | `hub/message.go:7-17` has neither [z3] |
| j | `component_documentation.md:1042-1048` — `Config` struct omits `H3Resolution` and the pool-size fields [z7] | `config.go:12-38` declares `DBMaxConns` (`:19`), `DBMinConns` (`:20`), `RedisPoolSize` (`:26`), `H3Resolution` (`:33`) [z1] |
| k | `component_documentation.md:1318-1336` — describes a file `internal/cache/device.go` holding `DeviceCache` [z7] | No `internal/cache/device.go` appears in any zone report; `DeviceCache` lives at `cache/redis.go:174-176` [z3] |
| l | `component_documentation.md` — all file links use the prefix `LatitudeX.backend/transit-backend/` [z7] | The tree is `backend/` [z1][z2][z3][z4] |
| m | Neither document mentions `backend/internal/services/ingest.go`, `interfaces.go`, `routes.go`, `stops.go`, `backend/internal/handlers/interfaces.go`, or `backend/internal/database/device_routes.go` [z7] | All six exist [z3][z4r][z2] |

### 7.7 One contradiction with no document on the other side

`main.go:153` registers `e.Static("/", "public")` [z1], but the `COPY` that would place a
`public/` directory into the runtime image is commented out at `backend/Dockerfile:63` [zg].
No zone report records a `public/` directory under `backend/`. No document in
`CLAUDE.md`, `BASELINE.md`, or `docs/audit/` mentions this route at all.

### 7.8 Disagreements *between* zone reports

These affect the confidence of citations above and are recorded so the inventory is not read
as more precise than its inputs.

| Item | `zone-4.md` | `zone-4-recheck.md` |
|---|---|---|
| `distance.go` constants | lines 6, 7, 8 | lines 5, 6, 7 |
| `geofencing.go` `resolution: 9` | line 22 | line 22 (agree) |
| `geofencing.go` `IsAtStop` | `:58-62` | `:56-62` |
| `geofencing.go` `FindNearbyBuses` | `:126-156` | `:123-156` |
| `arrivals.go` k-ring 3 | line 117 | line 116 |
| `arrivals.go` `GetArrivalsForStop` | `:45` | `:44-82` |
| `arrivals.go` `isApproaching` | `:115` | `:113-131` |

| Item | `zone-7.md` | `zone-gap.md` |
|---|---|---|
| `frontend/index.html` | 18 lines | 19 lines |
| `frontend/src/App.css` | 42 lines | 43 lines |
| `frontend/src/index.css` | 12 lines | 13 lines |
| `frontend/src/styles/globals.css` | 245 lines | 246 lines |
| `backend/.dockerignore` | 36 lines | 37 lines |
| `backend/go.sum` distinct modules | 30 | 39 |
| `backend/go.mod` indirect requires | 16 entries listed (`:15-30`) | "25 total", table shows 16 rows |

The `zone-7` / `zone-gap` line counts differ by exactly one on four of five files, which is
consistent with one pass counting a trailing newline and the other not. The `go.sum` and
`go.mod` counts are a genuine disagreement, not an off-by-one.

---

## 8. Third-party dependencies

**Provenance note — this section breaks the document's own rule.** Sections 1–7 above were
assembled exclusively from the nine zone reports, deliberately, per the header note. Pass 2
found that this left a structural gap: the zone reports **do** record individual import
lines file-by-file (e.g. `"github.com/jackc/pgx/v5/pgxpool"` at six separate call sites), but
nothing in Sections 1–7 aggregates them into a dependency inventory — a `grep -c pgx` against
this document returns zero. Reassembling that view from nine per-file zone reports is exactly
the failure mode this document exists to avoid (a second, unrecorded synthesis). The two
manifests are authoritative and short, so this section is built by **reading
`backend/go.mod` and `frontend/package.json` directly**, then citing the zone reports where
they independently confirm an import site, and citing source directly (marked "source,
Pass 3") where no zone report recorded one. This is the one deliberate exception to the
"read nothing but the zone reports" rule stated at the top of this file.

### 8.1 Backend (Go) — direct dependencies (6)

`backend/go.mod:6-11`, verified against `backend/go.mod` directly.

| Package | Version | What it does | Used by |
|---|---|---|---|
| `github.com/gorilla/websocket` | v1.5.3 | WebSocket upgrade (HTTP→WS handshake) and message framing | `handlers/websocket.go:9` [z3:517], `hub/client.go:7` [z3:565] |
| `github.com/jackc/pgx/v5` | v5.4.3 | PostgreSQL driver; `pgxpool.Pool` is the connection-pool type every repository holds | `database/db.go:13` [z2:33], `device_routes.go:8-9` [z2:93-94], `locations.go:9` [z2:148], `routes.go:8` [z2:292], `stops.go:8` [z2:363], `trips.go:6` [z2:452] — all six repository files |
| `github.com/joho/godotenv` | v1.5.1 | Loads a `.env` file into the process environment at startup, if one exists | `config/config.go:8` [z1:200]; call site `godotenv.Load()`, error result discarded [z1:252] |
| `github.com/labstack/echo/v4` | v4.13.4 | HTTP router, middleware chain, request/response helpers — the framework the whole HTTP layer is built on | `cmd/server/main.go:21` [z1:35], middleware package (`cors.go`, `errors.go`, `logging.go`, `recovery.go`) [z1:316-439], every file in `internal/handlers/` [z3:202-518] |
| `github.com/redis/go-redis/v9` | v9.0.5 | Redis client | `cache/redis.go:12` [z3:92] |
| `github.com/uber/h3-go/v4` | v4.4.0 | Go bindings for Uber's H3 hexagonal-grid geospatial indexing library | `database/locations.go:10` [z2:149], `services/geofencing.go:7` [z4-recheck:323,875 / z4:197] |

### 8.2 Backend (Go) — indirect (transitive) dependencies (16)

`backend/go.mod:15-30`, verified directly. **This resolves a disagreement recorded in §7.8**:
`zone-gap.md` headed this table "Indirect Requires (25 total)" but its own table body listed
only 16 rows, matching `zone-7.md`'s count and matching `go.mod` as read directly for this
section — the header figure in `zone-gap.md` was simply wrong. None of these are imported by
any `transit-backend` source file; they exist only to satisfy the six direct dependencies
above, per the "Notes" column zone-gap.md recorded.

| Package | Version | Pulled in by |
|---|---|---|
| `github.com/cespare/xxhash/v2` | v2.3.0 | go-redis [zg] |
| `github.com/dgryski/go-rendezvous` | v0.0.0-20200823014737-9f7001d12a5f | go-redis [zg] |
| `github.com/jackc/pgpassfile` | v1.0.0 | pgx [zg] |
| `github.com/jackc/pgservicefile` | v0.0.0-20221227161230-091c0ba34f0a | pgx [zg] |
| `github.com/jackc/puddle/v2` | v2.2.1 | pgx [zg] |
| `github.com/labstack/gommon` | v0.4.2 | Echo [zg] |
| `github.com/mattn/go-colorable` | v0.1.14 | Echo [zg] |
| `github.com/mattn/go-isatty` | v0.0.20 | Echo [zg] |
| `github.com/valyala/bytebufferpool` | v1.0.0 | Echo [zg] |
| `github.com/valyala/fasttemplate` | v1.2.2 | Echo [zg] |
| `golang.org/x/crypto` | v0.38.0 | transitive (not attributed by any zone report) — source, Pass 3 |
| `golang.org/x/net` | v0.40.0 | transitive — source, Pass 3 |
| `golang.org/x/sync` | v0.14.0 | transitive — source, Pass 3 |
| `golang.org/x/sys` | v0.33.0 | transitive — source, Pass 3 |
| `golang.org/x/text` | v0.25.0 | transitive — source, Pass 3 |
| `golang.org/x/time` | v0.11.0 | transitive — source, Pass 3 |

### 8.3 Frontend — dependencies (9)

`frontend/package.json:12-21` [z6:268-276], verified directly.

| Package | Version | What it does | Used by |
|---|---|---|---|
| `axios` | ^1.13.2 | HTTP client | `services/api/client.ts:6` [z6:402] — the shared Axios instance every `services/api/*.ts` module calls through; also imported directly by `services/api/osrm.ts` — no zone report records this second call site (source, Pass 3) |
| `clsx` | ^2.1.1 | Conditional CSS class-name composition utility | **No zone report records an import, and `grep -rn "clsx" frontend/src/` returns zero matches (source, Pass 3). Declared but unused.** |
| `date-fns` | ^4.1.0 | Date formatting/parsing utility library | **Same as `clsx`** — zero matches in `frontend/src/` (source, Pass 3). Declared but unused. |
| `leaflet` | ^1.9.4 | The underlying map-rendering library (`react-leaflet` wraps it) | `Map/MapContainer.tsx`, `Map/BusMarkers.tsx`, `Map/StopMarkers.tsx`, `Map/RouteCreatorMarkers.tsx` all `import L from 'leaflet'` [z5:86,170,208,275] |
| `lucide-react` | ^0.562.0 | Icon component set | **Same as `clsx`** — zero matches in `frontend/src/` (source, Pass 3). Declared but unused. |
| `react` | ^19.2.0 | UI component/rendering library | `main.tsx:1` — the render root [z7]; used throughout `components/`, `hooks/` |
| `react-dom` | ^19.2.0 | React's browser-DOM renderer | `main.tsx` — `createRoot` [z7] |
| `react-leaflet` | ^5.0.0 | React bindings for `leaflet` (declarative `<MapContainer>`, `<Marker>`, etc.) | All 7 files under `components/Map/` [z5:85,115,146,168,207,243,274] |
| `zustand` | ^5.0.9 | Client-side state store | `store/index.ts:6` [z5:629] and all 5 files under `store/slices/` [z5:646-747] |

Three of the nine declared `dependencies` — `clsx`, `date-fns`, `lucide-react` — have no
import anywhere under `frontend/src/`. This was not previously recorded: §6 (dead code) lists
dead *symbols within* the codebase but does not cover unused *package-level* dependencies.
Not fixed here — see §7 of the Pass 3 report.

### 8.4 Frontend — devDependencies (16)

`frontend/package.json:24-39` [z6:279-294], verified directly. Build/lint/type tooling, not
imported by application code — "used by" here means which config file wires it in.

| Package | Version | What it does | Wired in |
|---|---|---|---|
| `@eslint/js` | ^9.39.1 | ESLint's own recommended rule set | `eslint.config.js:1` [z6:230] |
| `@types/leaflet` | ^1.9.21 | TypeScript types for `leaflet` | ambient, no explicit import |
| `@types/node` | ^24.10.1 | TypeScript types for Node.js APIs | ambient |
| `@types/react` | ^19.2.5 | TypeScript types for `react` | ambient |
| `@types/react-dom` | ^19.2.3 | TypeScript types for `react-dom` | ambient |
| `@vitejs/plugin-react` | ^5.1.1 | Vite's React plugin (JSX transform, Fast Refresh) | `vite.config.ts:7` (per `00-inventory.md` §1.15) |
| `autoprefixer` | ^10.4.23 | Adds vendor-prefixed CSS rules at build time | `postcss.config.js:4` [z6:306] |
| `eslint` | ^9.39.1 | Linter | `package.json:9` script `"lint": "eslint ."` [z6:264] |
| `eslint-plugin-react-hooks` | ^7.0.1 | Lint rules for React hooks correctness | `eslint.config.js:3` [z6:232] |
| `eslint-plugin-react-refresh` | ^0.4.24 | Lint rule enforcing Fast-Refresh-safe exports | `eslint.config.js:4` [z6:233] |
| `globals` | ^16.5.0 | Predefined global-variable sets for ESLint | referenced by `eslint.config.js`, not directly grepped in any zone report (source, Pass 3) |
| `postcss` | ^8.5.6 | CSS transform pipeline that runs `tailwindcss` + `autoprefixer` | `postcss.config.js` |
| `tailwindcss` | ^3.4.19 | Utility-first CSS framework | `postcss.config.js:3` [z6:305], `@tailwind` directives in `styles/globals.css:1-3` [z7] |
| `typescript` | ~5.9.3 | TypeScript compiler | `package.json:8` script `"build": "tsc -b && vite build"` [z6:263]; `tsconfig*.json` |
| `typescript-eslint` | ^8.46.4 | TypeScript-aware ESLint rules | `eslint.config.js:5` [z6:234] |
| `vite` | ^7.2.4 | Dev server and production bundler | `package.json:7,8,10` scripts [z6:262-265]; `vite.config.ts` |

---

*Sections 1–7 were assembled from `docs/explain/raw/zone-{1,2,3,4,4-recheck,5,6,7,gap}.md`
only; no source file was read while writing them. Section 8 is the deliberate exception,
built directly from `backend/go.mod` and `frontend/package.json` — see its provenance note.
Sections 1–6 record what the zone reports state; section 7 records where they diverge from
`CLAUDE.md`, `BASELINE.md`, and `docs/audit/`. None of those three were edited.*
