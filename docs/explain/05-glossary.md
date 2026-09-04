# 05-glossary.md — Every term, plain English

Alphabetical. Each entry: what it means, then a concrete example from this codebase where
one helps. Most `file:line` citations reproduce ones `docs/explain/00-inventory.md` already
gives for the same fact; entries phrased "verified directly against source for this document"
are the exception — those specific claims were checked directly against the source file
because the inventory did not already cite a line for them. The absence of that phrase does
not mean a citation is unverified, only that it matches what the inventory already records.
*See also* lines cross-link related entries.

---

**Along-route distance** — see **ETA**. Not built yet; the ETA this system computes today is
straight-line, not along-route. Mentioned here only to head off the assumption that it exists.

**Axios** — a JavaScript/TypeScript library for making HTTP requests from the browser. This
codebase wraps a single shared Axios instance in `services/api/client.ts`, with request and
response interceptors that centralize error handling for status codes 400/404/429/500/503
(inventory §1.19, `client.ts:19-26,40-83`). Every other `services/api/*.ts` file calls through
that one instance rather than making raw `fetch()` calls. *See also:* **SPA**.

**Bridge network** — a private virtual network Docker creates so containers can reach each
other by service name instead of by IP address. This project's `transit-network` bridge
(`infra/docker-compose.yml:123-125`, inventory §1.14) is what lets the `backend` container
resolve `postgres` and `redis` as hostnames. *See also:* **Docker Compose**, **Container**.

**Broadcast** — sending one message to every currently-connected client at once, instead of
one at a time. This system's `Hub` has a `Broadcast` channel, buffered to 256 messages
(`hub/hub.go:17`, inventory §1.8); every location update goes in one end and comes out the
other, fanned out to every WebSocket client. *See also:* **Hub**, **Fan-out**, **Channel**.

**Cache** — a store that keeps a copy of data somewhere faster to read than its source of
truth, at the cost of that copy potentially going stale. This project's cache is **Redis**: it
holds each device's latest known location so a "where is this device right now" read doesn't
have to query **TimescaleDB** (`cache/redis.go`, inventory §1.4, §3.2). *See also:* **Redis**,
**Hot path**, **TTL**.

**Channel** — a Go primitive for passing values safely between goroutines without manual
locking. The `Hub` has three: `Broadcast` (buffered 256), and `Register`/`Unregister`
(unbuffered) (`hub/hub.go:17-19`, inventory §1.8). A goroutine writes to a channel; another
goroutine, running concurrently, reads from it. *See also:* **Goroutine**, **Hub**.

**Clean architecture** — a design where each layer depends only on the layer directly below
it, through an interface, never on a concrete type two layers down. This project's stated
layering is `handlers → services → repositories → database` (CLAUDE.md §3.2). Real violations
still exist — three concrete types sit where an interface belongs: `ArrivalsService.geoService`
(`services/arrivals.go:14`), `IngestService.hub` (`services/ingest.go:21`), and
`WebSocketHandler.hub` (`handlers/websocket.go:21-23`) (inventory §7.3.k). Inventory §7.3.l
records a fourth — `services/interfaces.go:6` importing `database` — but that one is stale: a
commit predating the zone reports (`fe918e0`) had already moved the type it referenced into
`models`, so that particular violation was already fixed by the time the zone report was
written (verified directly against source for this document). "Clean architecture" here
describes the intent, not an unbroken guarantee. *See also:* **Dependency injection**,
**Interface**, **Repository**.

**Compression policy** — a TimescaleDB feature that automatically rewrites old rows in a
hypertable into a compressed, columnar format to save disk space, once they pass an age
threshold. `location_history` compresses on `device_id` after 7 days
(`003_add_h3_postgis.sql:76-79,87`, inventory §4.4). *See also:* **Hypertable**,
**TimescaleDB**.

**Connection pool** — a set of already-open database connections that the application reuses
instead of opening a new one per request (opening a connection is comparatively slow).
`database/db.go`'s `New()` configures a `pgxpool.Pool` with `MaxConns`/`MinConns` from config,
a 1-hour connection lifetime, and a 30-minute idle timeout (`db.go:31-35`, verified directly
against source for this document). *See also:* **pgx**.

**Container** — either (a) a running instance of a Docker image — isolated, with its own
filesystem and process space, but sharing the host kernel — or (b) an architectural pattern
(Inversion of Control container), not used by name in this codebase. This glossary means (a)
throughout. This project runs three: `postgres`, `redis`, `backend` (`infra/docker-compose.yml`,
inventory §2.3). *See also:* **Docker**, **Docker Compose**, **Volume**.

