/**
 * Connection Status
 * Dark HUD chip showing backend health and WebSocket connection state
 */

import { useHealthCheck } from '../../hooks/useHealthCheck';
import { useWebSocket } from '../../hooks/useWebSocket';

export const ConnectionStatus = () => {
    const health = useHealthCheck();
    const connection = useWebSocket();

    const isHealthy = health.status === 'healthy';
    const isConnected = connection.status === 'connected';

    return (
        <div
            className="flex items-center gap-3 text-[10px] font-mono px-3 py-1.5 border border-hud-border bg-hud-bg/90 backdrop-blur-sm"
            style={{
                boxShadow: '0 0 8px rgba(0,0,0,0.4), 0 0 1px rgba(0, 245, 212, 0.1)',
            }}
        >
            {/* WS Status */}
            <div className="flex items-center gap-1.5" title={`WebSocket: ${connection.status}`}>
                <span
                    className={`w-1.5 h-1.5 rounded-full ${isConnected ? 'bg-hud-accent animate-blink' : 'bg-hud-danger'}`}
                ></span>
                <span
                    className="tracking-hud uppercase font-bold"
                    style={{ color: isConnected ? '#00f5d4' : '#ff3860' }}
                >
                    {isConnected ? 'LINK' : 'NO LINK'}
                </span>
            </div>

            {/* Separator */}
            <div className="w-px h-3 bg-hud-border"></div>

            {/* Backend Health */}
            <div className="flex items-center gap-1.5" title={`Last check: ${health.lastCheck.toLocaleTimeString()}`}>
                <span
                    className={`w-1.5 h-1.5 rounded-full ${isHealthy ? 'bg-hud-accent' : 'bg-hud-warn'}`}
                ></span>
                <span className="tracking-hud uppercase text-hud-text-dim">
                    {health.uptime > 0 ? `${Math.floor(health.uptime / 60)}M UP` : 'CHK...'}
                </span>
            </div>
        </div>
    );
};
