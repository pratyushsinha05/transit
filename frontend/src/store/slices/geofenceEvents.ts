import type { StateCreator } from 'zustand';
import type { GeofenceEvent } from '../../types/domain';

export interface GeofenceEventsSlice {
    events: {
        byZoneId: Map<string, GeofenceEvent[]>;
    };
    setEvents: (zoneId: string, events: GeofenceEvent[]) => void;
    updateEvent: (zoneId: string, event: GeofenceEvent) => void;
}

export const createGeofenceEventsSlice: StateCreator<
    GeofenceEventsSlice,
    [],
    [],
    GeofenceEventsSlice
> = (set) => ({
    events: {
        byZoneId: new Map(),
    },
    setEvents: (zoneId, events) => set((state) => ({
        events: {
            ...state.events,
            byZoneId: new Map(state.events.byZoneId).set(zoneId, events),
        }
    })),
    updateEvent: (zoneId, event) => set((state) => {
        const currentEvents = state.events.byZoneId.get(zoneId) || [];
        const index = currentEvents.findIndex(e => e.tripId === event.tripId);

        let newEvents;
        if (index >= 0) {
            newEvents = [...currentEvents];
            newEvents[index] = { ...newEvents[index], ...event };
        } else {
            newEvents = [...currentEvents, event];
        }

        return {
            events: {
                ...state.events,
                byZoneId: new Map(state.events.byZoneId).set(zoneId, newEvents),
            }
        };
    }),
});
