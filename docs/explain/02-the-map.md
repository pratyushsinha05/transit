# 02-the-map.md — A tour of the directories

This document walks the repository directory by directory. For each one it answers three
questions: what the directory is for, what you will find inside, and why the boundary exists
at all rather than everything living in one folder.

**Source of truth.** Claims here are traceable to `docs/explain/00-inventory.md` (cited as
"inventory §N") or to a `path:line` the inventory records. A handful of facts the inventory
does not state — for example which packages `pkg/geo` imports — were checked directly against
source and are marked *(verified directly)*. Terms in **bold-and-linked** form are defined in
[05-glossary.md](05-glossary.md); this document does not re-explain them.

---

## 1. The top level at a glance

```
transit/
├── backend/      Go module: the server, its packages, its SQL migrations
├── frontend/     React + TypeScript single-page app, built by Vite
├── infra/        Docker Compose file that boots Postgres, Redis, and the backend
├── docs/         Documentation — including stale prior-audit output, see §7
├── Makefile      ~70 targets: docker lifecycle, DB tasks, tests, API smoke checks
├── deploy.sh     Boots the whole stack end to end
├── cleanup.sh    Tears it back down
├── .env.example  22 environment variables with defaults
└── (BASELINE.md, CLAUDE.md, IDEAS.md, README.md — project notes)
```

Inventory §1.1 records the root file set: `Makefile` at 718 lines with `.DEFAULT_GOAL := help`
at `Makefile:679`, `deploy.sh` at 121 lines, `cleanup.sh` at 75, `.env.example` at 72.

**Why four top-level code/docs directories instead of one?** Three different reasons, one per
boundary:

- **`backend/` vs `frontend/` — different languages and different toolchains.** `backend/` is a
  Go module (`module transit-backend`, `backend/go.mod:1`, inventory §1.2) built with `go build`.
  `frontend/` is an npm package built with **[Vite](05-glossary.md)** (`frontend/package.json:7-10`,
  inventory §1.15). They share no build step, no dependency file, and no linter. Putting them in
  one directory would mean one directory with two unrelated dependency manifests, two lockfiles,
  and two sets of config files fighting for the same names.
- **`infra/` vs both — different deploy unit.** `infra/docker-compose.yml` describes how three
  **[containers](05-glossary.md)** are wired together (inventory §1.14). It is not code that
  either the Go module or the npm package compiles; it is the thing that runs them plus the two
  datastores neither of them contains.
- **`docs/` vs everything — prose, not executable.** Nothing under `docs/` is compiled, tested,
  or shipped. See §7 for why part of it should be read with suspicion.

There is a stray file at the root named `=`, seven bytes, whose entire content is the string
`31.3.2` (inventory §1.1, §6.7). It is not referenced by anything. Ignore it.

---

## 2. `backend/` — the Go module

The top of `backend/` holds the module's own build and packaging files, not application code
(inventory §1.2):

- `go.mod` / `go.sum` — the dependency manifest and lockfile. Six **[direct
  dependencies](05-glossary.md)** at `go.mod:6-11` and sixteen indirect ones at `:15-30`
  (inventory §8.1–§8.2). The six are: **[Echo](05-glossary.md)** (HTTP router), **[pgx](05-glossary.md)**
  (PostgreSQL driver), `gorilla/websocket`, `redis/go-redis`, `uber/h3-go` (**[H3](05-glossary.md)**
  bindings), and `joho/godotenv`.
- `Dockerfile` — two stages: a `golang:1.23-alpine` builder (`backend/Dockerfile:4`) and an
  `alpine:latest` runtime image running as a non-root user (`:39,50-51,69`), exposing port 8080
  (`:72`) (inventory §1.2).
- `healthcheck.sh` — 36 lines, tries `wget`, then `curl`, then `nc` against the
  **[health check](05-glossary.md)** endpoint (`healthcheck.sh:20,23,27`, inventory §1.2).

Inside, the module's *conventional* Go layout splits three ways: `cmd/`, `internal/`, and `pkg/`.
That three-way split is a widespread Go layout convention, and two of the three have real meaning
to the compiler. A fourth directory, `backend/migrations/` (§2.4 below), sits alongside them —
it is not part of the `cmd`/`internal`/`pkg` convention at all, but a Go package in its own right,
created solely so its SQL files can be embedded into the compiled binary.

