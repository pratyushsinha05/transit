/**
 * Health Service
 * Check backend service status
 */

import { apiClient } from './client';
import { API_CONFIG } from '../../config/apiConfig';
import { logger } from '../logger';

export const checkHealth = async () => {
    try {
        const endpoint = API_CONFIG.endpoints.health;
        const response = await apiClient.get(endpoint.path);

        return {
            status: response.data.status === 'ok',
            uptime: response.data.uptime,
            timestamp: new Date(response.data.timestamp),
        };
    } catch (error) {
        logger.error('Health check failed', { error });
        return { status: false, uptime: 0, timestamp: new Date() };
    }
};
