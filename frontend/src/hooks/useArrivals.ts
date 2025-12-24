/**
 * useArrivals Hook
 */

import { useState, useEffect, useCallback } from 'react';
import { useStore } from '../store';
import { fetchArrivals } from '../services/api/arrivals';
import { logger } from '../services/logger';

export const useArrivals = (stopId: string | null) => {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const EMPTY_ARRIVALS: any[] = [];

    // Select arrivals from store
    const arrivals = useStore(state => {
        if (!stopId) return EMPTY_ARRIVALS;
        return state.arrivals.byStopId.get(stopId) || EMPTY_ARRIVALS;
    });

    const setArrivals = useStore(state => state.setArrivals);

    const getArrivals = useCallback(async () => {
        if (!stopId) return;

        setLoading(true);
        setError(null);
        try {
            const data = await fetchArrivals(stopId);
            setArrivals(stopId, data);
        } catch (err: any) {
            const msg = err.message || 'Failed to fetch arrivals';
            setError(msg);
            logger.error('Error in useArrivals', { error: err });
        } finally {
            setLoading(false);
        }
    }, [stopId, setArrivals]);

    // Fetch on mount or stopId change
    useEffect(() => {
        if (stopId) {
            getArrivals();
        }
    }, [stopId, getArrivals]);

    return { arrivals, loading, error, refresh: getArrivals };
};
