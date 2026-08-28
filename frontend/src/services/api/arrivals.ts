/**
 * Arrivals API Service
 */

import { apiClient } from './client';
import { API_CONFIG } from '../../config/apiConfig';
import { transformArrival } from './transformers';
import { logger } from '../logger';

export const fetchArrivals = async (stopId: string) => {
    try {
        const endpoint = API_CONFIG.endpoints.arrivals;

        const response = await apiClient.get(endpoint.path, {
            params: { stop_id: stopId }
        });

        // Backend returns [] when no arrivals, but guard against null for safety
        const raw = response.data || [];
        if (!Array.isArray(raw)) {
            throw new Error('Expected array of arrivals from backend');
        }

        return raw.map(transformArrival);
    } catch (error) {
        logger.error('Failed to fetch arrivals', { stopId, error });
        throw error;
    }
};
