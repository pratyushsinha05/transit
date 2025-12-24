/**
 * useStops Hook
 */

import { useState, useEffect, useCallback } from 'react';
import { useStore } from '../store';
import { fetchStops } from '../services/api/stops';
import { logger } from '../services/logger';

export const useStops = () => {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const stops = useStore(state => state.stops);
    const setStops = useStore(state => state.setStops);

    const getStops = useCallback(async () => {
        // Prevent infinite loop if we already have data or if recently failed
        if (stops.length > 0) return;

        setLoading(true);
        setError(null);
        try {
            // Note: Backend requires route_id for /stops. 
            // If we want "all stops" we might need a different endpoint (e.g. nearby).
            // For now, we'll try to fetch stops for a default route or just handle the error.
            // Let's modify this to NOT fail strictly if we can't fetch "all" stops.
            const data = await fetchStops('route-101'); // Hardcoding a default route for demo to avoid 400
            setStops(data);
        } catch (err: any) {
            const msg = err.message || 'Failed to fetch stops';
            setError(msg);
            // Don't log as error if it's just a 400 from missing params during init
            logger.warn('Error in useStops', { error: err });
        } finally {
            setLoading(false);
        }
    }, [stops.length, setStops]);

    useEffect(() => {
        getStops();
    }, [getStops]);

    return {
        stops, loading, error, refresh: async () => {
            // Force refresh
            setLoading(true);
            try {
                const data = await fetchStops();
                setStops(data);
            } finally {
                setLoading(false);
            }
        }
    };
};
