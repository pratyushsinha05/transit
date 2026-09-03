/**
 * WebSocket Message Handler
 * Routes and processes messages from backend
 */

import { WS_CONFIG } from '../../config/wsConfig';
import { useStore } from '../../store';
import { logger } from '../logger';
import type { BusLocation } from '../../types/domain';

export const handleMessage = (event: MessageEvent) => {
    try {
        const raw = JSON.parse(event.data);
        const messageType = raw.type;

        // Log for debugging (verbose, maybe limit to debug level)
        // logger.debug('WebSocket message received', { type: messageType });

        // Route based on backend message type
        switch (messageType) {
            case WS_CONFIG.messageTypes.connected:
                handleConnected(raw);
                break;

            case WS_CONFIG.messageTypes.location_update:
                handleLocationUpdate(raw);
                break;

            case WS_CONFIG.messageTypes.arrival_update:
                handleArrivalUpdate(raw);
                break;

            case WS_CONFIG.messageTypes.route_update:
                handleRouteUpdate(raw);
                break;

            case WS_CONFIG.messageTypes.heartbeat:
                handleHeartbeat(raw);
                break;

            case WS_CONFIG.messageTypes.error:
                handleError(raw);
                break;

            default:
                // logger.warn('Unknown message type from backend', { messageType });
                break;
        }
    } catch (error) {
        logger.error('Failed to parse WebSocket message', { error });
    }
};

const handleConnected = (_raw: any) => {
    logger.info('Connected to WebSocket server');

    // Update connection state via store action
    useStore.getState().setConnectionStatus('connected');
};

const handleLocationUpdate = (raw: any) => {
    const schema = WS_CONFIG.schemas.locationUpdate;

    try {
        // Backend hub.Message fields (CLAUDE.md Sec 7.2): type, device_id,
        // route_id, latitude, longitude, speed, accuracy, h3_hex, timestamp
        // (unix seconds). Flat envelope, no `data` wrapper.
        const deviceId = raw[schema.busId]; // schema.busId = 'device_id'
        if (!deviceId) return; // invalid message, skip silently

        const rawTs = raw[schema.lastUpdate]; // schema.lastUpdate = 'timestamp' (unix seconds)
        const lastUpdate = rawTs ? new Date(rawTs * 1000) : new Date();

        const location: BusLocation = {
            id: deviceId,
            routeId: raw[schema.routeId] || '', // server-resolved; '' if device has no active trip
            lat: parseFloat(raw[schema.lat]),
            lng: parseFloat(raw[schema.lng]),
            speed: parseFloat(raw[schema.speed] ?? 0),
            lastUpdate,
            h3Hex: raw[schema.h3_hex] || undefined,
        };

        if (isNaN(location.lat) || isNaN(location.lng)) return; // bad coordinates

        useStore.getState().updateBus(location);
    } catch (error) {
        logger.error('Failed to process location update', { error });
    }
};

// Dead: backend never emits this message type. Kept as scaffolding for
// Phase 4 (GEOFENCE_EVENT) or future message types. See hub/message.go.
const handleArrivalUpdate = (raw: any) => {
    const schema = WS_CONFIG.schemas.arrivalUpdate;

    try {
        const arrival = {
            id: `${raw[schema.stopId]}_${raw[schema.busId]}_${raw[schema.routeId]}`, // Generate a frontend ID if needed
            stopId: raw[schema.stopId],
            busId: raw[schema.busId],
            eta: parseInt(raw[schema.eta]),
            status: raw[schema.status],
            routeId: raw[schema.routeId],
            route: raw['route_number'] || raw[schema.routeId], // Fallback
            timestamp: new Date(raw[schema.timestamp]),
        };

        useStore.getState().updateArrival(arrival.stopId, arrival as any);
    } catch (error) {
        logger.error('Failed to process arrival update', { error });
    }
};

// Dead: backend never emits this message type. Kept as scaffolding.
const handleRouteUpdate = (raw: any) => {
    // Routes usually static, but if updates come:
    logger.info('Route update received', { routeId: raw.route_id });
    // Implement if needed to update route path live
};

// Dead: backend never emits JSON heartbeats (protocol-level ping/pong is used).
const handleHeartbeat = (_raw: any) => {
    useStore.getState().updateHeartbeat(new Date());
};

// Dead: backend never emits this message type. Kept as scaffolding.
const handleError = (raw: any) => {
    logger.error('Backend reported error', { message: raw.message });
    useStore.getState().addNotification({
        type: 'error',
        message: `Backend Error: ${raw.message}`
    });
};
