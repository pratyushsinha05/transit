/**
 * Main Store
 * Combines all slices into a single state tree
 */

import { create } from 'zustand';
import { createBusLocationsSlice, type BusLocationsSlice } from './slices/busLocations';
import { createArrivalsSlice, type ArrivalsSlice } from './slices/arrivals';
import { createStopsSlice, type StopsSlice } from './slices/stops';
import { createConnectionSlice, type ConnectionSlice } from './slices/connection';
import { createUiSlice, type UiSlice } from './slices/ui';

export type StoreState = BusLocationsSlice & ArrivalsSlice & StopsSlice & ConnectionSlice & UiSlice;

export const useStore = create<StoreState>()((...a) => ({
    ...createBusLocationsSlice(...a),
    ...createArrivalsSlice(...a),
    ...createStopsSlice(...a),
    ...createConnectionSlice(...a),
    ...createUiSlice(...a),
}));