### 2.1 `backend/cmd/server/` — the executable entry point

**What it is for.** In Go, `cmd/` is where the *runnable programs* live: one subdirectory per
binary the module produces, and the subdirectory name becomes the binary name. Everything else
in the module is a library that some `cmd/` program pulls together.

**What you will find.** Exactly one subdirectory, `server/`, containing exactly one file:
`main.go`, 179 lines (inventory §1.3). This project produces one binary. `main.go` is confirmed
as the module's only Go entry point (inventory §2.1).

`main.go` is the **composition root** — the single place that constructs every real object and
hands each one its collaborators, which is what **[dependency injection](05-glossary.md)** means
in practice here. Inventory §2.1 records its execution order: load config (`main.go:48`), open
the database **[connection pool](05-glossary.md)** (`:56`), run **[migrations](05-glossary.md)**
(`:64`), connect **[Redis](05-glossary.md)** (`:70`), build five repositories (`:78-82`), four
services (`:88-91`), start the **[hub](05-glossary.md)** on its own **[goroutine](05-glossary.md)**
(`:94-95`) — then build a fifth service, `IngestService` (`:98`), which is constructed after the
hub rather than alongside the other four because it needs a live hub reference to broadcast into
— build six handlers (`:101-106`), assemble Echo with middleware ordered
Recovery → CORS → Logging (`:109-116`), register routes (`:121-153`), start the server (`:156-162`),
then wait for SIGINT/SIGTERM and shut down (`:165-177`).

It also holds eleven **[interface assertions](05-glossary.md)** at `main.go:31-41` (inventory
§1.3) — one-line compile-time checks that each concrete type still satisfies the interface it is
injected as.

**Why the split exists.** Keeping wiring in `cmd/` and logic in packages means the logic packages
never need to know how the program is assembled. It also means a second binary (a load generator,
a migration tool) would be a new `cmd/` subdirectory, not a new flag on an existing one.

### 2.2 `backend/internal/` — application packages the compiler fences off

**What `internal/` means.** This is not a naming convention that relies on good manners. Go's
compiler enforces it: a package under a directory named `internal/` can be imported **only** by
code inside the tree rooted at that `internal/` directory's parent. Here the parent is the module
root, so only `transit-backend/...` code can import `transit-backend/internal/...`. An outside
module that tried would fail to compile — not a lint warning, a build error.

**Why that matters here.** It gives the maintainer a free hand inside `internal/`. Packages can
be renamed, merged, split, or have their exported types changed without breaking anyone, because
structurally there is no outside caller to break. That is precisely the freedom this project needs:
inventory §6 lists a large amount of code inside `internal/` that has no recorded caller — dead
Redis methods (§6.2), unreachable repository methods (§6.3), unused service methods (§6.4). Cleaning
that up is a local decision, not a breaking change.

![Go import graph](../diagrams/go-import-graph.drawio.png)

**What you will find — eight packages.** They map onto the layering the project intends,
`handlers → services → repositories → database` (see **[clean architecture](05-glossary.md)**):

