/**
 * Geofence Events API Service
 */

import { apiClient } from './client';
import { API_CONFIG } from '../../config/apiConfig';
import { transformGeofenceEvent } from './transformers';
import { logger } from '../logger';

export const fetchGeofenceEvents = async (zoneId: string) => {
    try {
        const endpoint = API_CONFIG.endpoints.arrivals;

        const response = await apiClient.get(endpoint.path, {
            params: { stop_id: zoneId } // wire param remains stop_id
        });

        // Backend returns [] when no arrivals, but guard against null for safety
        const raw = response.data || [];
        if (!Array.isArray(raw)) {
            throw new Error('Expected array of geofence events from backend');
        }

        return raw.map(transformGeofenceEvent);
    } catch (error) {
        logger.error('Failed to fetch geofence events', { zoneId, error });
        throw error;
    }
};

export const fetchArrivals = fetchGeofenceEvents;
