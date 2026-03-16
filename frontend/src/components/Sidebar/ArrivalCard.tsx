/**
 * Arrival Card
 * Displays a single arrival as a telemetry readout row
 */

import type { Arrival } from '../../types/domain';
import { formatDistanceToNow } from 'date-fns';

interface Props {
    arrival: Arrival;
}

const statusConfig: Record<string, { color: string; label: string; pulse?: boolean }> = {
    on_time: { color: '#00f5d4', label: 'ON TIME' },
    delayed: { color: '#ff6b35', label: 'DELAYED' },
    arriving: { color: '#00f5d4', label: 'ARRIVING', pulse: true },
    cancelled: { color: '#ff3860', label: 'CANCELLED' },
    scheduled: { color: '#576574', label: 'SCHEDULED' },
};

export const ArrivalCard = ({ arrival }: Props) => {
    const status = statusConfig[arrival.status] || statusConfig.scheduled;

    return (
        <div className="px-4 py-3 border-b border-hud-border/50 hover:bg-hud-panel/40 transition-colors">
            {/* Top row: Route + Status */}
            <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                    <span
                        className={`w-1.5 h-1.5 rounded-full ${status.pulse ? 'animate-blink' : ''}`}
                        style={{ backgroundColor: status.color }}
                    ></span>
                    <span className="text-[12px] font-bold text-hud-text-bright tracking-hud uppercase">
                        {arrival.route}
                    </span>
                </div>
                <span
                    className="text-[8px] font-bold tracking-hud-wide uppercase px-1.5 py-0.5 border rounded-sm"
                    style={{
                        color: status.color,
                        borderColor: `${status.color}33`,
                        backgroundColor: `${status.color}0d`,
                    }}
                >
                    {status.label}
                </span>
            </div>

            {/* Telemetry fields */}
            <div className="grid grid-cols-3 gap-2">
                <div>
                    <div className="hud-label">VEHICLE</div>
                    <div className="text-[10px] text-hud-text tracking-hud">{arrival.busId}</div>
                </div>
                <div>
                    <div className="hud-label">ETA</div>
                    <div className="text-[14px] font-bold" style={{ color: status.color }}>
                        {arrival.eta}<span className="text-[8px] text-hud-text-dim ml-0.5">MIN</span>
                    </div>
                </div>
                <div>
                    <div className="hud-label">UPDATED</div>
                    <div className="text-[9px] text-hud-text-dim tracking-wider">
                        {formatDistanceToNow(arrival.timestamp, { addSuffix: true }).toUpperCase()}
                    </div>
                </div>
            </div>
        </div>
    );
};