| Package | Lines / files | What lives here |
|---|---|---|
| `config/` | `config.go`, 106 lines | Reads environment variables into a 16-field `Config` struct (`config.go:12-38,41-87`). Validates `DBName`, `DBUser`, and the H3 **[resolution](05-glossary.md)** range (`:74-84`). Inventory §1.5 |
| `handlers/` | 7 files | The HTTP layer. One file per endpoint group plus `interfaces.go`, which declares the five service interfaces handlers consume (`handlers/interfaces.go:13-39`). Inventory §1.7 |
| `services/` | 7 files | The business-logic layer. `arrivals.go` (166 lines), `geofencing.go` (224), `ingest.go` (82), `routes.go`, `stops.go`, `interfaces.go` declaring six repository interfaces (`services/interfaces.go:13-50`), plus `geofencing_test.go` (148 lines). Inventory §1.11 |
| `database/` | 6 files | The **[repository](05-glossary.md)** layer — SQL only. `db.go` opens the pool and runs migrations; `locations.go` (247 lines) is the largest; plus `routes.go`, `stops.go`, `trips.go`, `device_routes.go`. Inventory §1.6 |
| `cache/` | `interface.go`, `redis.go` | The Redis client behind a `CacheStore` interface with ten methods (`cache/interface.go:26-46`), plus four **[TTL](05-glossary.md)** constants (`:52,55,58,61`). Inventory §1.4 |
| `hub/` | 4 files | The **[WebSocket](05-glossary.md)** **[fan-out](05-glossary.md)** engine. `hub.go` (the `Run()` loop), `client.go` (per-connection read/write pumps), `message.go` (the 9-field wire struct), and `message_test.go`. Inventory §1.8 |
| `models/` | 5 files, 8–36 lines each | Plain domain structs — `Device`, `Location`, `Route`, `Stop`, `Trip` and friends. No behavior. Inventory §1.10 |
| `middleware/` | 4 files, 14–32 lines each | Cross-cutting HTTP concerns: **[CORS](05-glossary.md)**, error-to-JSON conversion, request logging, panic recovery. Inventory §1.9 |

**Two things a newcomer should know before trusting the layer diagram.**

First, the `interfaces.go` files exist because this project's rule is that the *consumer* declares
the interface it needs, not the producer. `handlers/interfaces.go` declares what handlers want from
services; `services/interfaces.go` declares what services want from repositories. See
**[interface](05-glossary.md)**.

Second, that boundary is not fully airtight, though the worst instance of it was already fixed
before this inventory existed. Inventory §7.3.l, citing the zone reports, records
`backend/internal/services/interfaces.go:6` as importing the `database` package to name the
concrete type `database.TripWithLocation` in an interface signature. That claim is stale: commit
`fe918e0` ("move `TripWithLocation` to models, close DEFECT-1 for real") moved the type into
`models` before the zone reports were even written, so `interfaces.go:6` was already importing
`transit-backend/internal/models`, not `database`, at the time the zone report was produced —
verified directly against source for this document. The zone report's claim was wrong when
written, not stale from later drift. The services package does not have that particular
compile-time dependency on `database`. Three other places still hold a
concrete type where the stated rule wants an interface: `ArrivalsService.geoService` is a
`*GeofencingService` (`services/arrivals.go:14`), `IngestService.hub` is a `*hub.Hub`
(`services/ingest.go:21`), and `WebSocketHandler.hub` is a `*hub.Hub` (`handlers/websocket.go:21-23`)
(inventory §7.3.k).

`models/` also contains two structs with zero references anywhere in the Go code: `models.Device`
(`models/device.go:3-7`) and `models.Trip` (`models/trip.go:3-9`), both confirmed dead by grep
(inventory §6.1).

### 2.3 `backend/pkg/` — the deliberate opposite of `internal/`

**What `pkg/` means.** It is the conventional counterpart to `internal/`: code that is safe, and
possibly intended, for other modules to import. The compiler enforces nothing here — `pkg/` is
pure convention — but the convention carries a promise. Code you put in `pkg/` should be stable
enough that outside callers will not be broken, and independent enough that importing it does not
drag in the rest of the application.

**What you will find.** One package, `pkg/geo/`, two files (inventory §1.12):

- `distance.go`, 38 lines — three constants (`EarthRadiusKm = 6371.0`, `MinSpeedKmh = 1.0`,
  `DefaultSpeed = 20.0`), the **[Haversine](05-glossary.md)** **[great-circle distance](05-glossary.md)**
  function (`:12-24`), and `CalculateETA` (`:26-37`).
- `distance_test.go`, 45 lines — table-driven tests for both, including the `DefaultSpeed` fallback
  branch (`:39-41`).

**Why this code and not other code.** `distance.go` imports exactly one package: the standard
library's `math` (`distance.go:3`, *verified directly*). Nothing under `backend/pkg/` imports
`transit-backend/internal` at all (*verified directly*). It performs no I/O, holds no state, and
reads no clock. Give it two latitude/longitude pairs and it returns a number. That is the kind of
code `pkg/` is for, and it is also why this is the only package in the repository with meaningful
test coverage — pure functions need no database, no mocks, and no fixtures to test.

