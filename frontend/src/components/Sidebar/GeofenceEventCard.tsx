/**
 * Geofence Event Card
 * Displays a single predicted geofence event as a telemetry readout row.
 *
 * Driven entirely by backend services.GeofencePrediction fields -- there is
 * no route/status/timestamp on this response, so the status chip reflects
 * the real is_approaching flag (k-ring membership) rather than an invented
 * enum.
 */

import type { GeofenceEvent } from '../../types/domain';

interface Props {
    event?: GeofenceEvent;
    arrival?: GeofenceEvent; // compatibility prop
}

export const GeofenceEventCard = ({ event, arrival }: Props) => {
    const item = event || arrival!;
    const statusColor = item.isApproaching ? '#00f5d4' : '#576574';
    const statusLabel = item.isApproaching ? 'APPROACHING' : 'EN ROUTE';

    return (
        <div className="px-4 py-3 border-b border-hud-border/50 hover:bg-hud-panel/40 transition-colors">
            {/* Top row: Vehicle + Status */}
            <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                    <span
                        className={`w-1.5 h-1.5 rounded-full ${item.isApproaching ? 'animate-blink' : ''}`}
                        style={{ backgroundColor: statusColor }}
                    ></span>
                    <span className="text-[12px] font-bold text-hud-text-bright tracking-hud uppercase">
                        {item.deviceName || item.deviceId}
                    </span>
                </div>
                <span
                    className="text-[8px] font-bold tracking-hud-wide uppercase px-1.5 py-0.5 border rounded-sm"
                    style={{
                        color: statusColor,
                        borderColor: `${statusColor}33`,
                        backgroundColor: `${statusColor}0d`,
                    }}
                >
                    {statusLabel}
                </span>
            </div>

            {/* Telemetry fields */}
            <div className="grid grid-cols-3 gap-2">
                <div>
                    <div className="hud-label">ETA</div>
                    <div className="text-[14px] font-bold" style={{ color: statusColor }}>
                        {item.etaMinutes}<span className="text-[8px] text-hud-text-dim ml-0.5">MIN</span>
                    </div>
                </div>
                <div>
                    <div className="hud-label">DISTANCE</div>
                    <div className="text-[10px] text-hud-text tracking-hud">
                        {item.distanceKm.toFixed(2)}<span className="text-[8px] text-hud-text-dim ml-0.5">KM</span>
                    </div>
                </div>
                <div>
                    <div className="hud-label">SPEED</div>
                    <div className="text-[10px] text-hud-text tracking-hud">
                        {Math.round(item.currentSpeed)}<span className="text-[8px] text-hud-text-dim ml-0.5">KM/H</span>
                    </div>
                </div>
            </div>
        </div>
    );
};

export const ArrivalCard = GeofenceEventCard;
