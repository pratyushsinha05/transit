import type { StateCreator } from 'zustand';

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
}

export const createUiSlice: StateCreator<
    UiSlice,
    [],
    [],
    UiSlice
> = (set) => ({
    notifications: [],
    selectedStopId: null,
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
    setSelectedStopId: (id) => set({ selectedStopId: id }),
});
