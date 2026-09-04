# LatitudeX Transit Engine

> High-throughput real-time geo-telemetry ingestion, spatial indexing, and WebSocket broadcast engine built with Go, TimescaleDB, PostGIS, Redis, and React.

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://golang.org)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.9-3178C6?style=flat&logo=typescript)](https://www.typescriptlang.org/)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react)](https://react.dev/)
[![TimescaleDB](https://img.shields.io/badge/TimescaleDB-PostgreSQL_16-FDB515?style=flat&logo=postgresql)](https://www.timescale.com/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker)](https://www.docker.com/)

---

## ⚡ Overview

LatitudeX Transit Engine is an event-driven telemetry platform designed to track mobile devices (vehicles, couriers, or transit fleets) in real time. It ingests high-frequency GPS coordinate streams, evaluates spatial geofencing triggers against designated zones, persists time-series history, and broadcasts updates to connected web clients with sub-millisecond dispatch.

### Core Capabilities
* **Dual-Path Ingestion Pipeline**: Ingests GPS telemetry via HTTP REST and bifurcates data in parallel:
  * **Analytical Path**: Writes partitioned time-series points to TimescaleDB with automatic PostGIS geometry conversion.
  * **Operational Path**: Updates Redis hot-cache for low-latency geospatial lookups and forwards pings to the streaming hub.
* **Lock-Isolated WebSocket Hub**: Monolithic broadcast hub with independent client `WritePump`/`ReadPump` goroutines, non-blocking broadcasts, atomic drop accounting, and a parallel graceful shutdown drain.
* **Hybrid Geospatial Architecture**: Combines Uber H3 hexagonal indexing (Resolution 9, ~100m edge length) for $O(1)$ cell lookups with PostGIS spherical geography calculations (`ST_DWithin`, `ST_Distance`).
* **Reactive Map SPA**: High-performance React 19 / Zustand dashboard utilizing Leaflet to smoothly animate device positions and geofence events with zero unnecessary DOM re-renders.

---

## 🏗️ System Architecture

```
[ Mobile Devices / GPS Simulators ]
               │
               ▼  HTTP POST /api/location
   ┌──────────────────────┐
   │    Echo HTTP API     │
   └──────────┬───────────┘
              │
              ▼  Ingestion Service
     ┌────────┴────────┐
     │                 │
     ▼                 ▼
┌──────────────┐ ┌──────────────┐
│ TimescaleDB  │ │ Redis Cache  │
│  (History)   │ │  (Hot State) │
└──────────────┘ └──────┬───────┘
                        │
                        ▼
               ┌─────────────────┐
               │  WebSocket Hub  │
               └────────┬────────┘
                        │ Broadcast Channel
                        ▼
             [ Connected React SPAs ]
```

For a comprehensive technical walkthrough of data flows, spatial math, concurrency safety, and database schema, see **[ARCHITECTURE.md](ARCHITECTURE.md)**.

---

## 📊 Measured Performance

Performance figures are measured on local development hardware (Apple M-series, macOS, arm64) using Go's standard benchmark toolchain:

| Metric | Measured Baseline | Operational Target | Status | Notes |
|---|---|---|---|---|
| **Ingest Handler Overhead** | **~2.13 µs** (2,127 ns/op) | $< 20\text{ ms}$ p99 | ✅ Pass | Isolates HTTP routing, JSON decode/encode, and validation; leaves $\sim19.9\text{ms}$ budget for DB/Redis I/O |
| **Ingest Allocations** | **8,419 B/op** (42 allocs/op) | Minimal heap churn | ✅ Pass | Primarily Echo context allocation and request/response buffer sizing |
| **Single-Core Ingest Ceiling** | **~468,000 req/s** | $\ge 1,000\text{ pings/s}$ | ✅ Pass | CPU decoding capacity exceeds target throughput by two orders of magnitude |
| **WebSocket Hub Shutdown** | **$\le$ 250 ms** | Bounded & non-blocking | ✅ Pass | Deterministic parallel client drain with zero goroutine leaks under `-race` |
| **Test Suite Concurrency** | **Zero Data Races** | Strict `-race` pass | ✅ Pass | All concurrency tests pass with race detector enabled and zero hardcoded sleeps |

---

## 🚀 Quick Start

### Prerequisites
* **Docker & Docker Compose** (Docker Desktop or Colima)
* **Go 1.25+** (optional, for local development outside Docker)
* **Node.js 20+ & npm** (for frontend development)
* **Make**

### 1. Launch Infrastructure & Backend
Start PostgreSQL (with TimescaleDB & PostGIS), Redis, and the backend engine using Docker Compose:

```bash
# Verify system dependencies
make check-deps

# Spin up all containers in background
make docker-up

# Verify health check
make docker-health
```

The services will be exposed at:
* **Backend API & WebSocket**: `http://localhost:8080` (Health check: `http://localhost:8080/health`)
* **PostgreSQL + TimescaleDB**: `localhost:5432` (`transit_user` / `transit_password`, database `transit_db`)
* **Redis**: `localhost:6379`

### 2. Launch Frontend SPA
In a separate terminal, start the React development server:

```bash
cd frontend
npm ci
npm run dev
```

Open `http://localhost:5173` to view the live interactive tracking map.

---

## 🛠️ Developer Commands

The project includes standardized targets via the root `Makefile`:

```bash
# Docker Orchestration
make docker-up          # Build and start all services
make docker-down        # Stop services (preserves data volumes)
make docker-clean       # Stop and remove all containers and volumes
make docker-logs        # Stream logs from all services

# Database Operations
make db-migrate         # Run embedded SQL migrations (001 -> 005)
make db-seed            # Populate database with demo routes, zones, and devices

# Quality & Verification
make test               # Run backend unit tests with race detection (-race -count=1)
make check              # Run formatting, linting, and structural layering checks
```

---

## 📁 Repository Layout

```
.
├── backend/
│   ├── cmd/server/          # Application entrypoint & dependency injection
│   ├── internal/
│   │   ├── cache/           # Redis geospatial & TTL cache client
│   │   ├── config/          # Environment configuration loader
│   │   ├── database/        # PostgreSQL / TimescaleDB repositories (pgx/v5)
│   │   ├── handlers/        # Echo HTTP & WebSocket request handlers
│   │   ├── hub/             # High-throughput concurrent WebSocket broadcast engine
│   │   ├── middleware/      # CORS, logging, and panic recovery middleware
│   │   ├── models/          # Core domain structures (Device, Zone, Trip, Route)
│   │   └── services/        # Business logic (Ingestion, Geofencing, Predictions)
│   ├── migrations/          # Embedded SQL migrations (TimescaleDB + PostGIS)
│   ├── pkg/geo/             # Haversine distance & along-route calculation helpers
│   └── test/                # Multi-container integration tests
├── frontend/
│   ├── src/
│   │   ├── components/      # Map view, Leaflet markers, sidebar panels
│   │   ├── config/          # API & WebSocket connection endpoints
│   │   ├── hooks/           # React lifecycle hooks
│   │   ├── services/        # WebSocket client & REST transformers
│   │   ├── store/           # Zustand reactive state slices (Devices, Zones, UI)
│   │   └── types/           # Frontend domain TypeScript definitions
│   └── vite.config.ts       # Vite configuration
├── infra/
│   └── docker-compose.yml   # Multi-service composition (Postgres/Timescale, Redis, Backend)
├── scripts/
│   └── gate.sh              # Quality gate (layering enforcement, vet, formatting)
├── ARCHITECTURE.md          # Deep-dive system design, spatial math & concurrency specs
└── README.md                # This file
```

---

## 🧪 Testing & Verification

The test suite enforces zero-sleep, race-free concurrency:

```bash
# Run backend test suite with race detector
cd backend
go test ./... -race -count=1

# Run integration tests against real database
go test -tags=integration ./test/... -v -count=1

# Run frontend typecheck and production build
cd ../frontend
npx tsc --noEmit && npm run build
```
