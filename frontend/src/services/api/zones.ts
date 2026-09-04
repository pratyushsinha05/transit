/**
 * Zones API Service
 */

import { apiClient } from './client';
import { API_CONFIG } from '../../config/apiConfig';
import { transformZone } from './transformers';
import { logger } from '../logger';

export const fetchZones = async (routeId?: string) => {
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

        const zones = response.data ? response.data.map(transformZone) : [];

        return zones;
    } catch (error) {
        logger.error('Failed to fetch zones', { error, routeId });
        throw error;
    }
};

export const fetchStops = fetchZones;
