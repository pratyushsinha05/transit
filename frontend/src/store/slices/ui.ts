import type { StateCreator } from 'zustand';
import type { RouteCreatorStop } from '../../types/domain';

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
    selectedStopId: string | null;
    setSelectedStopId: (id: string | null) => void;
    followedBusId: string | null;
    setFollowedBusId: (id: string | null) => void;

    // Route Creator
    routeCreatorMode: boolean;
    routeCreatorStops: RouteCreatorStop[];
    routeCreatorName: string;
    routeCreatorDescription: string;
    toggleRouteCreator: () => void;
    setRouteCreatorName: (name: string) => void;
    setRouteCreatorDescription: (desc: string) => void;
    addCreatorStop: (lat: number, lng: number) => void;
    removeCreatorStop: (tempId: string) => void;
    updateCreatorStopName: (tempId: string, name: string) => void;
    updateCreatorStopPosition: (tempId: string, lat: number, lng: number) => void;
    reorderCreatorStops: (fromIndex: number, toIndex: number) => void;
    clearCreatorStops: () => void;

    // Map Layers & Operations
    layerVisibility: { stops: boolean; buses: boolean; grid: boolean };
    hiddenRoutes: string[];
    toggleLayer: (layer: 'stops' | 'buses' | 'grid') => void;
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
    selectedStopId: null,
    followedBusId: null,

    // Route Creator defaults
    routeCreatorMode: false,
    routeCreatorStops: [],
    routeCreatorName: '',
    routeCreatorDescription: '',

    // Layer defaults
    layerVisibility: { stops: true, buses: true, grid: false },
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
    setSelectedStopId: (id) => set({ selectedStopId: id, followedBusId: null }),
    setFollowedBusId: (id) => set({ followedBusId: id, selectedStopId: null }),

    // Route Creator actions
    toggleRouteCreator: () => set((state) => ({
        routeCreatorMode: !state.routeCreatorMode,
        // Clear stops when exiting creator mode
        ...(state.routeCreatorMode ? {
            routeCreatorStops: [],
            routeCreatorName: '',
            routeCreatorDescription: '',
        } : {}),
    })),

    setRouteCreatorName: (name) => set({ routeCreatorName: name }),
    setRouteCreatorDescription: (desc) => set({ routeCreatorDescription: desc }),

    addCreatorStop: (lat, lng) => set((state) => ({
        routeCreatorStops: [
            ...state.routeCreatorStops,
            {
                tempId: generateTempId(),
                name: `STOP ${state.routeCreatorStops.length + 1}`,
                lat,
                lng,
            }
        ]
    })),

    removeCreatorStop: (tempId) => set((state) => ({
        routeCreatorStops: state.routeCreatorStops.filter(s => s.tempId !== tempId),
    })),

    updateCreatorStopName: (tempId, name) => set((state) => ({
        routeCreatorStops: state.routeCreatorStops.map(s =>
            s.tempId === tempId ? { ...s, name } : s
        ),
    })),

    updateCreatorStopPosition: (tempId, lat, lng) => set((state) => ({
        routeCreatorStops: state.routeCreatorStops.map(s =>
            s.tempId === tempId ? { ...s, lat, lng } : s
        ),
    })),

    reorderCreatorStops: (fromIndex, toIndex) => set((state) => {
        const stops = [...state.routeCreatorStops];
        const [moved] = stops.splice(fromIndex, 1);
        stops.splice(toIndex, 0, moved);
        return { routeCreatorStops: stops };
    }),

    clearCreatorStops: () => set({
        routeCreatorStops: [],
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
