/**
 * Connection Status
 * Shows backend health and WebSocket connection state
 */

import { useHealthCheck } from '../../hooks/useHealthCheck';
import { useWebSocket } from '../../hooks/useWebSocket';
import clsx from 'clsx';
import { Wifi, WifiOff, Activity } from 'lucide-react';

export const ConnectionStatus = () => {
    const health = useHealthCheck();
    const connection = useWebSocket();

    const isHealthy = health.status === 'healthy';
    const isConnected = connection.status === 'connected';

    return (
        <div className="flex items-center gap-4 text-sm bg-white/90 backdrop-blur px-3 py-1.5 rounded-full shadow-sm border border-gray-200">
            {/* WS Status */}
            <div className="flex items-center gap-1.5" title={`WebSocket: ${connection.status}`}>
                {isConnected ? (
                    <Wifi className="w-4 h-4 text-green-500" />
                ) : (
                    <WifiOff className="w-4 h-4 text-red-500" />
                )}
                <span className={clsx('hidden md:inline', isConnected ? 'text-green-700' : 'text-red-600')}>
                    {isConnected ? 'Real-time' : 'Offline'}
                </span>
            </div>

            {/* Backend Health */}
            <div className="flex items-center gap-1.5 border-l pl-4" title={`Last check: ${health.lastCheck.toLocaleTimeString()}`}>
                <Activity className={clsx('w-4 h-4', isHealthy ? 'text-green-500' : 'text-yellow-500')} />
                <span className="hidden md:inline font-mono text-xs text-gray-500">
                    {health.uptime > 0 ? `${Math.floor(health.uptime / 60)}m uptime` : 'Checking...'}
                </span>
            </div>
        </div>
    );
};
