/**
 * Routes API Service
 */

import { apiClient } from './client';
import { API_CONFIG } from '../../config/apiConfig';
import { transformRoute } from './transformers';
import { logger } from '../logger';

export const fetchRoutes = async () => {
    try {
        const endpoint = API_CONFIG.endpoints.routes;
        const response = await apiClient.get(endpoint.path);

        const routes = response.data.map(transformRoute);

        return routes;
    } catch (error) {
        logger.error('Failed to fetch routes', { error });
        throw error;
    }
};
