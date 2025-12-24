import type { StateCreator } from 'zustand';
import type { BusLocation } from '../../types/domain';

export interface BusLocationsSlice {
    buses: Map<string, BusLocation>;
    setBuses: (buses: Map<string, BusLocation>) => void;
    updateBus: (bus: BusLocation) => void;
}

export const createBusLocationsSlice: StateCreator<
    BusLocationsSlice,
    [],
    [],
    BusLocationsSlice
> = (set) => ({
    buses: new Map(),
    setBuses: (buses) => set({ buses }),
    updateBus: (bus) => set((state) => ({ buses: new Map(state.buses).set(bus.id, bus) })),
});
