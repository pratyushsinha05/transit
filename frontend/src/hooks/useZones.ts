/**
 * useZones Hook
 */

import { useState, useEffect, useCallback } from 'react';
import { useStore } from '../store';
import { fetchZones } from '../services/api/zones';
import { logger } from '../services/logger';

export const useZones = () => {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const zones = useStore(state => state.zones);
    const setZones = useStore(state => state.setZones);

    const getZones = useCallback(async () => {
        // Prevent infinite loop if we already have data or if recently failed
        if (zones.length > 0) return;

        setLoading(true);
        setError(null);
        try {
            // Note: Backend requires route_id for /stops. 
            // If we want "all stops" we might need a different endpoint (e.g. nearby).
            // For now, we'll try to fetch zones for a default route or just handle the error.
            const data = await fetchZones('route-101'); // Hardcoding a default route for demo to avoid 400
            setZones(data);
        } catch (err: any) {
            const msg = err.message || 'Failed to fetch zones';
            setError(msg);
            // Don't log as error if it's just a 400 from missing params during init
            logger.warn('Error in useZones', { error: err });
        } finally {
            setLoading(false);
        }
    }, [zones.length, setZones]);

    useEffect(() => {
        getZones();
    }, [getZones]);

    return {
        zones,
        stops: zones, // compatibility alias
        loading,
        error,
        refresh: async () => {
            // Force refresh
            setLoading(true);
            try {
                const data = await fetchZones();
                setZones(data);
            } finally {
                setLoading(false);
            }
        }
    };
};

export const useStops = useZones;
