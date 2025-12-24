import type { StateCreator } from 'zustand';
import type { Arrival } from '../../types/domain';

export interface ArrivalsSlice {
    arrivals: {
        byStopId: Map<string, Arrival[]>;
    };
    setArrivals: (stopId: string, arrivals: Arrival[]) => void;
    updateArrival: (stopId: string, arrival: Arrival) => void;
}

export const createArrivalsSlice: StateCreator<
    ArrivalsSlice,
    [],
    [],
    ArrivalsSlice
> = (set) => ({
    arrivals: {
        byStopId: new Map(),
    },
    setArrivals: (stopId, arrivals) => set((state) => ({
        arrivals: {
            ...state.arrivals,
            byStopId: new Map(state.arrivals.byStopId).set(stopId, arrivals),
        }
    })),
    updateArrival: (stopId, arrival) => set((state) => {
        const currentArrivals = state.arrivals.byStopId.get(stopId) || [];
        const index = currentArrivals.findIndex(a => a.busId === arrival.busId);

        let newArrivals;
        if (index >= 0) {
            newArrivals = [...currentArrivals];
            newArrivals[index] = { ...newArrivals[index], ...arrival };
        } else {
            newArrivals = [...currentArrivals, arrival];
        }

        return {
            arrivals: {
                ...state.arrivals,
                byStopId: new Map(state.arrivals.byStopId).set(stopId, newArrivals),
            }
        };
    }),
});
