/**
 * Stop Markers
 * Renders stops on the map as SVG crosshair reticles with HUD popups
 */

import { Marker, Popup } from 'react-leaflet';
import L from 'leaflet';
import { useStops } from '../../hooks/useStops';
import { useStore } from '../../store';
import { useArrivals } from '../../hooks/useArrivals';

// SVG crosshair reticle icon
const crosshairSvg = `
<svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg">
  <circle cx="9" cy="9" r="5" stroke="%2300f5d4" stroke-width="1" fill="none" opacity="0.6"/>
  <circle cx="9" cy="9" r="1.5" fill="%2300f5d4" opacity="0.8"/>
  <line x1="9" y1="0" x2="9" y2="5" stroke="%2300f5d4" stroke-width="0.8" opacity="0.5"/>
  <line x1="9" y1="13" x2="9" y2="18" stroke="%2300f5d4" stroke-width="0.8" opacity="0.5"/>
  <line x1="0" y1="9" x2="5" y2="9" stroke="%2300f5d4" stroke-width="0.8" opacity="0.5"/>
  <line x1="13" y1="9" x2="18" y2="9" stroke="%2300f5d4" stroke-width="0.8" opacity="0.5"/>
</svg>
`;

const stopIcon = L.divIcon({
    className: 'stop-crosshair-marker',
    html: crosshairSvg,
    iconSize: [18, 18],
    iconAnchor: [9, 9],
});

/** Inner popup component that fetches arrivals on mount */
const StopPopupContent = ({ stopId, stopName, stopAddress }: { stopId: string; stopName: string; stopAddress?: string }) => {
    const { arrivals } = useArrivals(stopId);

    return (
        <div className="p-3 min-w-[220px]">
            <div className="flex items-center gap-2 mb-3 pb-2 border-b border-hud-border">
                <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                    <circle cx="5" cy="5" r="3" stroke="#00f5d4" strokeWidth="1" fill="none" />
                    <circle cx="5" cy="5" r="1" fill="#00f5d4" />
                </svg>
                <span className="text-[10px] font-bold tracking-hud-wide text-hud-accent uppercase">
                    STOP INTEL
                </span>
            </div>

            <div className="space-y-2">
                <div>
                    <div className="hud-label">DESIGNATION</div>
                    <div className="hud-value text-[12px]">{stopName.toUpperCase()}</div>
                </div>
                {stopAddress && (
                    <div>
                        <div className="hud-label">SECTOR</div>
                        <div className="text-[10px] text-hud-text tracking-hud">{stopAddress.toUpperCase()}</div>
                    </div>
                )}

                {arrivals.length > 0 && (
                    <div className="pt-2 border-t border-hud-border">
                        <div className="hud-label mb-1">INCOMING VEHICLES</div>
                        {arrivals.slice(0, 3).map((arrival, i) => (
                            <div key={arrival.id || i} className="flex justify-between items-center py-1 border-b border-hud-border/50 last:border-b-0">
                                <span className="text-[10px] text-hud-text tracking-hud">{arrival.route}</span>
                                <span className="text-hud-accent font-bold text-[12px]">
                                    {arrival.eta}<span className="text-[8px] text-hud-text-dim ml-0.5">MIN</span>
                                </span>
                            </div>
                        ))}
                    </div>
                )}

                {arrivals.length === 0 && (
                    <div className="pt-2 border-t border-hud-border">
                        <div className="text-[9px] text-hud-text-dim tracking-hud text-center py-1">
                            NO ACTIVE APPROACHES
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
};

export const StopMarkers = () => {
    const { stops } = useStops();
    const setSelectedStopId = useStore(state => state.setSelectedStopId);
    const layerVisibility = useStore(state => state.layerVisibility);

    if (!layerVisibility.stops) return null;

    return (
        <>
            {stops.map((stop) => (
                <Marker
                    key={stop.id}
                    position={[stop.lat, stop.lng]}
                    icon={stopIcon}
                    eventHandlers={{
                        click: () => setSelectedStopId(stop.id),
                    }}
                >
                    <Popup>
                        <StopPopupContent
                            stopId={stop.id}
                            stopName={stop.name}
                            stopAddress={stop.address}
                        />
                    </Popup>
                </Marker>
            ))}
        </>
    );
};