**Context (`context.Context`)** — Go's standard mechanism for carrying a deadline,
cancellation signal, or request-scoped value through a call chain. `database.New()` uses a
10-second timeout context when first connecting to Postgres (`db.go:37-38`, verified directly
for this document). CLAUDE.md's convention (§7.1) is that every I/O-touching function takes
one as its first parameter — the inventory's zone reports don't audit this convention
directly, so treat it as the stated rule rather than a verified fact.

**CORS (Cross-Origin Resource Sharing)** — a browser security rule that blocks a web page from
calling an API on a different origin (domain/port) unless the API explicitly allows it. This
backend's CORS middleware allows every origin: `AllowOrigins: ["*"]`
(`middleware/cors.go:8-14`, inventory §1.9) — fine for a local demo, a real hazard in
production. *See also:* **Handler**.

**Dependency injection** — passing an object's collaborators in from outside (usually through
its constructor) rather than having it construct them itself. `backend/cmd/server/main.go` is
where this happens for the whole backend: it builds every repository, service, and handler
and wires them together (inventory §2.1, steps 5-10). *See also:* **Interface**, **Clean
architecture**.

**Device** — the generic term this codebase uses for one tracked thing sending GPS pings (a
bus, in the demo skin, but nothing backend-side is bus-specific). Two separate things share
this name and should not be conflated: the `devices` **table**, which is real and queried on
every ping (`database/locations.go:78,113,158`, `database/trips.go:34`, inventory §4.1), and
the Go struct `models.Device` (`models/device.go:3-7`, inventory §1.10), which is
**CONFIRMED-DEAD** — every query against the table scans its rows into `models.NearbyBus` or
`database.TripWithLocation` instead, and nothing in the Go code ever references
`models.Device` (inventory §6.1). *See also:* **Telemetry**, **Ping**.

**Direct dependency / indirect (transitive) dependency** — a *direct* dependency is a library
your code imports by name; an *indirect* one is a library that a direct dependency needs, and
that your build tool pulls in automatically. `backend/go.mod` lists 6 direct requires (e.g.
`github.com/labstack/echo/v4`) and 16 indirect ones that only exist to satisfy those 6
(`go.mod:6-11,15-30`, inventory §8.1-8.2). *See also:* **pgx**, **Echo**.

**Docker** — a tool for packaging an application and everything it needs to run (runtime,
libraries, files) into a single portable image, then running that image as an isolated
**container**. This backend's `Dockerfile` builds in two stages — a `golang:1.23-alpine`
builder, then a slim `alpine:latest` runtime image (`backend/Dockerfile:4,39`, inventory
§1.2). *See also:* **Container**, **Docker Compose**, **Volume**.

**Docker Compose** — a tool for defining and running several Docker containers together as one
unit, described in a single YAML file instead of separate manual `docker run` commands. This
project's `infra/docker-compose.yml` defines three services — `postgres`, `redis`, `backend`
— on a shared **bridge network**, with `backend` waiting on the other two to pass their health
checks first (`docker-compose.yml:96-100`, inventory §1.14, §2.3). *See also:* **Container**,
**Bridge network**, **Volume**.

**Echo** — the Go HTTP web framework this backend is built on: it provides the router,
middleware chaining, and request/response helpers. Every file under `internal/handlers/`
registers its routes through it, and `cmd/server/main.go` builds the Echo instance with
middleware ordered Recovery → CORS → Logging (`main.go:109-116`, inventory §2.1 step 11).
*See also:* **Handler**, **Middleware** (not a separate entry — see **CORS** for an example).

**ETA (Estimated Time of Arrival)** — how long until a device reaches a target point. This
system computes it as `distance / speed`, where distance is straight-line **Haversine**
distance and speed falls back to a hard-coded `DefaultSpeed = 20.0` km/h whenever the
device's reported speed is below 1.0 km/h (`pkg/geo/distance.go:26-37`, inventory §1.12).
It is *not* along-route distance — see **Along-route distance**. *See also:* **Great-circle
distance**, **Telemetry**.

**Fan-out** — the pattern of one input producing many parallel outputs. The `Hub`'s `Run()`
loop reads one message off the `Broadcast` channel and writes it to every registered client's
own channel — one message in, N sends out (`hub/hub.go:24-54`, inventory §1.8). *See also:*
**Broadcast**, **Hub**.

