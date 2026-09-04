import type { StateCreator } from 'zustand';
import type { Zone, Route } from '../../types/domain';

export interface ZonesSlice {
    zones: Zone[];
    routes: Route[];
    setZones: (zones: Zone[]) => void;
    setRoutes: (routes: Route[]) => void;
}

export const createZonesSlice: StateCreator<
    ZonesSlice,
    [],
    [],
    ZonesSlice
> = (set) => ({
    zones: [],
    routes: [],
    setZones: (zones) => set({ zones }),
    setRoutes: (routes) => set({ routes }),
});