### 2.4 `backend/migrations/` — schema history, compiled into the binary

**What it is for.** Versioned SQL scripts that build the database schema, applied in order and
recorded so none runs twice. See **[migration](05-glossary.md)**.

**What you will find** — four `.sql` files and one `.go` file (inventory §1.13):

- `001_create_tables.sql` (54 lines) — enables the **[TimescaleDB](05-glossary.md)** and
  **[PostGIS](05-glossary.md)** extensions (`:2-3`), creates `devices`, `routes`, `stops`,
  `location_history`, and `trips`, and turns `location_history` into a
  **[hypertable](05-glossary.md)** (`:43`).
- `002_add_indexes.sql` (6 lines) — a **[GIST index](05-glossary.md)** on `stops.geom` and a
  composite index on `location_history`.
- `003_add_h3_postgis.sql` (119 lines) — the biggest one: adds the `hex_res9` column (`:7`), five
  indexes, the `update_location_geom()` trigger function (`:97-105`) that derives PostGIS geometry
  from raw latitude/longitude on every insert, and the **[compression policy](05-glossary.md)** at
  seven days (`:87`).
- `004_seed_data.sql` (140 lines) — demo data: 5 devices, 3 routes, 15 stops, 4 trips, and 11
  `location_history` rows.
- `migrations.go` (7 lines) — the whole reason the SQL sits in a Go package. Its two meaningful
  lines are `//go:embed *.sql` and `var Files embed.FS` (`migrations.go:5-6`). That directive copies
  the `.sql` files into the compiled binary at build time, so the running server carries its own
  schema and does not need the files on disk beside it.

**One hazard worth knowing on day one.** The same `.sql` files are applied twice on a fresh
database. `infra/docker-compose.yml:23` mounts this directory into Postgres's
`/docker-entrypoint-initdb.d`, so Postgres runs every script itself at first init; the Go binary
then replays all four through its own `schema_migrations` ledger, which starts empty
(`backend/internal/database/db.go:57`). Scripts 001–003 are written to tolerate that. The eleven
seed rows in `004_seed_data.sql:94-99,102-105,108-111` carry no `ON CONFLICT` clause, so they are
inserted twice (inventory §4.7).

---

## 3. `infra/` — how the pieces are started together

**What it is for.** Local orchestration. One file: `docker-compose.yml`, 126 lines (inventory §1.14).

**What you will find.** Three services (inventory §2.3):

- `postgres` — image `timescale/timescaledb-ha:pg15-latest` (`:13`), i.e. Postgres 15 with
  TimescaleDB already installed, with the migrations directory mounted read-only (`:23`).
- `redis` — `redis:7-alpine` (`:38`), capped at 256MB with an LRU eviction policy (`:40-44`).
- `backend` — built from `../backend/Dockerfile` (`:64-65`), and configured to wait until both
  other services report healthy before it starts (`:96-100`).

Plus two **[volumes](05-glossary.md)** for data that must survive container restarts
(`postgres_data`, `redis_data`, `:114-118`) and a **[bridge network](05-glossary.md)** named
`transit-network` (`:123-125`) that lets the backend container reach the other two by service name.

**Why a separate directory for one file.** The compose file spans both other trees — it builds
from `../backend/` and mounts `../backend/migrations`. It belongs to neither. It also holds the
default value for most of the backend's environment variables (inventory §5.1), which makes it the
de facto configuration reference; keeping it at a fixed, obvious path is worth more than the one
directory it costs.

---

## 4. `frontend/` — the single-page app

The top of `frontend/` is npm and build configuration (inventory §1.15): `package.json` with nine
runtime dependencies and sixteen dev dependencies, `vite.config.ts` (13 lines) mapping the `@` path
alias to `./src`, three `tsconfig*.json` files, `tailwind.config.js` defining the dark `hud` color
palette (`:9-23`), `postcss.config.js`, and `eslint.config.js`.

