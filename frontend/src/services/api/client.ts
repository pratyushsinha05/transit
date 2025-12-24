/**
 * API Client
 * Configured Axios instance for communicating with the Go backend
 */

import axios from 'axios';
import { API_CONFIG } from '../../config/apiConfig';
import { logger } from '../logger';

// Extend Axios config to support metadata
declare module 'axios' {
    export interface InternalAxiosRequestConfig {
        metadata?: {
            startTime: number;
        };
    }
}

export const apiClient = axios.create({
    baseURL: API_CONFIG.BASE_URL,
    timeout: API_CONFIG.timeouts.default,
    headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
    },
});

// Request interceptor
apiClient.interceptors.request.use((config) => {
    // Add request timestamp for debugging
    config.metadata = { startTime: Date.now() };

    // Add API version header (for future versioning)
    // config.headers['X-API-Version'] = '1';

    return config;
});

// Response interceptor with error handling
apiClient.interceptors.response.use(
    (response) => {
        // Log response time
        if (response.config.metadata) {
            const duration = Date.now() - response.config.metadata.startTime;
            logger.debug('API response', {
                url: response.config.url,
                status: response.status,
                duration
            });
        }

        return response;
    },
    (error) => {
        // Handle specific error codes from Go backend
        if (error.response) {
            const status = error.response.status;
            const data = error.response.data;

            switch (status) {
                case 400:
                    logger.error('Bad request', { url: error.config?.url, data });
                    break;
                case 404:
                    logger.warn('Not found', { url: error.config?.url });
                    break;
                case 429:
                    logger.warn('Rate limited, will retry');
                    break;
                case 500:
                    logger.error('Server error', { url: error.config?.url });
                    break;
                case 503:
                    logger.error('Service unavailable (database/redis down?)');
                    break;
            }
        } else if (error.request) {
            logger.error('No response from server', { url: error.config?.url });
        }

        return Promise.reject(error);
    }
);
