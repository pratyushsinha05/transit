/**
 * useGeofenceEvents Hook
 */

import { useState, useEffect, useCallback } from 'react';
import { useStore } from '../store';
import { fetchGeofenceEvents } from '../services/api/geofenceEvents';
import { logger } from '../services/logger';

export const useGeofenceEvents = (zoneId: string | null) => {
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const EMPTY_EVENTS: any[] = [];

    // Select events from store
    const events = useStore(state => {
        if (!zoneId) return EMPTY_EVENTS;
        return state.events.byZoneId.get(zoneId) || EMPTY_EVENTS;
    });

    const setEvents = useStore(state => state.setEvents);

    const getEvents = useCallback(async () => {
        if (!zoneId) return;

        setLoading(true);
        setError(null);
        try {
            const data = await fetchGeofenceEvents(zoneId);
            setEvents(zoneId, data);
        } catch (err: any) {
            const msg = err.message || 'Failed to fetch geofence events';
            setError(msg);
            logger.error('Error in useGeofenceEvents', { error: err });
        } finally {
            setLoading(false);
        }
    }, [zoneId, setEvents]);

    // Fetch on mount or zoneId change
    useEffect(() => {
        if (zoneId) {
            getEvents();
        }
    }, [zoneId, getEvents]);

    return {
        events,
        arrivals: events, // compatibility alias
        loading,
        error,
        refresh: getEvents,
    };
};

export const useArrivals = useGeofenceEvents;
