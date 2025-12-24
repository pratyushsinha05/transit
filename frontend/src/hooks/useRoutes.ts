/**
 * useRoutes Hook
 */

import { useState, useEffect, useCallback } from 'react';
import { useStore } from '../store';
import { fetchRoutes } from '../services/api/routes';
import { logger } from '../services/logger';

export const useRoutes = () => {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const routes = useStore(state => state.routes);
    const setRoutes = useStore(state => state.setRoutes);

    const getRoutes = useCallback(async () => {
        if (routes.length > 0) return;

        setLoading(true);
        setError(null);
        try {
            const data = await fetchRoutes();
            setRoutes(data);
        } catch (err: any) {
            const msg = err.message || 'Failed to fetch routes';
            setError(msg);
            logger.error('Error in useRoutes', { error: err });
        } finally {
            setLoading(false);
        }
    }, [routes.length, setRoutes]);

    useEffect(() => {
        getRoutes();
    }, [getRoutes]);

    return {
        routes, loading, error, refresh: async () => {
            setLoading(true);
            try {
                const data = await fetchRoutes();
                setRoutes(data);
            } finally {
                setLoading(false);
            }
        }
    };
};
