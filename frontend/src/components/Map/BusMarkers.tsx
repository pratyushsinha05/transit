/**
 * Bus Markers
 * Renders buses on the map as glowing radar blips with ping animation
 */

import { Marker, Popup, Tooltip } from 'react-leaflet';
import L from 'leaflet';
import { useStore } from '../../store';
import type { BusLocation } from '../../types/domain';

// Radar blip icon — glowing dot with expanding ping rings
const createBusBlipIcon = () => L.divIcon({
    className: 'bus-blip-marker',
    html: `
        <div class="bus-blip">
            <div class="bus-blip-core"></div>
            <div class="bus-blip-ping"></div>
            <div class="bus-blip-ping bus-blip-ping-delayed"></div>
        </div>
    `,
    iconSize: [16, 16],
    iconAnchor: [8, 8],
});

const busBlipIcon = createBusBlipIcon();

export const BusMarkers = () => {
    const buses = useStore((state) => state.buses);
    const layerVisibility = useStore(state => state.layerVisibility);
    const hiddenRoutes = useStore(state => state.hiddenRoutes);

    if (!layerVisibility.buses) return null;

    const visibleBuses = Array.from(buses.values()).filter(
        (bus: BusLocation) => !hiddenRoutes.includes(bus.routeId)
    );

    return (
        <>
            {visibleBuses.map((bus: BusLocation) => (
                <Marker
                    key={bus.id}
                    position={[bus.lat, bus.lng]}
                    icon={busBlipIcon}
                >
                    <Tooltip direction="top" offset={[0, -12]} permanent={false}>
                        {bus.id}
                    </Tooltip>
                    <Popup>
                        <div className="p-3 min-w-[200px]">
                            <div className="flex items-center gap-2 mb-3 pb-2 border-b border-hud-border">
                                <span className="w-2 h-2 rounded-full bg-hud-accent animate-glow-pulse"></span>
                                <span className="text-[10px] font-bold tracking-hud-wide text-hud-accent uppercase">
                                    VEHICLE TELEMETRY
                                </span>
                            </div>

                            <div className="space-y-2">
                                <div>
                                    <div className="hud-label">VEHICLE ID</div>
                                    <div className="hud-value">{bus.id}</div>
                                </div>
                                <div className="grid grid-cols-2 gap-2">
                                    <div>
                                        <div className="hud-label">LAT</div>
                                        <div className="hud-value">{bus.lat.toFixed(6)}</div>
                                    </div>
                                    <div>
                                        <div className="hud-label">LNG</div>
                                        <div className="hud-value">{bus.lng.toFixed(6)}</div>
                                    </div>
                                </div>
                                <div className="grid grid-cols-2 gap-2">
                                    <div>
                                        <div className="hud-label">SPEED</div>
                                        <div className="hud-value">{Math.round(bus.speed)} <span className="text-[9px] text-hud-text-dim">KM/H</span></div>
                                    </div>
                                    <div>
                                        <div className="hud-label">ROUTE</div>
                                        <div className="hud-value">{bus.routeId}</div>
                                    </div>
                                </div>
                                <div className="pt-2 mt-2 border-t border-hud-border">
                                    <button 
                                        onClick={() => useStore.getState().setFollowedBusId(bus.id)}
                                        className="w-full py-1 text-[9px] font-bold tracking-hud-wide uppercase text-hud-bg bg-hud-accent/90 hover:bg-hud-accent transition-colors"
                                    >
                                        [⌖] FOLLOW TARGET
                                    </button>
                                </div>
                                <div className="pt-1 mt-1 border-t border-hud-border/30">
                                    <div className="text-[9px] text-hud-text-dim tracking-hud">
                                        LAST UPDATE: {bus.lastUpdate.toLocaleTimeString()}
                                    </div>
                                </div>
                            </div>
                        </div>
                    </Popup>
                </Marker>
            ))}
        </>
    );
};