**Geofence** — a virtual boundary around a real-world location; "inside" or "near" it is
computed, not physically sensed. This system approximates geofences with H3 hex cells: a
device is "approaching" a stop if its hex is within a k-ring of radius 3 (~500m) around the
stop's hex (`services/arrivals.go:113-131`, inventory §3.6). *See also:* **H3**, **k-ring**,
**Resolution**.

**GIST index** — a PostgreSQL index type (Generalized Search Tree) built for data that isn't
simple scalars — geometric shapes, ranges, full-text vectors. Both `stops.geom` and
`location_history.geom` have GIST indexes so PostGIS spatial queries don't have to scan every
row (`002_add_indexes.sql:2`, `003_add_h3_postgis.sql:32-33,62-63`, inventory §4.3-4.4).
*See also:* **PostGIS**, **Spatial index**.

**Goroutine** — a lightweight, independently-scheduled function execution in Go — cheaper
than an OS thread, and Go's runtime multiplexes many goroutines onto few real threads. This
backend starts at least two long-lived ones outside `main()`'s own execution: the WebSocket
hub's `Run()` loop and the Echo server's `Start()` call, both launched with `go` at
`main.go:95,156-162` (inventory §2.1, steps 8 and 13). Every connected WebSocket client adds
two more (`ReadPump`, `WritePump`). *See also:* **Channel**, **Hub**.

