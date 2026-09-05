# RESULTS.md — Measured Performance

> These numbers are measured, not claimed. Methodology, raw test output, and environment are included.
> Tracks progress against performance targets in `CLAUDE.md` §9.

---

## Benchmark: IngestLocation Handler (Phase 1B Baseline)

### 1. Scope & Methodology

- **What it measures:** HTTP boundary overhead for `POST /api/location` using Echo router and httptest, with a mocked `IngestService`. This isolates HTTP deserialization, JSON body parsing and type binding, coordinate and speed validation bounds checks, default timestamp injection, and JSON response serialization.
- **What it does NOT measure:** Real database persistence (TimescaleDB insert latency), PostGIS geometry trigger execution, Redis hot-cache writes, WebSocket hub fan-out to connected clients, or physical network transport. End-to-end latency benchmarks will be established when real infrastructure is under test in Phase 5+.

### 2. Environment

- **Go version:** `go1.26.0`
- **OS/Arch:** `darwin/arm64` (macOS)
- **CPU:** Apple M5 Pro

### 3. Raw Benchmark Output

```
goos: darwin
goarch: arm64
pkg: transit-backend/internal/handlers
cpu: Apple M5 Pro
BenchmarkIngestLocation-15    	 1691887	      2127 ns/op	    8419 B/op	      42 allocs/op
BenchmarkIngestLocation-15    	 1695256	      2141 ns/op	    8419 B/op	      42 allocs/op
BenchmarkIngestLocation-15    	 1707676	      2127 ns/op	    8419 B/op	      42 allocs/op
BenchmarkIngestLocation-15    	 1678935	      2135 ns/op	    8419 B/op	      42 allocs/op
BenchmarkIngestLocation-15    	 1686612	      2135 ns/op	    8419 B/op	      42 allocs/op
PASS
ok  	transit-backend/internal/handlers	29.040s
```

### 4. Interpretation

| Metric | Measured Baseline | Target (`CLAUDE.md` §9) | Status | Notes |
|---|---|---|---|---|
| Ingest Handler Latency | **~2.13 µs** (2,133 ns/op) | < 20 ms p99 (excluding client network) | Well within budget | Measures HTTP handler + JSON decode/encode; leaves ~19.99ms budget for DB + Redis I/O |
| Ingest Handler Allocations | **8,419 B/op** across **42 allocs/op** | — | Baseline | Primarily Echo context allocation, request JSON parsing, and response buffer formatting |
| Theoretical Max Ingest Throughput (Single Core) | **~468,000 req/s** | ≥ 1,000 pings/sec sustained | Pass | CPU capacity for HTTP decoding alone far exceeds the monolith throughput ceiling |
