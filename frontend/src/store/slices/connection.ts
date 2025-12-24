import type { StateCreator } from 'zustand';
import type { ConnectionState, HealthStatus } from '../../types/domain';

export interface ConnectionSlice {
    connection: ConnectionState;
    health: HealthStatus;
    setConnectionStatus: (status: ConnectionState['status']) => void;
    updateHeartbeat: (timestamp: Date) => void;
    setHealth: (health: HealthStatus) => void;
}

export const createConnectionSlice: StateCreator<
    ConnectionSlice,
    [],
    [],
    ConnectionSlice
> = (set) => ({
    connection: {
        status: 'disconnected',
        reconnectAttempts: 0,
    },
    health: {
        status: 'unknown',
        uptime: 0,
        lastCheck: new Date(),
    },
    setConnectionStatus: (status) => set((state) => ({
        connection: {
            ...state.connection,
            status,
            lastConnected: status === 'connected' ? new Date() : state.connection.lastConnected,
        }
    })),
    updateHeartbeat: (timestamp) => set((state) => ({
        connection: {
            ...state.connection,
            lastHeartbeat: timestamp,
        }
    })),
    setHealth: (health) => set({ health }),
});