**Great-circle distance** — the shortest distance between two points on the surface of a
sphere (as opposed to a straight line through the earth, which isn't walkable). See
**Haversine** for how this codebase computes it.

**H3** — Uber's hexagonal hierarchical geospatial indexing system: it divides the earth's
surface into hexagonal cells at multiple zoom levels ("resolutions") and gives each cell a
unique ID. This backend uses `github.com/uber/h3-go/v4`, Uber's Go bindings, hard-coded to
resolution 9 in two places (`services/geofencing.go:22`, `database/locations.go:23`, inventory
§8.1, §3.1). *See also:* **Hexagonal grid**, **Resolution**, **k-ring**.

**Handler** — in this codebase's layering, the HTTP-facing layer: it parses the request,
calls a service, and writes the response — it should hold no business logic of its own.
`handlers/location.go`'s `IngestLocation` validates the incoming JSON body, then delegates
everything else to `IngestService` (`location.go:27-61`, inventory §3.2). *See also:*
**Service**, **Clean architecture**.

**Haversine** — a formula for computing **great-circle distance** between two latitude/
longitude points, assuming the earth is a perfect sphere. This project's implementation lives
at `pkg/geo/distance.go:12-24` (inventory §3.4) and is the distance term in every **ETA**
calculation. *See also:* **Great-circle distance**, **ETA**.

**Health check** — an endpoint or command that reports whether a service is alive and ready.
This backend exposes `GET /health` (inventory §3.1); Docker Compose calls it (via
`healthcheck.sh`, inventory §1.2) to decide when the `backend` container is ready, and the
`backend` service's own `depends_on` waits on Postgres and Redis passing *their* health checks
first (`infra/docker-compose.yml:96-100`, inventory §2.3).

**Hexagonal grid** — a tiling of a surface using hexagons instead of squares; hexagons have
the useful property that every neighboring cell is the same distance from the center cell
(a square grid's diagonal neighbors are farther than its edge neighbors). This is what **H3**
overlays on the earth. *See also:* **H3**, **k-ring**.

**Hot path** — the fast, low-latency route data takes for read access, as opposed to the
durable but slower path. This system's hot path is Redis: `IngestService` writes each
device's latest location to Redis via `DeviceCache.SetDeviceState`
(`cache/redis.go:184-205`, inventory §3.2) in addition to the durable Postgres write, so a
"where is this device right now" read doesn't have to query TimescaleDB. *See also:* **Redis**,
**TTL**.

**Hub** — the component that owns the set of connected WebSocket clients and fans broadcast
messages out to all of them. This system's `Hub` struct (`hub/hub.go:7-13`, inventory §1.8)
is a **singleton**, constructed once in `main()` and run on its own **goroutine** for the
life of the process — with no shutdown path, a documented open defect (CLAUDE.md DEFECT-6;
inventory §7.4.j confirms the hub start line but not that the gap is closed). *See also:*
**Broadcast**, **Fan-out**, **WebSocket**.

**Hypertable** — a TimescaleDB table that looks like a normal Postgres table to application
code but is internally partitioned into time-based chunks, which is what makes time-range
queries and the **compression policy** fast at scale. `location_history` is this project's
one hypertable (`create_hypertable(...)`, `001_create_tables.sql:43`, inventory §4.4).
*See also:* **TimescaleDB**, **Time-series**.

**Interface** — in Go, a set of method signatures with no implementation; any type that
implements all those methods satisfies the interface automatically, with no explicit
"implements" declaration. This project's convention is that the *consumer* package declares
the interface it needs (CLAUDE.md §7.1) — e.g. `handlers/interfaces.go` declares
`IngestService` with the one method handlers actually call (inventory §1.7). *See also:*
**Interface assertion**, **Dependency injection**.

**Interface assertion** — a Go idiom, `var _ SomeInterface = (*ConcreteType)(nil)`, that
forces a compile error if `ConcreteType` stops satisfying `SomeInterface` — a compile-time
guarantee with no runtime cost. `cmd/server/main.go` has 11 of these
(`main.go:31-41`, inventory §1.3). *See also:* **Interface**.

**JSONB** — PostgreSQL's binary JSON column type: it stores arbitrary JSON but, unlike plain
`JSON`, is indexable and faster to query. `location_history.metadata` is declared `JSONB`
(`001_create_tables.sql:32-40`, inventory §4.4), though no zone report records anything
writing a non-empty value into it.

**k-ring** — the set of hexagonal cells within N "rings" of a center cell in an **H3** grid —
k-ring 0 is just the center cell, k-ring 1 adds its 6 immediate neighbors, and so on. This
system's arrival-approach check uses k-ring radius 3 (~500m at resolution 9) around a stop's
hex (`services/arrivals.go:113-131`, inventory §3.6). *See also:* **H3**, **Resolution**,
**Geofence**.

**Leaflet** — the JavaScript library that actually draws the interactive map (tiles, markers,
polylines, pan/zoom). This project uses it through `react-leaflet` bindings in every file
under `frontend/src/components/Map/` (inventory §1.17); `MapContainer.tsx` mounts
`<LeafletMap zoom={13}>` with CartoDB Dark Matter tiles (`MapContainer.tsx:36,42-47`).
*See also:* **Tile server**, **SPA**.

**Migration** — a versioned SQL script that changes the database schema, applied in order and
tracked so it never runs twice on a given database. This backend embeds four
(`backend/migrations/001..004*.sql`, inventory §1.13) and tracks which have run in a
`schema_migrations` table it creates at startup (`database/db.go:56-61`, inventory §4.6).
A double-application hazard exists because Docker Compose also mounts the same SQL files into
Postgres's own init path (inventory §4.7) — see `03-data-flow.md` and `04-components.md` for
how that plays out. *See also:* **Connection pool**.

**`omitempty`** — a Go JSON struct-tag option that drops a field from the serialized output
entirely when its value is the zero value (`0`, `""`, `false`, etc). This project's
`hub.Message` deliberately has **no** `omitempty` on any numeric field
(`hub/message.go:7-17`, inventory §1.8) — a stopped device with `speed: 0` must serialize the
key, not omit it, or the frontend can't distinguish "stopped" from "not reported."

**OSRM (Open Source Routing Machine)** — an open-source routing engine that turns a list of
waypoints into a road-following path. This project calls the public OSRM demo server
(`https://router.project-osrm.org`) from the browser, only on the route-creator screen, to
snap operator-drawn waypoints onto roads — falling back to straight lines if the call fails
(`services/api/osrm.ts:9,16-38`, inventory §3.12).

**pgx** — `github.com/jackc/pgx/v5`, this backend's PostgreSQL driver — the Go library that
actually speaks the Postgres wire protocol. Every repository holds a `*pgxpool.Pool` (pgx's
pooled-connection type) rather than a single connection (`database/db.go:13,17`, inventory
§8.1). Verified directly against `backend/go.mod:7` and `database/db.go` for this document.
*See also:* **Connection pool**.

**Ping** — this word means two unrelated things in this codebase, easy to conflate. (1) A GPS
location report from a tracked device — the subject of `POST /api/location` and the whole
`03-data-flow.md` document. (2) A WebSocket keep-alive frame — `hub/client.go`'s
`pingPeriod`/`pongWait` constants (`client.go:15,18`, inventory §1.8) that detect a dead
connection, unrelated to GPS data. *See also:* **Telemetry**, **WebSocket**.

**PostGIS** — a PostgreSQL extension that adds geometric/geographic data types and spatial
functions (distance, containment, nearest-neighbor) directly into SQL. Enabled by
`001_create_tables.sql:3` (inventory §4). `stops.geom` and `location_history.geom` are both
`GEOMETRY(POINT, 4326)` columns it manages — 4326 is the SRID (Spatial Reference ID, PostGIS's
numeric label for a coordinate system) for plain latitude/longitude, the system called WGS 84
(the same one GPS itself uses). *See also:* **ST_DWithin**, **Spatial index**, **GIST index**.

**React** — the JavaScript UI library the whole frontend is built with; components declare
what the UI should look like for a given state, and React handles updating the actual DOM.
`main.tsx` mounts the single root component, `<App />`, inside `<StrictMode>`
(`main.tsx:5-9`, inventory §2.2). *See also:* **SPA**, **Zustand**.

**Redis** — an in-memory key-value store, used here as the **hot path** cache for each
device's latest known location (`cache/redis.go`, inventory §1.4). Configured with a 256MB
memory cap and an LRU (Least Recently Used) eviction policy in Docker Compose — once the cap
is hit, Redis discards whichever key has gone longest without being read, to make room for new
writes (`infra/docker-compose.yml:40-44`, inventory §2.3). *See also:* **Hot path**, **TTL**,
**Connection pool**.

**Repository** — in this codebase's layering, the data-access layer: it holds the SQL and
returns Go structs, with no business logic. `LocationRepository.Insert`
(`database/locations.go:47-75`, inventory §1.6) is one example — it does exactly one thing,
insert a row, and nothing decides *whether* to insert. *See also:* **Handler**, **Service**,
**Clean architecture**.

**Resolution** — in **H3**, how fine-grained the hexagonal grid is; resolution 0 covers
continents in a handful of cells, resolution 15 covers roughly a square meter. This project
hard-codes resolution 9 (~175m hex edge) in two places, despite an `H3_RESOLUTION` environment
variable that is parsed, validated, and never actually read (`config.go:66,82-84`, inventory
§5.1) — a documented dead-config gap. *See also:* **H3**, **k-ring**.

**Semantic versioning (semver)** — a version-numbering convention, `MAJOR.MINOR.PATCH`, where
each position signals a different kind of change. `frontend/package.json`'s `^1.13.2` for
`axios` means "any version compatible with 1.13.2, up to but not including 2.0.0"; `~5.9.3`
for `typescript` is narrower — patch-level updates only (inventory §8.3-8.4).

**Service** — in this codebase's layering, the business-logic layer, between handlers and
repositories: it decides *what* to do, calling one or more repositories to do it.
`IngestService.IngestLocation` resolves the device's active route, inserts the location row,
updates the Redis cache, and broadcasts to the hub — four repository/cache/hub calls behind
one method (`services/ingest.go:29-81`, inventory §3.2). *See also:* **Handler**,
**Repository**, **Clean architecture**.

**Singleton** — a value that is constructed exactly once and shared everywhere it's needed,
rather than re-created per use. The frontend's WebSocket client is one:
`services/websocket/websocketClient.ts` constructs one `WebSocketClient` instance and exports
it directly (`websocketClient.ts:11-82`, inventory §1.19). *See also:* **Hub** (the backend's
equivalent — one `Hub` per process, not per request).

**Slice** — in this project's Zustand **store**, one focused piece of the global state plus
the actions that mutate it, composed together into the single store. `store/index.ts`
combines five: `busLocations`, `arrivals`, `stops`, `connection`, `ui`
(`store/index.ts:15`, inventory §1.18). Slices are meant to stay independent — no slice reads
another slice's state directly (CLAUDE.md §7.3). *See also:* **Store**, **Zustand**.

**SPA (Single-Page Application)** — a web app that loads one HTML page once and then updates
the DOM in place via JavaScript, instead of requesting a new page from the server on every
navigation. This frontend has no router at all — `App.tsx` is a single static layout
(`App.tsx:8-42`, inventory §2.2) — so it is not just a single-page app but a single-*screen*
one. *See also:* **React**, **Vite**.

**Spatial index** — a database index structured for "what's near this point" or "what
contains this point" queries, which a normal B-tree index can't answer efficiently. This
project uses two different kinds side by side: a **GIST index** (PostGIS, exact) and the
**H3** hex-cell index (approximate, O(1) lookup) — see CLAUDE.md §3.4 for why both exist.
*See also:* **PostGIS**, **H3**, **GIST index**.

**`ST_DWithin`** — a PostGIS function that tests whether two geometries are within a given
distance of each other — the SQL-level building block for "find everything near this point."
Used in the `GET /api/nearby/devices` and `GET /api/nearby/stops` queries
(`database/locations.go:157-178`, `database/stops.go:71-83`, inventory §3.7-3.8) — the underlying query works over any **device**.
*See also:* **PostGIS**.

**Stop** — a fixed point on a **route** that devices are tracked against — the thing a
**geofence** surrounds. Stored in the `stops` table (`id`, `route_id`, `name`, `latitude`,
`longitude`, `sequence_number`, `geom`; `001_create_tables.sql:21-29`, inventory §4.3) and
modeled in Go as `models.Stop` (`models/stop.go:3-9`, inventory §1.10). Despite the transit
flavor of the name, nothing about a stop is bus-specific — it's a point of interest a device
can approach or pass, and the demo happens to call it a bus stop. *See also:* **Geofence**,
**Device**.

**Store** — in Zustand, the single global object holding all client-side application state,
composed from **slice**s. `useStore` is this project's store hook (`store/index.ts:15`,
inventory §1.18); any component can read from it, but slices should stay independent of each
other. *See also:* **Slice**, **Zustand**.

**Telemetry** — data automatically reported by a remote device about its own state — here,
a device's GPS coordinates, speed, and accuracy, sent as a **ping**. *See also:* **Ping**,
**ETA**.

**Tile server** — a web server that serves pre-rendered map images as small square "tiles,"
which **Leaflet** stitches together into the visible map. This project uses CartoDB's public
Dark Matter tile server (`basemaps.cartocdn.com`, `MapContainer.tsx:44`), verified directly
against source for this document — the inventory itself only records "CartoDB dark tiles"
without the literal hostname (inventory §1.17). *See also:* **Leaflet**.

**Time-series** — data whose primary organizing dimension is time — a sequence of timestamped
measurements, as opposed to a table of independent entities. `location_history` is this
project's one time-series table, which is why it's a **TimescaleDB hypertable** rather than a
plain table. *See also:* **TimescaleDB**, **Hypertable**.

**TimescaleDB** — a PostgreSQL extension that turns a normal table into a **hypertable** —
transparently partitioned by time — and adds time-series-specific features like the
**compression policy**. Enabled by `001_create_tables.sql:2` (inventory §4); the Docker image
is `timescale/timescaledb-ha:pg15-latest` (inventory §2.3), so it's Postgres 15 with the
extension pre-installed, not a separate database engine.

**TTL (Time To Live)** — how long a cached value is kept before it's considered stale and
expires automatically. This project declares four TTL constants in
`cache/interface.go:52,55,58,61` (inventory §1.4); `DeviceLocationTTL` is 5 minutes.
*See also:* **Redis**, **Hot path**.

**Upgrader** — in a WebSocket library, the component that converts (upgrades) a normal HTTP
request into a persistent WebSocket connection. This backend's upgrader has
`CheckOrigin` hard-coded to always return `true` (`handlers/websocket.go:13-19`, inventory
§1.7) — meaning any website can open a WebSocket to this server, a real security gap outside
a local demo. *See also:* **WebSocket**.

**Vite** — the frontend build tool and dev server: it serves the app instantly during
development (no full bundle rebuild per change) and produces an optimized production bundle
via `npm run build` (`tsc -b && vite build`, `package.json:8`, inventory §8.4). *See also:*
**SPA**.

**Volume** — a Docker-managed persistent storage location that survives container restarts
and recreation (unlike a container's own writable filesystem layer, which is discarded).
This project has two: `postgres_data` and `redis_data`
(`infra/docker-compose.yml:114-118`, inventory §1.14). *See also:* **Docker**, **Container**.

**WebSocket** — a network protocol that upgrades a single HTTP request into a persistent,
bidirectional connection: after the handshake, either side can push messages at any time,
with no new request/response round trip needed per message. This is how live location updates
reach the browser — see `03-data-flow.md` for the full hop-by-hop path. *See also:* **Hub**,
**Upgrader**, **Ping**.

**Zustand** — the small state-management library this frontend uses for client-side state
(as opposed to server data or URL state). One global **store**, composed of five **slice**s
(inventory §1.18). *See also:* **Store**, **Slice**, **React**.
