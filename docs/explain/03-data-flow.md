# 03-data-flow.md — The life of one GPS ping, device to moving dot

This document follows **one** location report from a tracked device until it moves a marker on
someone's map. Nothing else. Other request paths (arrivals, nearby, route creation) are covered
in `04-components.md`.

Terms in **bold** on first use are defined in [`05-glossary.md`](./05-glossary.md).

**Sources.** Structure and route wiring come from [`00-inventory.md`](./00-inventory.md). Every
line number below was re-verified against the source file itself while writing this document.
Where the source disagreed with the inventory, the source wins and the difference is recorded
under [DEVIATIONS](#deviations) at the end.

---

## The path in one line

```
device → POST /api/location → LocationHandler → IngestService
                                                    ├─(a)→ TimescaleDB   (durable)
                                                    ├─(b)→ Redis         (hot state)
                                                    └─(c)→ Hub channel   (live)
                                                             ↓
                          Hub.Run() → per-client channel → WebSocket frame
                                                             ↓
       browser: websocketClient → messageHandler → Zustand store → <BusMarkers /> → marker moves
```

Fifteen hops follow. Hops 1–2 happen once, when a browser tab opens. Hops 3–15 happen for
every single ping.

---

## Part 0 — the standing connection (happens once per browser tab)

A ping can only reach a browser that is already listening. That listening connection is set up
long before any ping arrives, so it comes first.

### Hop 1 — The browser opens a WebSocket

A **WebSocket** is a network protocol that starts life as an ordinary HTTP request and then
*upgrades* into a connection that stays open. The difference matters here:

- **Ordinary HTTP** is one question, one answer. The browser asks, the server replies, the
  connection is finished. The server cannot speak first. To get fresh data the browser has to
  ask again, and again — polling.
- **A WebSocket** stays open after the handshake. Either side can send a message whenever it
  likes, with no new request. That is what makes a *pushed* location update possible at all.

**Arrives:** nothing — this is browser-initiated on component mount.

**Runs:** `useWebSocket()` calls `wsClient.connect()` inside a `useEffect` with an empty
dependency array, so it fires once per mount
(`frontend/src/hooks/useWebSocket.ts:13-21`). `connect()` constructs
`new WebSocket(WS_CONFIG.url)` (`frontend/src/services/websocket/websocketClient.ts:23`),
default `ws://localhost:8080/ws` (`frontend/src/config/wsConfig.ts:7`). It then attaches four
callbacks: `onopen`, `onmessage`, `onclose`, `onerror`
(`websocketClient.ts:26,37,41,50`). `onmessage` is the one that matters for a ping — it forwards
straight to `handleMessage` (`websocketClient.ts:37-39`).

**Leaves:** an HTTP `GET /ws` carrying the WebSocket upgrade headers.

**Note on where this lives.** The app's only live socket is opened as a side effect of
rendering the header status widget: `useWebSocket()` is called from
`frontend/src/components/Header/ConnectionStatus.tsx:11` (inventory §2.2). If that widget were
removed from the layout, live updates would stop, and nothing in the map code would say why.

### Hop 2 — The server upgrades it and registers a client

**Arrives:** `GET /ws`, routed to `wsHandler.HandleWS` (`backend/cmd/server/main.go:150`).

**Runs:** `upgrader.Upgrade` turns the HTTP request into a persistent connection
(`backend/internal/handlers/websocket.go:30`). The **upgrader**'s `CheckOrigin` returns `true`
unconditionally (`websocket.go:16-18`), so any web page anywhere may open this socket — a real
security gap outside a local demo. The handler then builds a `hub.Client` with its own
outbound **channel** buffered to 256 (`websocket.go:36`), pushes it onto the **Hub**'s
`Register` channel (`websocket.go:37`) — the Hub is the single object that owns every connected
client and fans broadcasts out to them; see "What the Hub is, and why it needs to exist" below
for the full explanation — and starts two goroutines: `WritePump` and `ReadPump`
(`websocket.go:41-42`).

`Register` is *unbuffered* (`backend/internal/hub/hub.go:18`), so line 37 blocks until
`Hub.Run()` picks the client up and adds it to the map of live clients (`hub.go:27-30`).

**Leaves:** one entry in `Hub.Clients`, and a goroutine sitting on `client.Send` waiting for
something to write.

**What each pump does.** `ReadPump` reads from the socket and throws the result away — the
loop at `backend/internal/hub/client.go:51-59` discards both the message type and the payload.
Its real jobs are (1) to notice when the connection dies and unregister the client
(`client.go:44-47`) and (2) to refresh the read deadline on each keep-alive pong
(`client.go:50`). **This server never accepts data from a browser over the WebSocket.** Traffic
is one-directional in practice: server → browser only.

`WritePump` is the goroutine that will eventually deliver our ping — see Hop 12.

---

## Part 1 — the ping arrives and is written

### Hop 3 — The device POSTs its coordinates

**Arrives:** an HTTP `POST /api/location` with a JSON body matching `models.Location`:
`device_id`, `latitude`, `longitude`, `speed`, `accuracy`, `timestamp`
(`backend/internal/models/location.go:4-12`).

Note what is **not** in that body: there is no `route_id`. The device does not get to claim
which route it is on. That is resolved server-side at Hop 7.

**Runs:** Echo's router matches the registration at `backend/cmd/server/main.go:132` and calls
`locHandler.IngestLocation`. The middleware chain (Recovery → CORS → Logging) runs first
(`main.go:114-116`).

**Leaves:** an `echo.Context` handed to the handler.

### Hop 4 — The handler validates and nothing else

**Arrives:** the raw request body.

**Runs:** `IngestLocation` binds JSON into a `models.Location`
(`backend/internal/handlers/location.go:28-31`), then applies four checks:

| Check | Line | Response if it fails |
|---|---|---|
| `device_id` non-empty | `location.go:34` | 400 |
| latitude ∈ [-90, 90], longitude ∈ [-180, 180] | `location.go:37` | 400 |
| speed ∈ [0, 350] | `location.go:40` | 400 |
| `timestamp == 0` → set to `time.Now().Unix()` | `location.go:45-47` | *(not a rejection — a default)* |

That is the whole of the handler's logic. It holds no repository, no cache, and no Hub — only
an `IngestService` interface (`location.go:15-17`). This is the **handler** layer doing exactly
what the layering intends: parse, validate, delegate.

**Leaves:** a validated `*models.Location`, and a call to `h.service.IngestLocation(ctx, &loc)`
(`location.go:51`).

### Hop 5 — The service takes over

**Arrives:** `ctx` and `*models.Location`.

**Runs:** `IngestService.IngestLocation`
(`backend/internal/services/ingest.go:36-81`). This one method owns all four side effects of a
ping. It holds four collaborators, injected at construction
(`backend/cmd/server/main.go:98`):

| Field | Type | Declared at |
|---|---|---|
| `locRepo` | `LocationRepository` (interface) | `ingest.go:18` |
| `routeRepo` | `DeviceRouteRepository` (interface) | `ingest.go:19` |
| `cache` | `DeviceCache` (interface) | `ingest.go:20` |
| `hub` | `*hub.Hub` — **a concrete type, not an interface** | `ingest.go:21` |

The Hub is the odd one out. Three of the four dependencies are interfaces the `services`
package declares for itself; the fourth reaches directly for the concrete `*hub.Hub`. The
inventory records this as a stated-layering violation (§7.3.k). In practice it means
`IngestService` cannot be unit-tested against a fake broadcaster without constructing a real
Hub.

**Leaves:** four things, in the order below.

### Hop 6 — Branch (a): the durable write to TimescaleDB

This is the only branch whose failure aborts the request.

**Arrives:** `*models.Location`.

**Runs:** `s.locRepo.Insert(ctx, loc)` (`ingest.go:37`) →
`LocationRepository.Insert` (`backend/internal/database/locations.go:47-75`). Three things
happen there:

1. **The H3 cell is computed.** If `loc.HexRes9` is empty, `CalculateHex` converts the
   lat/lng into an **H3** hexagonal-grid cell ID (`locations.go:49-51`, computed at
   `locations.go:37-44`). The **resolution** is hard-coded to `9` in the constructor
   (`locations.go:23`) — roughly a 175 m hex edge. The `H3_RESOLUTION` environment variable is
   parsed and range-validated but never reaches this line (inventory §5.1).
2. **The timestamp becomes a real time.** `time.Unix(loc.Timestamp, 0)`, falling back to
   `time.Now()` when the field is zero (`locations.go:54-57`). The handler already defaulted it
   at Hop 4, so this second fallback is normally unreachable through the HTTP path.
3. **One `INSERT` runs** against the `location_history` **hypertable**
   (`locations.go:60-63`), seven columns: `time, device_id, latitude, longitude, speed,
   accuracy, hex_res9`.

`geom` is **not** in that `INSERT`. A `BEFORE INSERT` trigger fills it in inside Postgres:
`trg_location_geom` calls `update_location_geom()`, which sets
`NEW.geom := ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326)`
(`backend/migrations/003_add_h3_postgis.sql:97-105,108-118`, per inventory §4.4). So the row
lands with both a hex ID (approximate, fast equality lookups) and a **PostGIS** point (exact,
for distance queries).

**Leaves:** one row in `location_history`, and — as a side effect — `loc.HexRes9` mutated on
the struct the caller still holds. Hops 8, 9 and 10 all read that mutated field.

**If it fails:** `IngestLocation` returns immediately with a wrapped error (`ingest.go:37-39`);
the handler turns that into a 500 (`location.go:51-54`). Redis is not written and nothing is
broadcast.

### Hop 7 — The route ID is resolved server-side

**Arrives:** `loc.DeviceID`.

**Runs:** `s.routeRepo.GetActiveRouteID(ctx, loc.DeviceID)` (`ingest.go:46`) →
`backend/internal/database/device_routes.go:29-44`, one query:

```sql
SELECT route_id FROM trips
WHERE device_id = $1 AND status = 'IN_PROGRESS'
ORDER BY started_at DESC LIMIT 1
```

A device with no active trip is not an error: `pgx.ErrNoRows` is converted to `("", nil)`
(`device_routes.go:38-40`).

**Leaves:** a `routeID` string, possibly empty.

**If it fails:** the error is logged with `log.Printf` and execution continues
(`ingest.go:47-49`). The durable write already succeeded; the service treats route
resolution as enrichment, not as part of the transaction.

**Cost note.** This is one extra database round-trip on every single ping. `trips` is written
only by the seed migration (inventory §4.5) — no Go code ever advances a trip — so the answer
is static for the lifetime of the demo, and the query is repeated anyway.

### Hop 8 — Branch (b): the hot-state write to Redis

**Arrives:** the location fields, assembled into a `map[string]interface{}` with keys
`latitude`, `longitude`, `speed`, `hex_res9`, `last_seen` (`ingest.go:51-57`).

**Runs:** `s.cache.SetDeviceState(...)` (`ingest.go:58`) →
`DeviceCache.SetDeviceState` (`backend/internal/cache/redis.go:184-205`). That method
type-asserts each map value back out into a typed `cache.DeviceLocation` struct
(`redis.go:186-202`) and delegates to `RedisCache.SetDeviceLocation`
(`redis.go:204`), which JSON-encodes it and writes it under key `device:<id>:loc` with a
5-minute **TTL** (`redis.go:50,57`; `DeviceLocationTTL` at
`backend/internal/cache/interface.go:52`).

**Leaves:** one Redis key holding this device's latest known position.

**If it fails:** logged, execution continues (`ingest.go:58-60`). Same reasoning as Hop 7.

**Two honest observations about this hop.**

1. `DeviceCache` is labelled a legacy backward-compatibility shim in its own source comment
   (`redis.go:170-173`, "Deprecated: Use CacheStore interface methods instead"). It is
   nevertheless the path that is actually wired (`main.go:85,98`). The typed, purpose-built
   `RedisCache.SetDeviceLocation` is reached only *through* the deprecated wrapper, because
   the interface the service declares (`services/interfaces.go:41`) exposes only the
   `map[string]interface{}` form. The map round-trip is pure overhead: typed struct → map →
   typed struct.
2. **Nothing reads this cache back.** `DeviceCache.GetDeviceState` (`redis.go:208-224`) and
   `RedisCache.GetDeviceLocation` (`redis.go:62-79`) exist; a `grep` across `backend/` for
   both names finds only their definitions and the wrapper's internal delegation — no caller
   anywhere. The write half of the hot path is built; the read half is not.

### Hop 9 — Branch (c): the message goes onto the Hub's Broadcast channel

**Arrives:** the location, plus `routeID` from Hop 7 and `loc.HexRes9` from Hop 6.

**Runs:** a `hub.Message` is constructed field by field (`ingest.go:62-72`) and pushed onto the
Hub's `Broadcast` channel inside a `select` with a `default:` arm (`ingest.go:73-78`).

The wire envelope is flat — no `data` wrapper, no `omitempty` on any field
(`backend/internal/hub/message.go:7-17`):

```go
type Message struct {
	Type      string  `json:"type"`
	DeviceID  string  `json:"device_id"`
	RouteID   string  `json:"route_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
	Accuracy  float64 `json:"accuracy"`
	H3Hex     string  `json:"h3_hex"`
	Timestamp int64   `json:"timestamp"`
}
```

`Type` is the constant `"LOCATION_UPDATE"`, uppercase (`message.go:20`). The absence of
[`omitempty`](./05-glossary.md) is deliberate: a stopped device must serialize `"speed": 0`
rather than dropping the key, or the browser cannot tell "stopped" apart from "not reported".
`backend/internal/hub/message_test.go` pins both properties — nine keys, no `data` wrapper, no
`heading`, and `speed` present as `0` (inventory §1.8).

Note the naming seam: the database column and the Go model field are both `hex_res9`
(`models/location.go:11`), while the wire field is `h3_hex` (`message.go:15`). One name
internally, a different one on the wire.

**Leaves:** a `Message` value sitting in a channel buffered to 256 (`hub.go:17`).

**If the buffer is full:** the `default:` arm fires and the message is **dropped**, with a
`log.Printf` (`ingest.go:75-77`) — so the drop is logged server-side, but no client is ever told
anything was missed. The ping is still in the database; only the live update is lost. The source
comment at `ingest.go:76` calls this out explicitly as "backpressure handling," not an oversight —
a slow or absent Hub must never block ingestion.

### Hop 10 — The device gets its 200

**Arrives:** a `nil` error back from the service.

**Runs:** the handler serializes a small JSON body.

**Leaves:** `200 OK` with `{"status":"ok","hex_res9":"<cell id>"}`
(`backend/internal/handlers/location.go:57-60`). The `hex_res9` value is only available here
because `Insert` mutated the struct in place at Hop 6 — a coupling the handler's own comment
calls out (`location.go:49-50`).

The device is now done. Hops 11–15 continue on other goroutines and in other processes.

---

## Why the write fans into three branches instead of one

Hops 6, 8 and 9 all write the same fact to three different places. That is intentional, and the
three have genuinely different jobs.

| Branch | Store | Optimized for | Failure behavior |
|---|---|---|---|
| (a) `LocationRepository.Insert` | TimescaleDB | **Durability and history.** Survives restart; compresses after 7 days (`003_add_h3_postgis.sql:87`); supports "where was this device an hour ago". | Aborts the request (`ingest.go:37-39`) |
| (b) `DeviceCache.SetDeviceState` | Redis | **Latency of a single read.** One in-memory key lookup for "where is device X *now*", instead of a `DISTINCT ON` scan over a time-series table. | Logged, ignored (`ingest.go:58-60`) |
| (c) `hub.Broadcast` | in-process channel | **Push.** Nobody asked; the update is delivered anyway, to everyone watching. | Dropped if the buffer is full (`ingest.go:73-78`) |

The reason for (a) *and* (b) is the classic split: the durable store is authoritative but slow
to answer "latest per device", and the cache is fast but expendable. Losing Redis loses nothing
permanent — it can be rebuilt from `location_history`. Losing Postgres loses the history.

The reason (c) is separate from both is that it is not a *store* at all. A cache answers a
question when someone asks. The Hub answers nobody; it pushes.

**State it plainly, though:** the read side of (b) does not exist yet in this codebase (Hop 8,
note 2). Today, Redis is written on every ping and never read. The durability-versus-latency
split is the *design*; only half of the latency half is implemented.

---

![Sequence diagram: one GPS ping travelling from device through handler, service, TimescaleDB, Redis and the Hub, out over the WebSocket to the browser map](../diagrams/ping-sequence.drawio.png)

---

## Part 2 — the ping is broadcast

### What the Hub is, and why it needs to exist

The **Hub** is the single object that owns the set of connected WebSocket clients
(`backend/internal/hub/hub.go:7-13`). It is built exactly once, in `main()`
(`backend/cmd/server/main.go:94`), and run on its own goroutine for the life of the process
(`main.go:95`).

Everything else in this backend is per-request: a handler is invoked for one HTTP call, a
service method runs for one caller, a repository query answers one question. A broadcast cannot
work that way. The set of clients is *shared, long-lived, and mutated concurrently* — a browser
can connect or vanish at any moment, while a ping arriving on a completely different goroutine
needs to iterate that same set. Some single component has to own it.

The Hub owns it through three channels and a mutex (a lock that lets only one goroutine touch
the shared client set at a time, preventing two concurrent writes from corrupting it)
(`hub.go:8-12`), so no caller ever touches `Clients` directly:

- `Register` — a new client joins (unbuffered)
- `Unregister` — a client leaves (unbuffered)
- `Broadcast` — one message to send to everyone (buffered 256)

That is the whole design: one input, N outputs — **fan-out**.

### The Hub has no shutdown path — this is an open, unfixed defect

`Hub.Run()` is an infinite `for { select { ... } }` over exactly three cases: `Register`,
`Unregister`, `Broadcast` (`backend/internal/hub/hub.go:24-54`). **There is no `ctx` case and
no `done` case.** Once started, the loop cannot be told to stop.

It is started fire-and-forget — `go wsHub.Run()` at `backend/cmd/server/main.go:95`, with no
handle kept and nothing to wait on. On `SIGINT`/`SIGTERM`, graceful shutdown calls
`e.Shutdown(ctx)` and nothing else (`main.go:170-177`, inventory §2.1 step 15). The Hub
goroutine, and every per-client `ReadPump`/`WritePump` goroutine (`websocket.go:41-42`), are
left running until the process dies.

This is tracked internally as **DEFECT-6** and is recorded in inventory §1.8 and §7.4.j. It is
**not fixed**. Two consequences worth knowing while reading the rest of this document:

1. Shutdown is abrupt for connected clients — no close frames, no drain of in-flight messages.
2. It blocks the tests that would pin down Hop 11's drop behavior, because a test cannot start
   a Hub and then stop it cleanly.

### Hop 11 — Hub.Run() fans the message out

**Arrives:** one `Message` on the `Broadcast` channel.

**Runs:** the third case of the select (`hub.go:38-51`). It takes a read lock, iterates every
registered client, and attempts a non-blocking send onto each client's own `Send` channel:

```go
for client := range h.Clients {
    select {
    case client.Send <- message:
    default:
        // buffer full — skip this client
    }
}
```

**Leaves:** one copy of the message on each healthy client's channel.

**The `default:` arm is a silent drop.** If a client's 256-slot buffer is full — a browser on a
bad connection, a tab the OS has throttled — that client simply misses the update
(`hub.go:43-49`). Nothing is logged, no metric is emitted, and the client is not disconnected.
The source comments there acknowledge this and note that a more robust system would disconnect
the client instead. As written, a stuck client silently and permanently falls behind, because
its buffer never drains.

Note also that this loop drops for *individual* clients, while Hop 9 drops for *everyone* at
once. Two different drop points on the same path, and they fail differently: Hop 9's drop is
logged server-side but silent to clients; this one is silent both server-side and to the client
— nothing records it happened at all.

### Hop 12 — WritePump turns it into a WebSocket frame

**Arrives:** the message on `client.Send`.

**Runs:** `Client.WritePump`, the goroutine started at Hop 2
(`backend/internal/hub/client.go:66-93`). It sets a 10-second write deadline
(`client.go:75`) and calls `c.Conn.WriteJSON(message)` (`client.go:82`) — this is where the Go
struct is finally serialized to the JSON described at Hop 9.

The same loop has a second case: a ticker that sends a WebSocket keep-alive ping every
`pingPeriod` (`client.go:86-90`, `pingPeriod = (pongWait * 9) / 10 = 54s`, `client.go:15,18`).
That is protocol plumbing to detect dead connections and has **nothing to do with a GPS ping**
— the glossary entry for [Ping](./05-glossary.md) covers both meanings of the word.

If the hub closed the channel (client unregistered), `ok` is false and the pump sends a close
frame and returns (`client.go:76-80`).

**Leaves:** JSON bytes on the wire, one WebSocket text frame:

```json
{"type":"LOCATION_UPDATE","device_id":"bus-01","route_id":"route-101",
 "latitude":37.7749,"longitude":-122.4194,"speed":0,"accuracy":5,
 "h3_hex":"8928308280fffff","timestamp":1756500000}