Two files here are worth flagging early. `frontend/README.md` is still the unmodified Vite starter
template README and says nothing about this project (inventory §1.15). And `index.html` is under
twenty lines: a favicon, a webfont, a `<div id="root">`, and one module script pointing at
`/src/main.tsx` (`index.html:5,10-12,15,16`, inventory §1.15). Everything else is JavaScript.

`frontend/public/` holds `vite.svg`, the favicon referenced from `index.html:5` (inventory §1.16).
`frontend/dist/` and `frontend/node_modules/` are build output and installed packages; both are
generated and both are gitignored (`frontend/.gitignore`, inventory §1.15).

### 4.1 `frontend/src/` — the organizing convention

The layout groups by *role in the data path*, not by file type. Data comes in through
`services/`, is normalized into `store/`, is read by `hooks/` and `components/`, with `types/` and
`config/` describing the shapes and endpoints involved.

![Frontend module graph](../diagrams/frontend-module-graph.drawio.png)

At the root of `src/` sit `main.tsx` (9 lines — mounts `<App />` into `#root` inside `<StrictMode>`)
and `App.tsx` (42 lines). `App.tsx` is a single static layout that mounts five components: `Sidebar`,
`MapContainer`, `ConnectionStatus`, `OperationsPanel`, `TelemetryTray` (`App.tsx:16,20,25,31,36`).
There is no router and there are no context providers (inventory §2.2). This is not just a
**[single-page application](05-glossary.md)** — it is a single-*screen* one.

### 4.2 `components/` — grouped by area of the screen

Four subdirectories, named after where they appear (inventory §1.17):

- **`Map/`** — seven files, all built on **[Leaflet](05-glossary.md)** through the `react-leaflet`
  bindings. `MapContainer.tsx` sets up the map and mounts the other six (`:36,42-47,50-61`).
  `BusMarkers.tsx` and `StopMarkers.tsx` draw the two marker layers. `RoutePolyline.tsx` draws route
  lines. `RouteCreatorMarkers.tsx` holds the **[OSRM](05-glossary.md)** call that snaps
  operator-drawn waypoints onto roads (`:49-77`). `MapCameraHandler.tsx` and `MapClickHandler.tsx`
  handle follow-camera and click-to-add behavior.
- **`Sidebar/`** — six files: the `Sidebar` shell, `OperationsPanel` (two layer toggles, `:40-46,49-56`),
  the route-creator form, and the arrivals list with its card. `StopDetails.tsx` is also here, but no
  component in the inventory's frontend survey imports it (inventory §6.5).
- **`Header/`** — one file, `ConnectionStatus.tsx`. Note where the live socket actually comes from:
  this widget calls `useWebSocket()` at `:11`, so the app's only WebSocket connection opens as a side
  effect of rendering the header (inventory §2.2).
- **`Footer/`** — one file, `TelemetryTray.tsx`, a clock and status strip.

**Why by area rather than by type.** Everything a screen region needs sits in one folder, so
changing the sidebar means opening one directory. The trade-off is that shared behavior has to be
pulled out deliberately — which is what the next three directories are.

### 4.3 `hooks/` — the data-fetching and subscription layer

Five files, 25–57 lines each (inventory §1.18). Each is a React hook that owns one piece of the
"how do we get this data and keep it fresh" problem, so components can ask for data without
containing fetch logic:

- `useArrivals.ts` — fetches arrivals for a stop and writes them into the store.
- `useRoutes.ts` — fetches routes once, with an early exit when `routes.length > 0`.
- `useStops.ts` — fetches stops. Note the route id is hardcoded to `'route-101'` at `:28`.
- `useHealthCheck.ts` — polls `/health`, default every 30 seconds (`:15-24`).
- `useWebSocket.ts` — opens the socket on mount and closes it on unmount (`:13-21`).

### 4.4 `store/` — the state layer

One **[store](05-glossary.md)**, built with **[Zustand](05-glossary.md)**. `store/index.ts` composes
five **[slices](05-glossary.md)** into a single `useStore` hook (`store/index.ts:15`), and
`store/slices/` holds one file per slice (inventory §1.18): `busLocations.ts` (a `Map` of device id
to location), `arrivals.ts` (arrivals keyed by stop), `stops.ts` (stops and routes), `connection.ts`
(socket and health status), and `ui.ts` (149 lines — selection, follow-mode, route-creator state,
layer visibility, and sixteen actions).

