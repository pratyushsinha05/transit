/**
 * Stops API Service
 */

import { apiClient } from './client';
import { API_CONFIG } from '../../config/apiConfig';
import { transformStop } from './transformers';
import { logger } from '../logger';

export const fetchStops = async (routeId?: string) => {
    try {
        const endpoint = API_CONFIG.endpoints.stops;
        // Backend requires route_id for now, or might support nearby.
        // To avoid 400, if no routeId is provided, we might need a different strategy.
        // For now, let's just support the parameter.

        const params: any = {};
        if (routeId) {
            params.route_id = routeId;
        }

        const response = await apiClient.get(endpoint.path, { params });

        const stops = response.data ? response.data.map(transformStop) : [];

        return stops;
    } catch (error) {
        logger.error('Failed to fetch stops', { error, routeId });
        throw error;
    }
};
