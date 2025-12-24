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

        // Transform responses using actual backend schema
        const arrivals = response.data.map(transformArrival);

        // Validate
        if (!Array.isArray(arrivals)) {
            throw new Error('Expected array of arrivals');
        }

        return arrivals;
    } catch (error) {
        logger.error('Failed to fetch arrivals', { stopId, error });
        throw error;
    }
};
