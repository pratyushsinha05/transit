# Frontend Features Ideation Prompt for Deepsearch

You are an expert UX/UI Designer and Frontend Architect. We are building the frontend for a **Real-Time Transit Application** (Transit POC) and need to define the core features, pages, sidebar menus, and interactive and operational buttons. 

The backend is already implemented and has the following capabilities and data models. Based *strictly* on what the backend supports (or can easily support with its current entities), please propose a comprehensive list of frontend features.

## Backend Capabilities & Context
Our backend is written in Go and uses PostgreSQL + PostGIS, TimescaleDB, and Redis. It provides the following:

1. **Real-time WebSocket Hub (`WSAdapter`)**: 
   - Broadcasts live `LOCATION_UPDATE` events (containing latitude, longitude, and speed) to connected clients. 
2. **REST APIs**:
   - `POST /api/location`: Ingests GPS telemetry (lat, lng, speed, timestamp) from devices/buses.
   - `GET /api/arrivals?stop_id={id}`: Returns ETA (estimated time of arrival in minutes) and active trips for a specific bus stop.
3. **Core Database Entities**:
   - **`devices`**: Transit vehicles (e.g., `id: bus-001`, `name`, `status`). 
   - **`stops`**: Physical bus stops with geospatial coordinates (`geom`) and sequence order along a specific route.
   - **`location_history`**: A time-series ledger of all device movements.
   - **Trips & Routes**: Entities linking vehicles to sequences of stops.

## Your Task
Please act as the product manager and lead designer and provide a detailed feature specification for the frontend (which is being built in React/Vite with Tailwind CSS). 

Include the following in your response:

1. **Information Architecture (Navigation / Sidebar)**
   - What should the main navigation look like? (e.g., Live Map, Routes, Fleet Status)
   - What sidebar menus or navigation tabs are most appropriate?

2. **Core Pages & Views**
   - Detail the primary screens (e.g., a "Dashboard" or "Live Tracking Map").
   - What information should be displayed on each screen? (e.g., showing bus markers moving in real-time on a map, plotting stops).

3. **Interactive Elements (Buttons, Modals, Forms)**
   - What specific buttons should exist? (e.g., "Simulate Bus Movement", "View Stop Schedule", "Track Specific Bus").
   - What happens when a user clicks a bus marker versus a bus stop marker?

4. **User Roles / Personas (Optional, but recommended)**
   - Should we distinguish between an "Admin/Dispatcher" view (seeing all fleet statuses) and a "Passenger" view (looking for arrivals at a specific stop)?

**Constraint:** Do not propose features that require massive backend rewrites (e.g., user ticketing/payments), unless you specify that it would require new backend endpoints. Keep the focus tightly aligned to location tracking, ETA prediction, and transit visualization.
