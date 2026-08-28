/**
 * Data Transformers
 * Convert Go backend responses to frontend types
 */

import { API_CONFIG } from '../../config/apiConfig';
import type { Arrival, Stop, BusLocation, Route } from '../../types/domain';
import { logger } from '../logger';

export const transformArrival = (raw: any): Arrival => {
    try {
        const schema = API_CONFIG.schemas.arrival;

        // Map backend services.ArrivalPrediction fields to the frontend type
        const arrival: Arrival = {
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
        if (!arrival.id || !arrival.deviceId) {
            throw new Error('Missing required fields: trip_id or device_id');
        }

        return arrival;
    } catch (error: any) {
        logger.error('Failed to transform arrival', { raw, error });
        throw new Error(`Invalid arrival format: ${error.message}`);
    }
};

export const transformStop = (raw: any): Stop => {
    try {
        const schema = API_CONFIG.schemas.stop;

        const stop: Stop = {
            id: raw[schema.id],
            name: raw[schema.name],
            lat: parseFloat(raw[schema.lat]),
            lng: parseFloat(raw[schema.lng]),
            address: raw[schema.address],
            h3Hex: raw[schema.h3_hex],  // Backend-calculated H3 hexagon
            arrivals: raw[schema.arrivals]?.map(transformArrival) || [],
        };

        // Validate coordinates
        if (isNaN(stop.lat) || isNaN(stop.lng)) {
            throw new Error('Invalid coordinates');
        }

        if (stop.lat < -90 || stop.lat > 90 || stop.lng < -180 || stop.lng > 180) {
            throw new Error('Coordinates out of bounds');
        }

        return stop;
    } catch (error) {
        logger.error('Failed to transform stop', { raw, error });
        throw error;
    }
};

export const transformBus = (raw: any): BusLocation => {
    try {
        const schema = API_CONFIG.schemas.bus;

        const bus: BusLocation = {
            id: raw[schema.id],
            routeId: raw[schema.routeId],
            lat: parseFloat(raw[schema.lat]),
            lng: parseFloat(raw[schema.lng]),
            speed: parseFloat(raw[schema.speed] || 0),
            heading: parseFloat(raw[schema.heading] || 0),
            lastUpdate: new Date(raw[schema.lastUpdate]),
            h3Hex: raw[schema.h3_hex],  // Backend-calculated geospatial hex
        };

        // Validate heading (0-360)
        if (bus.heading < 0 || bus.heading > 360) {
            bus.heading = 0; // Fallback
        }

        return bus;
    } catch (error) {
        logger.error('Failed to transform bus', { raw, error });
        throw error;
    }
};

export const transformRoute = (raw: any): Route => {
    try {
        const schema = API_CONFIG.schemas.route;

        const route: Route = {
            id: raw[schema.id],
            number: raw[schema.number],
            name: raw[schema.name],
            description: raw[schema.description],
            stops: raw[schema.stops] || [],
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