**Why separate from `hooks/`.** Hooks decide *when* data is fetched; slices hold *what* is currently
known. Keeping them apart means a component that only reads state does not accidentally trigger a
fetch, and the WebSocket path — which writes into the store without any hook involved — has
somewhere to write to.

### 4.5 `services/` — the network layer

Everything that talks to something outside the browser (inventory §1.19):

- **`services/api/`** — one module per endpoint, all routed through a single shared
  **[Axios](05-glossary.md)** instance in `client.ts`, whose response interceptor centralizes
  handling for status codes 400/404/429/500/503 (`client.ts:19-26,40-83`). Alongside it:
  `arrivals.ts`, `routes.ts`, `stops.ts`, `createRoute.ts`, `health.ts`, the third-party `osrm.ts`
  (`:9`), and `transformers.ts` (120 lines), which converts raw JSON into the app's own types.
- **`services/websocket/`** — `websocketClient.ts` holds the connection as a **[singleton](05-glossary.md)**
  with reconnect logic (`:16-59,69-79`); `messageHandler.ts` parses each frame and dispatches into
  the store (`:20-48`).
- **`services/logger.ts`** and **`services/errorHandler.ts`** — the former is used; the latter has no
  recorded importer anywhere, and every recorded error path calls `logger.error` directly
  (inventory §6.5).

One thing to expect when reading `messageHandler.ts`: its switch handles six message types, but the
backend defines exactly one, `"LOCATION_UPDATE"` (`backend/internal/hub/message.go:20`). Five of the
six branches are unreachable (inventory §6.6).

### 4.6 `types/` and `config/` — the shared shapes and the endpoint table

- **`types/domain.ts`** (91 lines) — every shared TypeScript type in one file: `Arrival`, `Stop`,
  `Route`, `BusLocation`, `HealthStatus`, `ConnectionState`, and the route-creator payloads
  (inventory §1.18). Worth knowing before you use it: `BusLocation` declares `heading: number` as a
  required field, and the backend's message struct has no such field — the test at
  `backend/internal/hub/message_test.go:51-52` asserts the key is absent (inventory §6.6).
- **`config/apiConfig.ts`** (130 lines) — the base URL, five endpoint descriptors, the expected
  response schemas that `transformers.ts` reads against, and timeout/retry settings.
- **`config/wsConfig.ts`** (87 lines) — the socket URL (default `ws://localhost:8080/ws`, `:7`),
  nine declared message-type literals (`:14-30`), reconnect interval and heartbeat interval.

**Why config is data in a file rather than constants scattered through the code.** The transformers
read field names *out of* these schema objects, so endpoint shape changes are meant to be one edit.
Eight of the nine declared message types have no backend emitter (inventory §6.6), which is easier
to notice with the list in one place than spread across call sites.

### 4.7 `styles/`, `assets/`, and `utils/`

- `styles/globals.css` (245 lines) — the Tailwind directives, ten `--hud-*` custom properties, and
  the visual effects (a radar ping, a scanline sweep, a base64 SVG noise overlay at `:215-224`).
  This is the stylesheet `App.tsx:6` actually imports.
- `App.css` sits beside `App.tsx` but nothing imports it — it is unmodified Vite starter boilerplate,
  confirmed dead by grep (inventory §6.5). Same for `assets/react.svg`.
- `utils/` **is an empty directory** (inventory §1.16). It once held `h3Helpers.ts`, which is why
  some older documents still list that file (inventory §6.5).

---

## 5. `docs/` — and a warning about most of it

`docs/` contains four things, and they do not have equal standing.

**`docs/explain/` — current, and the intended source of truth.** `00-inventory.md` is the
mechanically-assembled repository inventory: it was built exclusively from the nine zone reports in
`docs/explain/raw/`, with no source file opened during writing, so that every claim is traceable
through a citable intermediate (inventory header, lines 3–11). `05-glossary.md` defines the
vocabulary. This file is part of the same set.

