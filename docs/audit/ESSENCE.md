# Transit POC: The Essence

## The True Nature of the Codebase
Transit is a **real-time spatial event broadcaster**. 

At its absolute core, it strips away complex routing engines (like OSRM) in favor of high-throughput telemetry ingestion. It solves one specific problem: getting a GPS ping from a moving vehicle into the browser of an end-user as fast as possible, using a hybrid spatial index to determine if that vehicle is near a bus stop.

## The Core Machinery
1. **The Firehose**: A single REST endpoint (`POST /api/location`) receives GPS pings.
2. **The Split**: The data is instantly bifurcated. Path A writes the data to a time-series disk (TimescaleDB) for historical analytics. Path B pushes the data into memory (Redis) and fires it through a Go channel.
3. **The Broadcast**: A monolithic WebSocket hub reads the channel and sprays the raw payload to every connected browser concurrently.
4. **The Visualizer**: A React/Zustand SPA catches the WebSocket payloads and manipulates SVG markers on a Leaflet map bypassing traditional DOM updates for speed.

## The Illusion
The repository presents itself structurally as a heavily-abstracted Enterprise Clean Architecture application (with distinct Handler, Service, and Repository layers). However, in execution, it violates its own architecture for expediency. The `ArrivalHandler` bypasses the `ArrivalsService` entirely, calculating raw math against database repositories directly. The UI presents a sophisticated "H3 Grid Layer", but no underlying rendering logic exists for it.

## The Verdict
It is a fast, highly-concurrent telemetry pipeline dressed up as a complex Transit application. Its strength lies in its Dockerized deployment simplicity and raw WebSocket performance; its weakness lies in its test coverage and structural drift between its intended Clean Architecture and its actual execution flow.
