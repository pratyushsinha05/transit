/**
 * API Configuration
 * Single source of truth for all API contracts
 */

export const API_CONFIG = {
    // Base URLs
    BASE_URL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
    WS_URL: import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws',

    // Endpoints (based on actual Go backend)
    endpoints: {
        arrivals: {
            path: '/api/arrivals',           // GET: Upcoming bus arrivals for stop
            method: 'GET',
            description: 'Get upcoming arrivals for a stop',
            timeout: 10000,
        },

        stops: {
            path: '/api/stops',              // GET: All bus stops
            method: 'GET',
            description: 'Get all bus stops with coordinates',
            timeout: 20000,
            cacheTime: 3600000,              // Cache for 1 hour
        },

        routes: {
            path: '/api/routes',             // GET: All bus routes
            method: 'GET',
            description: 'Get all bus routes and their patterns',
            timeout: 20000,
            cacheTime: 3600000,
        },

        location: {
            path: '/api/location',           // POST: User's device location
            method: 'POST',
            description: 'Post device location for server-side distance calculations',
            timeout: 5000,
            optional: true,                  // Not critical if fails
        },

        health: {
            path: '/health',                 // GET: Backend health check
            method: 'GET',
            description: 'Check backend service health',
            timeout: 5000,
            interval: 30000,                 // Check every 30 seconds
        },
    },

    // Response schemas (from Go backend)
    // Format: component expects these field names
    schemas: {
        // Mirrors backend services.ArrivalPrediction exactly -- see
        // backend/internal/services/arrivals.go. There is no bus_id, status,
        // route, or timestamp field on this response.
        arrival: {
            id: 'trip_id',                   // trip_id doubles as the arrival's unique ID
            tripId: 'trip_id',
            deviceId: 'device_id',
            deviceName: 'device_name',
            etaMinutes: 'eta_minutes',
            distanceKm: 'distance_km',
            currentSpeed: 'current_speed',
            hexRes9: 'hex_res9',             // omitempty on the backend
            isApproaching: 'is_approaching',
        },

        stop: {
            id: 'id',                        // Stop ID
            name: 'name',                    // Stop name
            lat: 'latitude',                 // Latitude (float)
            lng: 'longitude',                // Longitude (float)
            address: 'address',              // Street address
            arrivals: 'arrivals',            // Array of upcoming arrivals (optional)
            h3_hex: 'h3_hex',               // H3 hexagon ID (for geospatial queries)
        },

        route: {
            id: 'id',                        // Route ID
            number: 'number',                // Route number (e.g., '42')
            name: 'name',                    // Route name
            description: 'description',      // Route description
            stops: 'stops',                  // Array of stop IDs in order
            pattern: 'pattern',              // GeoJSON LineString of route path
        },

        bus: {
            id: 'id',                        // Bus ID
            routeId: 'route_id',             // Current route
            lat: 'latitude',                 // Current latitude
            lng: 'longitude',                // Current longitude
            speed: 'speed',                  // Speed in km/h
            heading: 'heading',              // Direction in degrees (0-360)
            lastUpdate: 'last_updated',      // Timestamp of last location update
            h3_hex: 'h3_hex',               // Current H3 hexagon (backend-calculated)
        },

        health: {
            status: 'status',                // 'ok' or 'error'
            timestamp: 'timestamp',          // Server time
            uptime: 'uptime',                // Uptime in seconds
        },
    },

    // Query parameter configurations
    queryParams: {
        arrivals: {
            stop_id: 'stop_id',              // Required: Stop ID
        },
    },

    // Timeouts & retry config
    timeouts: {
        default: 30000,
        streaming: 60000,
        websocket: 45000,
    },

    // Retry configuration
    retries: {
        maxAttempts: 3,
        baseDelay: 100,                    // milliseconds
        maxDelay: 5000,
        exponentialBackoff: true,
    },
}
