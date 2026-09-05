# Transit POC: Architecture

## 1. System Overview
Transit is a full-stack, real-time vehicle tracking and ETA prediction system. It utilizes a monolithic Go backend with an Echo HTTP server, and a React 19 SPA frontend bundled with Vite.

## 2. Infrastructure & Data Layer
- **PostgreSQL / TimescaleDB**: The core persistent store. Time-series hypertable (`location_history`) stores all GPS pings. Compression policies are enforced.
- **PostGIS**: Extends Postgres for spatial queries (`ST_DWithin`, `ST_Distance`).
- **Redis**: An in-memory cache holding the latest known location of every device, preventing database thrashing on high-frequency WebSocket broadcasts.
- **Docker Compose**: Orchestrates the entire backend stack across a shared `transit-network` bridge.

## 3. Backend Architecture (Clean Monolith)
The backend enforces Clean Architecture patterns, structurally organized into layered components:
- `cmd/server/main.go`: The Composition Root. Initializes all dependencies, connects to DB/Redis, and wires handlers to services.
- `internal/database`: Concrete repositories (`LocationRepo`, `StopRepo`, etc.) interacting with Postgres via `pgx/v5`.
- `internal/services`: Business logic (e.g. `GeofencingService` using Uber's `H3` library to resolve bounding hexes, `ArrivalsService` calculating ETA).
- `internal/handlers`: Echo HTTP controllers.
- `internal/hub`: A native Go channel-based WebSocket hub broadcasting `LOCATION_UPDATE` events.

## 4. Frontend Architecture
- **Framework**: React 19 SPA, written in TypeScript, bundled by Vite.
- **State Management**: Zustand (organized into 5 slices: `busLocations`, `arrivals`, `stops`, `connection`, `ui`).
- **Styling**: Tailwind CSS with a highly customized "HUD" dark-mode palette.
- **Map Engine**: `react-leaflet` with CartoDB Dark Matter tiles.

## 5. Data Flow (Live Location Pings)
1. Device POSTs GPS coordinates to `/api/location`.
2. Backend `IngestLocation` handler writes immediately to TimescaleDB.
3. Postgres trigger generates PostGIS geometry from lat/lng.
4. Backend `IngestLocation` pushes the update to the WebSocket Hub via a channel.
5. The Hub loops through connected clients, writing the JSON payload to their TCP sockets.
6. Frontend `wsClient.onmessage` parses the JSON and triggers the Zustand `updateBus` action.
7. `<BusMarkers />` React component detects the new reference and moves the Leaflet map icon seamlessly.

## 6. Known Architectural Flaws / Drift
- **Handler Bypass**: The `ArrivalsService` (which implements H3 k-ring logic to determine if a bus is truly "approaching") is initialized in `main.go` but is **ignored** by the `ArrivalHandler`. The handler accesses repositories directly, dropping critical business logic.
- **Contract Mismatch**: The frontend expects `route_id` and `h3_hex` on WebSocket `LOCATION_UPDATE` payloads, but the backend `Message` struct drops these fields during serialization.
- **Ghost UI**: The frontend Operations Panel allows toggling an "H3 Spatial Grid" layer, but the Map rendering logic for this layer does not exist.
