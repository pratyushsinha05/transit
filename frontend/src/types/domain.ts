/**
 * Domain Types
 * Core data structures used throughout the application
 */

// Mirrors backend services.ArrivalPrediction (GET /api/arrivals). There is no
// route/status/timestamp on this response -- the backend doesn't have that
// data on this path. Do not add fields here that transformArrival can't
// populate from a real response field.
export interface Arrival {
    id: string; // = tripId; every active trip has exactly one prediction
    tripId: string;
    deviceId: string;
    deviceName: string;
    etaMinutes: number;
    distanceKm: number;
    currentSpeed: number; // km/h
    hexRes9?: string;
    isApproaching: boolean;
}

export interface Stop {
    id: string;
    name: string;
    lat: number;
    lng: number;
    address?: string;
    h3Hex?: string; // H3 hexagon ID from backend
    arrivals: Arrival[];
}

export interface Route {
    id: string;
    number: string;
    name: string;
    description?: string;
    stops: string[]; // Array of Stop IDs
    pattern?: GeoJSON.LineString; // GeoJSON path
}

export interface BusLocation {
    id: string;
    routeId: string;
    lat: number;
    lng: number;
    speed: number; // km/h
    lastUpdate: Date;
    h3Hex?: string; // H3 hexagon ID from backend
}

export interface HealthStatus {
    status: 'healthy' | 'unhealthy' | 'error' | 'unknown';
    uptime: number; // seconds
    lastCheck: Date;
}

export interface ConnectionState {
    status: 'connected' | 'disconnected' | 'connecting' | 'reconnecting';
    lastConnected?: Date;
    lastHeartbeat?: Date;
    reconnectAttempts: number;
}

// ── Route Creator Types ──

export interface RouteCreatorStop {
    tempId: string;
    name: string;
    lat: number;
    lng: number;
}

export interface CreateRoutePayload {
    name: string;
    description: string;
    stops: {
        name: string;
        latitude: number;
        longitude: number;
    }[];
}

export interface CreateRouteResponse {
    id: string;
    name: string;
    description: string;
    stop_count: number;
    stop_ids: string[];
}
