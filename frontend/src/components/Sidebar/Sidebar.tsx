/**
 * Sidebar
 * NASA Mission Control telemetry panel with route creator mode
 */

import { useStore } from '../../store';
import { ArrivalsList } from './ArrivalsList';
import { RouteCreatorPanel } from './RouteCreatorPanel';
import { useStops } from '../../hooks/useStops';

export const Sidebar = () => {
    const selectedStopId = useStore(state => state.selectedStopId);
    const setSelectedStopId = useStore(state => state.setSelectedStopId);
    const { stops } = useStops();
    const connection = useStore(state => state.connection);
    const buses = useStore(state => state.buses);
    const routeCreatorMode = useStore(state => state.routeCreatorMode);
    const toggleRouteCreator = useStore(state => state.toggleRouteCreator);

    const isConnected = connection.status === 'connected';
    const busCount = buses.size;

    return (
        <div className="h-full flex flex-col bg-hud-bg w-80 relative z-[1000] border-r border-hud-border"
             style={{ boxShadow: '1px 0 8px rgba(0,0,0,0.5), 1px 0 1px rgba(0, 245, 212, 0.05)' }}>
            {/* Scanline + Noise overlays */}
            <div className="scanline-overlay"></div>
            <div className="noise-overlay"></div>

            {/* ── Header ── */}
            <div className="relative z-20 p-4 border-b border-hud-border">
                {/* Status indicator */}
                <div className="flex items-center gap-2 mb-3">
                    <span className={`w-1.5 h-1.5 rounded-full ${isConnected ? 'bg-hud-accent animate-blink' : 'bg-hud-danger'}`}></span>
                    <span className="text-[9px] font-bold tracking-hud-wide uppercase"
                          style={{ color: isConnected ? '#00f5d4' : '#ff3860' }}>
                        {isConnected ? 'TRACKING ACTIVE' : 'SIGNAL LOST'}
                    </span>
                </div>

                {/* Title */}
                <h1 className="text-lg font-bold text-hud-accent tracking-hud-wide leading-none mb-1">
                    LATITUDEX
                </h1>
                <p className="text-[9px] text-hud-text-dim tracking-hud-wide uppercase">
                    LIVE TRANSIT OPERATIONS
                </p>

                {/* Quick stats */}
                <div className="flex gap-4 mt-3 pt-3 border-t border-hud-border">
                    <div>
                        <div className="hud-label">VEHICLES</div>
                        <div className="hud-value text-[16px]">{busCount}</div>
                    </div>
                    <div>
                        <div className="hud-label">STOPS</div>
                        <div className="hud-value text-[16px]">{stops.length}</div>
                    </div>
                    <div>
                        <div className="hud-label">STATUS</div>
                        <div className="text-[11px] mt-0.5" style={{ color: isConnected ? '#00f5d4' : '#ff3860' }}>
                            {isConnected ? 'NOMINAL' : 'OFFLINE'}
                        </div>
                    </div>
                </div>

                {/* Mode Tabs */}
                <div className="mt-4 flex border border-hud-border">
                    <button
                        onClick={() => {
                            if (routeCreatorMode) toggleRouteCreator();
                        }}
                        className={`flex-1 py-1.5 text-[9px] font-bold tracking-hud-wide uppercase transition-all ${!routeCreatorMode ? 'bg-hud-accent/20 text-hud-bright border-b-2 border-hud-accent' : 'bg-transparent text-hud-text-dim hover:bg-hud-panel/30 hover:text-hud-text border-b-2 border-transparent'}`}
                    >
                        TRACKING
                    </button>
                    <button
                        onClick={() => {
                            if (!routeCreatorMode) {
                                setSelectedStopId(null);
                                toggleRouteCreator();
                            }
                        }}
                        className={`flex-1 py-1.5 text-[9px] font-bold tracking-hud-wide uppercase transition-all border-l border-r border-hud-border ${routeCreatorMode ? 'bg-hud-warn/20 text-hud-warn border-b-2 border-hud-warn' : 'bg-transparent text-hud-text-dim hover:bg-hud-panel/30 hover:text-hud-text border-b-2 border-transparent'}`}
                    >
                        BUILDER
                    </button>
                    <button
                        disabled
                        className="flex-1 py-1.5 text-[9px] font-bold tracking-hud-wide uppercase bg-transparent text-hud-text-dim/30 border-b-2 border-transparent cursor-not-allowed"
                        title="Alert Configuration Module (Offline)"
                    >
                        ALERTS
                    </button>
                </div>

                {/* Search (only in tracking mode) */}
                {!routeCreatorMode && (
                    <div className="mt-3">
                        <input
                            type="text"
                            placeholder="SEARCH STOPS..."
                            className="w-full px-3 py-1.5 bg-hud-panel border border-hud-border rounded-none text-[10px] text-hud-text tracking-hud uppercase font-mono placeholder:text-hud-text-dim focus:outline-none focus:border-hud-accent/40 transition-colors"
                            style={{ boxShadow: 'inset 0 0 4px rgba(0,0,0,0.3)' }}
                        />
                    </div>
                )}
            </div>

            {/* ── Content ── */}
            <div className="flex-1 overflow-hidden relative z-20">
                {routeCreatorMode ? (
                    <RouteCreatorPanel />
                ) : selectedStopId ? (
                    <div className="h-full flex flex-col">
                        <button
                            onClick={() => setSelectedStopId(null)}
                            className="flex items-center gap-2 px-4 py-2 text-[10px] text-hud-accent tracking-hud uppercase hover:bg-hud-panel border-b border-hud-border transition-colors text-left"
                        >
                            <span className="text-hud-text-dim">◂</span> BACK TO STOP INDEX
                        </button>
                        <ArrivalsList stopId={selectedStopId} />
                    </div>
                ) : (
                    <div className="h-full overflow-y-auto">
                        {/* Section header */}
                        <div className="px-4 py-2 border-b border-hud-border bg-hud-panel/50">
                            <span className="text-[9px] font-bold text-hud-text-dim tracking-hud-wide uppercase">
                                STOP INDEX // {stops.length} REGISTERED
                            </span>
                        </div>

                        {stops.map((stop, index) => (
                            <div
                                key={stop.id}
                                onClick={() => setSelectedStopId(stop.id)}
                                className="group px-4 py-3 border-b border-hud-border/50 cursor-pointer hover:bg-hud-panel/60 transition-all"
                            >
                                <div className="flex items-start gap-3">
                                    {/* Index number */}
                                    <span className="text-[10px] text-hud-text-dim font-bold tabular-nums w-4 shrink-0 mt-0.5">
                                        {String(index + 1).padStart(2, '0')}
                                    </span>
                                    <div className="flex-1 min-w-0">
                                        <div className="text-[11px] text-hud-text tracking-hud uppercase group-hover:text-hud-accent transition-colors truncate">
                                            {stop.name}
                                        </div>
                                        {stop.address && (
                                            <div className="text-[9px] text-hud-text-dim tracking-hud uppercase truncate mt-0.5">
                                                {stop.address}
                                            </div>
                                        )}
                                    </div>
                                    {/* Marker dot */}
                                    <span className="w-1.5 h-1.5 rounded-full bg-hud-accent/30 mt-1.5 shrink-0 group-hover:bg-hud-accent transition-colors"></span>
                                </div>
                            </div>
                        ))}

                        {stops.length === 0 && (
                            <div className="px-4 py-8 text-center">
                                <div className="text-[10px] text-hud-text-dim tracking-hud uppercase">
                                    AWAITING DATA FEED...
                                </div>
                            </div>
                        )}
                    </div>
                )}
            </div>

            {/* ── Footer ── */}
            <div className="relative z-20 px-4 py-2 border-t border-hud-border bg-hud-panel/30">
                <div className="text-[8px] text-hud-text-dim tracking-hud-wide uppercase text-center">
                    SYS // LATITUDEX v0.1.0 // {new Date().getFullYear()}
                </div>
            </div>
        </div>
    );
};
