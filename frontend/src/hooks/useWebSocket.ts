/**
 * useWebSocket Hook
 * Manages WebSocket connection lifecycle
 */

import { useEffect } from 'react';
import { wsClient } from '../services/websocket/websocketClient';
import { useStore } from '../store';

export const useWebSocket = () => {
    const connection = useStore((state) => state.connection);

    useEffect(() => {
        // Connect on mount
        wsClient.connect();

        return () => {
            // Disconnect on unmount
            wsClient.disconnect();
        };
    }, []);

    return connection;
};
