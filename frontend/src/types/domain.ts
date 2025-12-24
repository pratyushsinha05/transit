/**
 * Domain Types
 * Core data structures used throughout the application
 */

export type ArrivalStatus = 'on_time' | 'delayed' | 'arriving' | 'cancelled' | 'scheduled';

export interface Arrival {
    id: string;
    busId: string;
    eta: number; // Minutes
    status: ArrivalStatus;
    route: string; // Route number/name
    routeId: string;
    timestamp: Date;
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
    heading: number; // degrees 0-360
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