```

---

## Part 3 — the browser moves the dot

### Hop 13 — The socket callback fires

**Arrives:** a `MessageEvent` on the open WebSocket from Hop 1.

**Runs:** the `onmessage` callback set at
`frontend/src/services/websocket/websocketClient.ts:37-39`. It does one thing: forward the raw
event to `handleMessage`.

**Leaves:** a call into the message handler.

There is no queue, no batching, and no throttle here. One frame in, one synchronous handler
call, one store write, one React render pass. At the throughput this repo targets that is a
design decision worth being aware of, not a bug — but it is worth naming.

### Hop 14 — The message is parsed and dispatched

**Arrives:** `event.data`, a JSON string.

**Runs:** `handleMessage` (`frontend/src/services/websocket/messageHandler.ts:11-52`):

1. `JSON.parse(event.data)`, wrapped in a `try/catch` that logs and swallows bad frames
   (`messageHandler.ts:13,49-51`).
2. A `switch` on `raw.type` across six cases plus a `default`
   (`messageHandler.ts:20-48`). Our message matches
   `WS_CONFIG.messageTypes.location_update`, which is the string `'LOCATION_UPDATE'`
   (`frontend/src/config/wsConfig.ts:20`) — uppercase, matching the Go constant exactly.
3. `handleLocationUpdate(raw)` (`messageHandler.ts:61-90`) reads each field through the schema
   map at `wsConfig.ts:35-46`, which exists so the wire names live in one file rather than
   being scattered as string literals.

The field mapping, in full:

| Wire field | Read at | Becomes | Notes |
|---|---|---|---|
| `device_id` | `messageHandler.ts:68` | `id` | falsy → return early, message discarded (`:69`) |
| `route_id` | `messageHandler.ts:76` | `routeId` | `''` when the device has no active trip |
| `latitude` | `messageHandler.ts:77` | `lat` | `parseFloat` |
| `longitude` | `messageHandler.ts:78` | `lng` | `parseFloat` |
| `speed` | `messageHandler.ts:79` | `speed` | `?? 0` — nullish, so a real `0` survives |
| `timestamp` | `messageHandler.ts:71-72` | `lastUpdate` | `new Date(unixSeconds * 1000)` |
| `h3_hex` | `messageHandler.ts:81` | `h3Hex` | optional |
| `accuracy` | — | — | **declared in the schema (`wsConfig.ts:43`) but never read.** `BusLocation` has no accuracy field (`frontend/src/types/domain.ts:41-49`). The backend sends it; the frontend discards it. |

A last guard rejects unparseable coordinates: `if (isNaN(location.lat) || isNaN(location.lng))
return;` (`messageHandler.ts:84`).

**Leaves:** `useStore.getState().updateBus(location)` (`messageHandler.ts:86`).

**Dead branches in this switch.** Five of the six cases can never fire. The backend defines
exactly one message type, `"LOCATION_UPDATE"` (`hub/message.go:20`), so
`handleConnected`, `handleArrivalUpdate`, `handleRouteUpdate`, `handleHeartbeat` and
`handleError` have no producer (inventory §6.6). These "six cases" are the `switch` statement's
own branches, one per message type it knows how to handle. Separately, `wsConfig.ts:14-30`
*declares* nine message-type string constants in total — the six the switch has a case for, plus
three more (`disconnected`, `heartbeat_ack`, `system_message`) that were never even given a
`case` at all. Counting both groups together, eight of the nine declared types are unreachable;
only `LOCATION_UPDATE` has a live producer. This is half-built plumbing for messages the server
was never taught to send.

### Hop 15 — The store updates and the marker moves

**Arrives:** a `BusLocation` object.

**Runs, in three steps:**

1. **The Zustand slice writes.** `updateBus` copies the `buses` Map and sets one key:
   `set((state) => ({ buses: new Map(state.buses).set(bus.id, bus) }))`
   (`frontend/src/store/slices/busLocations.ts:18`). The copy is the point — a **new** Map
   identity is what tells React something changed. Mutating the existing Map in place would
   update the data and render nothing.
2. **`<BusMarkers />` re-renders.** It subscribes to three pieces of store state:
   `buses` (`frontend/src/components/Map/BusMarkers.tsx:28`), `layerVisibility` (`:29`), and
   `hiddenRoutes` (`:30`). The changed `buses` reference wakes it. It returns `null` outright
   if the buses layer is toggled off (`:32`), then filters out any device whose `routeId` is in
   `hiddenRoutes` (`:34-36`).
3. **Leaflet moves the marker.** Each surviving device renders one `<Marker>` keyed by
   `bus.id`, positioned at `[bus.lat, bus.lng]` (`BusMarkers.tsx:41-44`). Because the key is
   stable across renders, **react-leaflet** updates the existing marker's position rather than
   destroying and recreating it — which is why the dot appears to move rather than blink.

**Leaves:** a repositioned dot on the map. The popup attached to it shows device ID, lat/lng,
speed and route, plus a "FOLLOW TARGET" button that writes `followedBusId` back into the store
(`BusMarkers.tsx:85`).

`<BusMarkers />` is mounted by `MapContainer.tsx:57`. `MapContainer` itself reads no store
state at all (inventory §1.17) — it is a static shell; every live layer subscribes for itself.

---

## What is broken or half-built on this exact path

Collected in one place, plainly. Everything here is on the ping path described above.

| # | Issue | Where |
|---|---|---|
| 1 | **The Hub has no shutdown path.** `Run()` is an infinite loop with no `ctx`/`done` case; started fire-and-forget; graceful shutdown only stops Echo. Open defect (DEFECT-6). | `hub/hub.go:24-54`, `main.go:95,170-177` |
| 2 | **Two silent drop points.** A full `Broadcast` buffer drops for everyone (logged); a full per-client buffer drops for that client (not logged, no disconnect, never recovers). | `services/ingest.go:73-78`, `hub/hub.go:43-49` |
| 3 | **The Redis hot path is write-only.** Every ping writes it; no code reads it back. Verified by grep for `GetDeviceState` / `GetDeviceLocation` across `backend/`. | `cache/redis.go:208-224`, `cache/redis.go:62-79` |
| 4 | **The wired cache path is a self-declared deprecated shim,** and it round-trips a typed struct through a `map[string]interface{}` and back. | `cache/redis.go:170-173,184-205` |
| 5 | **`IngestService` depends on the concrete `*hub.Hub`,** not an interface, unlike its other three collaborators. Blocks testing broadcast in isolation. | `services/ingest.go:21` |
| 6 | **A per-ping query for a value that never changes.** `trips` is written only by the seed migration; the route lookup runs on every ping anyway. | `services/ingest.go:46`, `database/device_routes.go:29-44` |
| 7 | **`accuracy` is sent and then discarded.** It is in the Go struct, on the wire, and in the frontend schema map — and no frontend code reads it. | `hub/message.go:14`, `wsConfig.ts:43`, `domain.ts:41-49` |
| 8 | **Five of six message-handler branches are unreachable.** The backend defines one message type; the frontend handles six and declares nine. | `hub/message.go:20`, `messageHandler.ts:20-48`, `wsConfig.ts:14-30` |
| 9 | **`models.LocationUpdate` is a dead parallel wire struct.** It duplicates `hub.Message` minus `route_id`, and nothing constructs it. The live wire type is `hub.Message`. | `models/location.go:15-24` (inventory §6.1) |
| 10 | **The WebSocket upgrader accepts every origin.** `CheckOrigin` returns `true` unconditionally, so any page on any domain can open this socket. | `handlers/websocket.go:16-18` |
| 11 | **`ReadPump` discards everything a client sends.** The connection is bidirectional by protocol and one-directional in practice. Not a bug — but nothing documents it at the API surface. | `hub/client.go:51-59` |
| 12 | **Seed data is inserted twice on a fresh volume,** which inflates `location_history` before any real ping arrives. The compose file mounts the migrations into Postgres's own init path *and* the Go binary replays them through an empty ledger. | `infra/docker-compose.yml:23`, `database/db.go:57`, `004_seed_data.sql:94-99,102-105,108-111` (inventory §4.7) |

---

## DEVIATIONS

Points where the source, read directly for this document, disagrees with
[`00-inventory.md`](./00-inventory.md). Recorded, not corrected — the inventory is not edited by
this document.

1. **`docs/diagrams/ping-sequence.drawio.png` exists** and is embedded above. No deviation —
   noted only because the instruction required reporting its absence if missing.

2. **Inventory §6.2 marks `RedisCache.SetDeviceLocation` as UNREACHABLE-BY-INTERFACE.** It is
   reachable, and it runs on every ping: `DeviceCache.SetDeviceState` calls it directly at
   `backend/internal/cache/redis.go:204`. The narrower claim the inventory makes — that it is
   absent from the `services.DeviceCache` interface — is correct
   (`services/interfaces.go:41`). But "unreachable" overstates it. `GetDeviceLocation`
   (`redis.go:62-79`) is likewise reached from `GetDeviceState` (`redis.go:209`) — though since
   `GetDeviceState` itself has no caller, that whole read chain is dead at the top.

3. **Inventory §6.6 records `BusLocation.heading` as a "PERMANENT GHOST" required field at
   `domain.ts:41-50`.** That field no longer exists. `frontend/src/types/domain.ts:41-49`
   declares seven fields — `id`, `routeId`, `lat`, `lng`, `speed`, `lastUpdate`, `h3Hex` — and
   no `heading`. `messageHandler.ts:74-82` does not set one either. The inventory's finding was
   true of an earlier revision of these files.

4. **Line drift in `frontend/src/services/websocket/messageHandler.ts`.** The file is 129
   lines, not the 132 the inventory records (§1.19), and the handlers after
   `handleLocationUpdate` sit two lines earlier than cited. Verified positions:
   `handleLocationUpdate` `:61-90` (inventory: `:61-92`), `handleArrivalUpdate` `:92-111`
   (`:94-113`), `handleRouteUpdate` `:113-117` (`:115-119`), `handleHeartbeat` `:119-121`
   (`:121-123`), `handleError` `:123-129` (`:125-131`). `handleMessage` `:11-52` and its switch
   `:20-48` match. All citations in this document use the verified positions.

5. **`BusMarkers.tsx` — `setFollowedBusId` is at `:85`, not `:89`** (inventory §1.17). The
   other three citations in that row (`:12-25`, `:28`, `:29`, `:30`) are exact.

6. **`services/ingest.go` — `IngestLocation` spans `:36-81`, not `:29-81`** (inventory §1.11,
   §3.2). Line 29 is the first line of the doc comment; the `func` keyword is at line 36. Not
   an error so much as a convention difference, recorded because this document cites the
   function body repeatedly.

7. **`handlers/location.go` — the timestamp default spans `:45-47`, not `:45-46`**
   (inventory §3.2). Trivial, recorded for completeness.
