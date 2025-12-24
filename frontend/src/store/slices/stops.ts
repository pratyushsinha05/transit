import type { StateCreator } from 'zustand';
import type { Stop, Route } from '../../types/domain';

export interface StopsSlice {
    stops: Stop[];
    routes: Route[];
    setStops: (stops: Stop[]) => void;
    setRoutes: (routes: Route[]) => void;
}

export const createStopsSlice: StateCreator<
    StopsSlice,
    [],
    [],
    StopsSlice
> = (set) => ({
    stops: [],
    routes: [],
    setStops: (stops) => set({ stops }),
    setRoutes: (routes) => set({ routes }),
});
