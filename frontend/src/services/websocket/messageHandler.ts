/**
 * WebSocket Message Handler
 * Routes and processes messages from backend
 */

import { WS_CONFIG } from '../../config/wsConfig';
import { useStore } from '../../store';
import { logger } from '../logger';
import type { DeviceLocation } from '../../types/domain';

export const handleMessage = (event: MessageEvent) => {
    try {
        const raw = JSON.parse(event.data);
        const messageType = raw.type;

        // Log for debugging (verbose, maybe limit to debug level)
        // logger.debug('WebSocket message received', { type: messageType });

        // Route based on backend message type
        switch (messageType) {
            case WS_CONFIG.messageTypes.location_update:
                handleLocationUpdate(raw);
                break;

            default:
                // logger.warn('Unknown message type from backend', { messageType });
                break;
        }
    } catch (error) {
        logger.error('Failed to parse WebSocket message', { error });
    }
};

const handleLocationUpdate = (raw: any) => {
    const schema = WS_CONFIG.schemas.locationUpdate;

    try {
        // Backend hub.Message fields (CLAUDE.md Sec 7.2): type, device_id,
        // route_id, latitude, longitude, speed, accuracy, h3_hex, timestamp
        // (unix seconds). Flat envelope, no `data` wrapper.
        const deviceId = raw[schema.deviceId] || raw[schema.busId]; // schema.deviceId = 'device_id'
        if (!deviceId) return; // invalid message, skip silently

        const rawTs = raw[schema.lastUpdate]; // schema.lastUpdate = 'timestamp' (unix seconds)
        const lastUpdate = rawTs ? new Date(rawTs * 1000) : new Date();

        const location: DeviceLocation = {
            id: deviceId,
            routeId: raw[schema.routeId] || '', // server-resolved; '' if device has no active trip
            lat: parseFloat(raw[schema.lat]),
            lng: parseFloat(raw[schema.lng]),
            speed: parseFloat(raw[schema.speed] ?? 0),
            lastUpdate,
            h3Hex: raw[schema.h3_hex] || undefined,
        };

        if (isNaN(location.lat) || isNaN(location.lng)) return; // bad coordinates

        useStore.getState().updateDevice(location);
    } catch (error) {
        logger.error('Failed to process location update', { error });
    }
};
