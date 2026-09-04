/**
 * Data Transformers
 * Convert Go backend responses to frontend types
 */

import { API_CONFIG } from '../../config/apiConfig';
import type { GeofenceEvent, Zone, DeviceLocation, Route } from '../../types/domain';
import { logger } from '../logger';

export const transformGeofenceEvent = (raw: any): GeofenceEvent => {
    try {
        const schema = API_CONFIG.schemas.geofenceEvent;

        // Map backend services.GeofencePrediction fields to the frontend type
        const event: GeofenceEvent = {
            id: raw[schema.id],
            tripId: raw[schema.tripId],
            deviceId: raw[schema.deviceId],
            deviceName: raw[schema.deviceName],
            etaMinutes: parseInt(raw[schema.etaMinutes] ?? 0, 10),
            distanceKm: parseFloat(raw[schema.distanceKm] ?? 0),
            currentSpeed: parseFloat(raw[schema.currentSpeed] ?? 0),
            hexRes9: raw[schema.hexRes9] || undefined,
            isApproaching: Boolean(raw[schema.isApproaching]),
        };

        // Validation
        if (!event.id || !event.deviceId) {
            throw new Error('Missing required fields: trip_id or device_id');
        }

        return event;
    } catch (error: any) {
        logger.error('Failed to transform geofence event', { raw, error });
        throw new Error(`Invalid geofence event format: ${error.message}`);
    }
};

export const transformArrival = transformGeofenceEvent;

export const transformZone = (raw: any): Zone => {
    try {
        const schema = API_CONFIG.schemas.zone;

        const zone: Zone = {
            id: raw[schema.id],
            name: raw[schema.name],
            lat: parseFloat(raw[schema.lat]),
            lng: parseFloat(raw[schema.lng]),
            address: raw[schema.address],
            h3Hex: raw[schema.h3_hex],  // Backend-calculated H3 hexagon
            events: raw[schema.arrivals]?.map(transformGeofenceEvent) || [],
        };

        // Validate coordinates
        if (isNaN(zone.lat) || isNaN(zone.lng)) {
            throw new Error('Invalid coordinates');
        }

        if (zone.lat < -90 || zone.lat > 90 || zone.lng < -180 || zone.lng > 180) {
            throw new Error('Coordinates out of bounds');
        }

        return zone;
    } catch (error) {
        logger.error('Failed to transform zone', { raw, error });
        throw error;
    }
};

export const transformStop = transformZone;

export const transformDevice = (raw: any): DeviceLocation => {
    try {
        const schema = API_CONFIG.schemas.device;

        const device: DeviceLocation = {
            id: raw[schema.id],
            routeId: raw[schema.routeId],
            lat: parseFloat(raw[schema.lat]),
            lng: parseFloat(raw[schema.lng]),
            speed: parseFloat(raw[schema.speed] || 0),
            lastUpdate: new Date(raw[schema.lastUpdate]),
            h3Hex: raw[schema.h3_hex],  // Backend-calculated geospatial hex
        };

        return device;
    } catch (error) {
        logger.error('Failed to transform device', { raw, error });
        throw error;
    }
};

export const transformBus = transformDevice;

export const transformRoute = (raw: any): Route => {
    try {
        const schema = API_CONFIG.schemas.route;

        const route: Route = {
            id: raw[schema.id],
            number: raw[schema.number],
            name: raw[schema.name],
            description: raw[schema.description],
            zones: raw[schema.stops] || [],
            pattern: raw[schema.pattern], // GeoJSON LineString from backend
        };

        // Validate GeoJSON if present
        if (route.pattern && route.pattern.type !== 'LineString') {
            logger.warn('Invalid route pattern type', { route });
        }

        return route;
    } catch (error) {
        logger.error('Failed to transform route', { raw, error });
        throw error;
    }
};
