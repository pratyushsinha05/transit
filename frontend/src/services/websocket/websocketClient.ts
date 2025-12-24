/**
 * WebSocket Client
 * Manages connection lifecycle
 */

import { WS_CONFIG } from '../../config/wsConfig';
import { handleMessage } from './messageHandler';
import { useStore } from '../../store';
import { logger } from '../logger';

class WebSocketClient {
    private ws: WebSocket | null = null;
    private reconnectTimer: any = null;
    private shouldReconnect = true;

    connect() {
        if (this.ws?.readyState === WebSocket.OPEN) return;

        logger.info('Connecting to WebSocket...', { url: WS_CONFIG.url });
        useStore.getState().setConnectionStatus('connecting');

        try {
            this.ws = new WebSocket(WS_CONFIG.url);
            this.shouldReconnect = true;

            this.ws.onopen = () => {
                logger.info('WebSocket connected');
                useStore.getState().setConnectionStatus('connected');

                // Clear reconnect timer if any
                if (this.reconnectTimer) {
                    clearTimeout(this.reconnectTimer);
                    this.reconnectTimer = null;
                }
            };

            this.ws.onmessage = (event) => {
                handleMessage(event);
            };

            this.ws.onclose = () => {
                logger.warn('WebSocket disconnected');
                useStore.getState().setConnectionStatus('disconnected');

                if (this.shouldReconnect) {
                    this.scheduleReconnect();
                }
            };

            this.ws.onerror = (error) => {
                logger.error('WebSocket error', { error });
                // Error will trigger close usually
            };

        } catch (error) {
            logger.error('Failed to create WebSocket connection', { error });
            this.scheduleReconnect();
        }
    }

    disconnect() {
        this.shouldReconnect = false;
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }

    private scheduleReconnect() {
        if (this.reconnectTimer) return;

        const delay = WS_CONFIG.reconnectInterval;
        logger.info(`Scheduling reconnect in ${delay}ms`);

        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            this.connect();
        }, delay);
    }
}

export const wsClient = new WebSocketClient();
