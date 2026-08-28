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
        location_update: 'LOCATION_UPDATE', // Must match Go const MsgTypeLocationUpdate

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
            // Exact JSON field names from Go hub.Message struct
            busId: 'device_id',              // hub.Message.DeviceID -> json:"device_id"
            lat: 'latitude',                 // hub.Message.Latitude -> json:"latitude"
            lng: 'longitude',                // hub.Message.Longitude -> json:"longitude"
            speed: 'speed',                  // hub.Message.Speed -> json:"speed"
            accuracy: 'accuracy',            // hub.Message.Accuracy -> json:"accuracy"
            lastUpdate: 'timestamp',         // hub.Message.Timestamp -> json:"timestamp" (unix int64)
            // Note: routeId, heading, h3_hex are NOT in hub.Message — not populated
            routeId: null,
            heading: null,
            h3_hex: null,
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
