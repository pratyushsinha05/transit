/**
 * WebSocket Configuration
 * Message types and schemas for real-time communication
 */

export const WS_CONFIG = {
    url: import.meta.env.VITE_WS_URL || 'ws://localhost:8080/ws',
    reconnectInterval: 3000,
    heartbeatInterval: 30000,

    // Message types sent by Go backend
    messageTypes: {
        // Connection lifecycle
        connected: 'connected',            // Backend: "{type: 'connected'}"
        disconnected: 'disconnected',
        heartbeat: 'heartbeat',            // Ping-pong every 30s
        heartbeat_ack: 'heartbeat_ack',

        // Real-time location updates
        location_update: 'location_update', // Bus location changed

        // Real-time arrival updates
        arrival_update: 'arrival_update',   // ETA updated for arrival

        // Route information
        route_update: 'route_update',       // Route pattern/stops updated

        // Error/system messages
        system_message: 'system_message',
        error: 'error',
    },

    // Message schema mappings (what backend sends)
    schemas: {
        locationUpdate: {
            // Expected by component -> Backend field name
            busId: 'bus_id',                 // Unique bus ID
            lat: 'latitude',                 // Current latitude
            lng: 'longitude',                // Current longitude
            routeId: 'route_id',             // Current route
            speed: 'speed',                  // Speed km/h
            heading: 'heading',              // Direction 0-360
            lastUpdate: 'last_updated',      // ISO 8601 timestamp
            h3_hex: 'h3_hex',               // H3 hexagon (server calculated)
        },

        arrivalUpdate: {
            stopId: 'stop_id',               // Which stop
            busId: 'bus_id',                 // Which bus
            eta: 'eta',                      // Minutes until arrival
            status: 'status',                // 'on_time', 'delayed', 'arriving'
            routeId: 'route_id',             // Route number
            timestamp: 'timestamp',          // When update generated
        },

        routeUpdate: {
            routeId: 'route_id',
            pattern: 'pattern',              // GeoJSON LineString
            stops: 'stops',                  // Array of stop IDs
        },

        heartbeat: {
            timestamp: 'timestamp',          // Server time
            clientId: 'client_id',           // Optional: client identifier
        },
    },

    // Transformers for messages (applied in messageHandler.ts)
    transformers: {
        normalizeLocationUpdate: (raw: any) => raw, // Placeholder, implemented in handler
        normalizeArrivalUpdate: (raw: any) => raw,
        normalizeRouteUpdate: (raw: any) => raw,
    },

    // Channel subscriptions (future: if backend supports)
    // Currently all messages broadcast to all clients
    channels: {
        locations: 'channel:bus:locations',
        arrivals: 'channel:arrivals',
        system: 'channel:system',
    },
}
