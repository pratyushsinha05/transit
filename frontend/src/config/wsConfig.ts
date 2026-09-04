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
        // Real-time location updates
        location_update: 'LOCATION_UPDATE', // Must match Go const MsgTypeLocationUpdate
    },

    // Message schema mappings (what backend sends)
    schemas: {
        locationUpdate: {
            // Exact JSON field names from Go hub.Message struct (CLAUDE.md
            // Sec 7.2). Flat envelope, no `data` wrapper.
            deviceId: 'device_id',            // hub.Message.DeviceID -> json:"device_id"
            busId: 'device_id',               // compatibility alias
            routeId: 'route_id',              // hub.Message.RouteID -> json:"route_id" (server-resolved, may be "")
            lat: 'latitude',                  // hub.Message.Latitude -> json:"latitude"
            lng: 'longitude',                 // hub.Message.Longitude -> json:"longitude"
            speed: 'speed',                   // hub.Message.Speed -> json:"speed"
            accuracy: 'accuracy',             // hub.Message.Accuracy -> json:"accuracy"
            h3_hex: 'h3_hex',                 // hub.Message.H3Hex -> json:"h3_hex"
            lastUpdate: 'timestamp',          // hub.Message.Timestamp -> json:"timestamp" (unix seconds)
        },

        arrivalUpdate: {
            zoneId: 'stop_id',                // Which zone
            stopId: 'stop_id',
            deviceId: 'bus_id',               // Which device
            busId: 'bus_id',
            eta: 'eta',                       // Minutes until arrival
            status: 'status',                 // 'on_time', 'delayed', 'arriving'
            routeId: 'route_id',              // Route number
            timestamp: 'timestamp',           // When update generated
        },

        routeUpdate: {
            routeId: 'route_id',
            zones: 'stops',                   // Array of zone IDs
            stops: 'stops',
        },

        heartbeat: {
            timestamp: 'timestamp',           // Server time
            clientId: 'client_id',            // Optional: client identifier
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
};
