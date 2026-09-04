import type { StateCreator } from 'zustand';
import type { RouteCreatorZone } from '../../types/domain';

export interface Notification {
    id: string;
    type: 'info' | 'success' | 'warning' | 'error';
    message: string;
    timestamp: Date;
}

export interface UiSlice {
    notifications: Notification[];
    addNotification: (notification: Omit<Notification, 'id' | 'timestamp'>) => void;
    removeNotification: (id: string) => void;
    selectedZoneId: string | null;
    setSelectedZoneId: (id: string | null) => void;
    followedDeviceId: string | null;
    setFollowedDeviceId: (id: string | null) => void;

    // Route Creator
    routeCreatorMode: boolean;
    routeCreatorZones: RouteCreatorZone[];
    routeCreatorName: string;
    routeCreatorDescription: string;
    toggleRouteCreator: () => void;
    setRouteCreatorName: (name: string) => void;
    setRouteCreatorDescription: (desc: string) => void;
    addCreatorZone: (lat: number, lng: number) => void;
    removeCreatorZone: (tempId: string) => void;
    updateCreatorZoneName: (tempId: string, name: string) => void;
    updateCreatorZonePosition: (tempId: string, lat: number, lng: number) => void;
    reorderCreatorZones: (fromIndex: number, toIndex: number) => void;
    clearCreatorZones: () => void;

    // Map Layers & Operations
    layerVisibility: { zones: boolean; devices: boolean };
    hiddenRoutes: string[];
    toggleLayer: (layer: 'zones' | 'devices') => void;
    toggleRouteVisibility: (routeId: string) => void;
}

const generateTempId = () => Math.random().toString(36).substring(2, 10);

export const createUiSlice: StateCreator<
    UiSlice,
    [],
    [],
    UiSlice
> = (set) => ({
    notifications: [],
    selectedZoneId: null,
    followedDeviceId: null,

    // Route Creator defaults
    routeCreatorMode: false,
    routeCreatorZones: [],
    routeCreatorName: '',
    routeCreatorDescription: '',

    // Layer defaults
    layerVisibility: { zones: true, devices: true },
    hiddenRoutes: [],

    addNotification: (notification) => set((state) => ({
        notifications: [
            ...state.notifications,
            {
                ...notification,
                id: Math.random().toString(36).substring(7),
                timestamp: new Date(),
            }
        ]
    })),
    removeNotification: (id) => set((state) => ({
        notifications: state.notifications.filter(n => n.id !== id),
    })),
    setSelectedZoneId: (id) => set({ selectedZoneId: id, followedDeviceId: null }),
    setFollowedDeviceId: (id) => set({ followedDeviceId: id, selectedZoneId: null }),

    // Route Creator actions
    toggleRouteCreator: () => set((state) => ({
        routeCreatorMode: !state.routeCreatorMode,
        // Clear zones when exiting creator mode
        ...(state.routeCreatorMode ? {
            routeCreatorZones: [],
            routeCreatorName: '',
            routeCreatorDescription: '',
        } : {}),
    })),

    setRouteCreatorName: (name) => set({ routeCreatorName: name }),
    setRouteCreatorDescription: (desc) => set({ routeCreatorDescription: desc }),

    addCreatorZone: (lat, lng) => set((state) => ({
        routeCreatorZones: [
            ...state.routeCreatorZones,
            {
                tempId: generateTempId(),
                name: `ZONE ${state.routeCreatorZones.length + 1}`,
                lat,
                lng,
            }
        ]
    })),

    removeCreatorZone: (tempId) => set((state) => ({
        routeCreatorZones: state.routeCreatorZones.filter(s => s.tempId !== tempId),
    })),

    updateCreatorZoneName: (tempId, name) => set((state) => ({
        routeCreatorZones: state.routeCreatorZones.map(s =>
            s.tempId === tempId ? { ...s, name } : s
        ),
    })),

    updateCreatorZonePosition: (tempId, lat, lng) => set((state) => ({
        routeCreatorZones: state.routeCreatorZones.map(s =>
            s.tempId === tempId ? { ...s, lat, lng } : s
        ),
    })),

    reorderCreatorZones: (fromIndex, toIndex) => set((state) => {
        const zones = [...state.routeCreatorZones];
        const [moved] = zones.splice(fromIndex, 1);
        zones.splice(toIndex, 0, moved);
        return { routeCreatorZones: zones };
    }),

    clearCreatorZones: () => set({
        routeCreatorZones: [],
        routeCreatorName: '',
        routeCreatorDescription: '',
    }),

    // Map Operations actions
    toggleLayer: (layer) => set((state) => ({
        layerVisibility: {
            ...state.layerVisibility,
            [layer]: !state.layerVisibility[layer]
        }
    })),

    toggleRouteVisibility: (routeId) => set((state) => ({
        hiddenRoutes: state.hiddenRoutes.includes(routeId)
            ? state.hiddenRoutes.filter(id => id !== routeId)
            : [...state.hiddenRoutes, routeId]
    })),
});
