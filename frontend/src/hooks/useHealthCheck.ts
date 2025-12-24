/**
 * useHealthCheck Hook
 * Polls backend health status
 */

import { useEffect } from 'react';
import { useStore } from '../store';
import { checkHealth } from '../services/api/health';
import { API_CONFIG } from '../config/apiConfig';

export const useHealthCheck = () => {
    const health = useStore((state) => state.health);
    const setHealth = useStore((state) => state.setHealth);

    useEffect(() => {
        // Initial health check
        checkHealthImmediate();

        // Check every X ms (default 30s)
        const intervalTime = API_CONFIG.endpoints.health.interval || 30000;
        const interval = setInterval(checkHealthImmediate, intervalTime);

        return () => clearInterval(interval);
    }, []);

    const checkHealthImmediate = async () => {
        const result = await checkHealth();

        setHealth({
            status: result.status ? 'healthy' : 'unhealthy',
            uptime: result.uptime,
            lastCheck: result.timestamp,
        });
    };

    return health;
};
