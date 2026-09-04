/**
 * Geofence Event List
 * Dark HUD telemetry panel listing upcoming geofence events for a zone
 */

import { useGeofenceEvents } from '../../hooks/useGeofenceEvents';
import { GeofenceEventCard } from './GeofenceEventCard';
import { useStore } from '../../store';

interface Props {
    zoneId?: string;
    stopId?: string; // compatibility prop
}

export const GeofenceEventList = ({ zoneId, stopId }: Props) => {
    const targetId = zoneId || stopId || '';
    const { events, loading, error, refresh } = useGeofenceEvents(targetId);
    const zones = useStore(state => state.zones);
    const zone = zones.find(z => z.id === targetId);

    if (loading && events.length === 0) {
        return (
            <div className="p-6 text-center">
                <div className="w-4 h-4 border border-hud-accent/40 border-t-hud-accent rounded-full animate-spin mx-auto mb-3"></div>
                <div className="text-[10px] text-hud-text-dim tracking-hud uppercase">
                    ACQUIRING DATA...
                </div>
            </div>
        );
    }

    if (error) {
        return (
            <div className="p-6 text-center">
                <div className="text-[10px] text-hud-danger tracking-hud uppercase mb-2">
                    ⚠ TELEMETRY ERROR
                </div>
                <button
                    onClick={refresh}
                    className="text-[10px] text-hud-accent tracking-hud uppercase hover:underline"
                >
                    RETRY ACQUISITION
                </button>
            </div>
        );
    }

    return (
        <div className="flex flex-col h-full">
            {/* Zone header */}
            <div className="p-4 border-b border-hud-border bg-hud-panel/40">
                <div className="hud-label mb-1">ZONE DESIGNATION</div>
                <h2 className="text-[13px] font-bold text-hud-accent tracking-hud uppercase leading-tight">
                    {zone?.name || 'UNKNOWN'}
                </h2>
                {zone?.address && (
                    <p className="text-[9px] text-hud-text-dim tracking-hud uppercase mt-1">{zone.address}</p>
                )}
                {zone?.h3Hex && (
                    <div className="mt-2 flex items-center gap-1.5">
                        <span className="hud-label">H3 HEX</span>
                        <code className="text-[9px] text-hud-accent/60 tracking-wider">{zone.h3Hex}</code>
                    </div>
                )}
            </div>

            {/* Events section header */}
            <div className="px-4 py-2 border-b border-hud-border bg-hud-panel/20">
                <div className="flex items-center justify-between">
                    <span className="text-[9px] font-bold text-hud-text-dim tracking-hud-wide uppercase">
                        APPROACH VECTORS // {events.length} ACTIVE
                    </span>
                    <button
                        onClick={refresh}
                        className="text-[8px] text-hud-accent/50 tracking-hud uppercase hover:text-hud-accent transition-colors"
                    >
                        REFRESH
                    </button>
                </div>
            </div>

            {/* Events list */}
            <div className="flex-1 overflow-y-auto">
                {events.length === 0 ? (
                    <div className="text-center py-10">
                        <div className="text-[10px] text-hud-text-dim tracking-hud uppercase">
                            NO ACTIVE APPROACH VECTORS
                        </div>
                    </div>
                ) : (
                    events.map(event => (
                        <GeofenceEventCard key={event.id} event={event} />
                    ))
                )}
            </div>
        </div>
    );
};

export const ArrivalsList = GeofenceEventList;