**`docs/explain/raw/` — the inventory's inputs.** Nine per-zone extraction reports
(`zone-1.md` … `zone-7.md`, `zone-4-recheck.md`, `zone-gap.md`). You will not normally read these;
they exist so the inventory's citations bottom out somewhere.

**`docs/diagrams/` — generated diagrams.** The `.drawio` sources with their rendered `.png` exports,
plus the `.json` files they were generated from. Two of them are embedded above.

**`docs/audit/` plus `docs/backend_lld.md` and `docs/component_documentation.md` — PRIOR, STALE
AUDIT OUTPUT. Not source of truth. Do not trust without cross-checking.**

This is the part a newcomer most needs warning about, because these files read like authoritative
architecture documentation and are not. Inventory §7.5 and §7.6 catalog contradictions between them
and the actual code. A sample:

- `docs/audit/ESSENCE.md:6` says the system "strips away complex routing engines (like OSRM)."
  OSRM is live: `frontend/src/services/api/osrm.ts:9` targets the public OSRM server and
  `RouteCreatorMarkers.tsx:65` calls it (inventory §7.5.a).
- `docs/audit/ARCHITECTURE.md:36` and `ALGORITHM.md:31` say `ArrivalHandler` bypasses its service
  and hits repositories directly. It does not: `handlers/arrivals.go:11-13` holds a service
  interface (inventory §7.5.c).
- `docs/audit/ARCHITECTURE.md:38` describes an "H3 Spatial Grid" toggle in the operations panel.
  `OperationsPanel.tsx` has two toggles, `stops` and `buses` (inventory §7.5.d).
- `docs/audit/STATE.md:5` states "CONTRADICTIONS OPEN: 0". Inventory §7 lists more than thirty
  (inventory §7.5.k).
- `docs/audit/reports/00-file-tree.txt` lists `frontend/src/utils/h3Helpers.ts`, which is not on
  disk, and omits at least ten backend files that are — including `device_routes.go`,
  `handlers/interfaces.go`, `hub/message_test.go`, and four files under `services/`
  (inventory §6.7, §7.5.l).
- `docs/backend_lld.md:175-181` (221 lines, last modified March 2026) says the `stops` table has a
  column named `sequence`. The migration declares `sequence_number` (inventory §7.6.a).
- `docs/component_documentation.md:1318-1336` (2167 lines, generated December 2025) documents a file
  `internal/cache/device.go`. No such file exists; `DeviceCache` lives at `cache/redis.go:174-176`
  (inventory §7.6.k). All of that document's file links use a path prefix,
  `LatitudeX.backend/transit-backend/`, that does not match this tree (inventory §7.6.l).

These are not nitpicks about line numbers. They are wrong about which code runs, which columns
exist, and which files are present. If you read something in `docs/audit/`, `backend_lld.md`, or
`component_documentation.md`, verify it against the code or against `00-inventory.md` before acting
on it.

---

## 6. Directories not covered by this tour

- `.claude/` — agent definitions used while producing these docs (inventory §1.21).
- `.claude-flow/` and the copies of it inside `backend/`, `frontend/`, and elsewhere — generated
  tooling state, thousands of lines of JSON, not project code (inventory §1.21).
- `frontend/dist/`, `frontend/node_modules/` — build output and installed packages.
- `.git/` — version control.

---

## DEVIATIONS

1. **Facts verified directly against source rather than through the inventory.** Two claims in §2.3
   are not stated by `00-inventory.md` and were checked against the tree for this document: that
   `backend/pkg/geo/distance.go:3` imports only the standard library's `math`, and that nothing
   under `backend/pkg/` imports `transit-backend/internal`. Likewise, §2.1's claim that `backend/cmd/`
   has exactly one subdirectory comes from a directory listing; the inventory establishes only that
   `main.go` is the sole Go entry point (§2.1).
2. **A root file exists that the inventory's root census omits.** `UNDERSTANDING_PASS.md` is present
   at the repository root but does not appear in inventory §1.1. It is not described in this tour,
   because no verified record of its contents exists in the source of truth.
3. **Both requested diagram images were found and embedded** at
   `docs/diagrams/go-import-graph.drawio.png` and `docs/diagrams/frontend-module-graph.drawio.png`.
   No substitution or omission was needed.
