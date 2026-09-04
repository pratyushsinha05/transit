/**
 * Main Store
 * Combines all slices into a single state tree
 */

import { create } from 'zustand';
import { createDevicesSlice, type DevicesSlice } from './slices/devices';
import { createGeofenceEventsSlice, type GeofenceEventsSlice } from './slices/geofenceEvents';
import { createZonesSlice, type ZonesSlice } from './slices/zones';
import { createConnectionSlice, type ConnectionSlice } from './slices/connection';
import { createUiSlice, type UiSlice } from './slices/ui';

export type StoreState = DevicesSlice & GeofenceEventsSlice & ZonesSlice & ConnectionSlice & UiSlice;

export const useStore = create<StoreState>()((...a) => ({
    ...createDevicesSlice(...a),
    ...createGeofenceEventsSlice(...a),
    ...createZonesSlice(...a),
    ...createConnectionSlice(...a),
    ...createUiSlice(...a),
}));
