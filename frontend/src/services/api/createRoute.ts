/**
 * Create Route API Service
 * POST /api/routes — creates a route with stops
 */

import { apiClient } from './client';
import type { CreateRoutePayload, CreateRouteResponse } from '../../types/domain';
import { logger } from '../logger';

export const createRoute = async (payload: CreateRoutePayload): Promise<CreateRouteResponse> => {
    try {
        const response = await apiClient.post('/api/routes', payload);
        return response.data;
    } catch (error) {
        logger.error('Failed to create route', { error, payload });
        throw error;
    }
};
