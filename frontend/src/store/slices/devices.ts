import type { StateCreator } from 'zustand';
import type { DeviceLocation } from '../../types/domain';

export interface DevicesSlice {
    devices: Map<string, DeviceLocation>;
    setDevices: (devices: Map<string, DeviceLocation>) => void;
    updateDevice: (device: DeviceLocation) => void;
}

export const createDevicesSlice: StateCreator<
    DevicesSlice,
    [],
    [],
    DevicesSlice
> = (set) => ({
    devices: new Map(),
    setDevices: (devices) => set({ devices }),
    updateDevice: (device) => set((state) => ({ devices: new Map(state.devices).set(device.id, device) })),
});
