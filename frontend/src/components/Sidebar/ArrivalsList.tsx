/**
 * Arrivals List
 * Dark HUD telemetry panel listing upcoming arrivals for a stop
 */

import { useArrivals } from '../../hooks/useArrivals';
import { ArrivalCard } from './ArrivalCard';
import { useStore } from '../../store';

interface Props {
    stopId: string;
}

export const ArrivalsList = ({ stopId }: Props) => {
    const { arrivals, loading, error, refresh } = useArrivals(stopId);
    const stops = useStore(state => state.stops);
    const stop = stops.find(s => s.id === stopId);

    if (loading && arrivals.length === 0) {
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
            {/* Stop header */}
            <div className="p-4 border-b border-hud-border bg-hud-panel/40">
                <div className="hud-label mb-1">STOP DESIGNATION</div>
                <h2 className="text-[13px] font-bold text-hud-accent tracking-hud uppercase leading-tight">
                    {stop?.name || 'UNKNOWN'}
                </h2>
                {stop?.address && (
                    <p className="text-[9px] text-hud-text-dim tracking-hud uppercase mt-1">{stop.address}</p>
                )}
                {stop?.h3Hex && (
                    <div className="mt-2 flex items-center gap-1.5">
                        <span className="hud-label">H3 HEX</span>
                        <code className="text-[9px] text-hud-accent/60 tracking-wider">{stop.h3Hex}</code>
                    </div>
                )}
            </div>

            {/* Arrivals section header */}
            <div className="px-4 py-2 border-b border-hud-border bg-hud-panel/20">
                <div className="flex items-center justify-between">
                    <span className="text-[9px] font-bold text-hud-text-dim tracking-hud-wide uppercase">
                        APPROACH VECTORS // {arrivals.length} ACTIVE
                    </span>
                    <button
                        onClick={refresh}
                        className="text-[8px] text-hud-accent/50 tracking-hud uppercase hover:text-hud-accent transition-colors"
                    >
                        REFRESH
                    </button>
                </div>
            </div>

            {/* Arrivals list */}
            <div className="flex-1 overflow-y-auto">
                {arrivals.length === 0 ? (
                    <div className="text-center py-10">
                        <div className="text-[10px] text-hud-text-dim tracking-hud uppercase">
                            NO ACTIVE APPROACH VECTORS
                        </div>
                    </div>
                ) : (
                    arrivals.map(arrival => (
                        <ArrivalCard key={arrival.id} arrival={arrival} />
                    ))
                )}
            </div>
        </div>
    );
};
