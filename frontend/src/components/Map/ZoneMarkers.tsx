/**
 * Zone Markers
 * Renders zones on the map as SVG crosshair reticles with HUD popups
 */

import { Marker, Popup } from 'react-leaflet';
import L from 'leaflet';
import { useZones } from '../../hooks/useZones';
import { useStore } from '../../store';
import { useGeofenceEvents } from '../../hooks/useGeofenceEvents';

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

const zoneIcon = L.divIcon({
    className: 'zone-crosshair-marker',
    html: crosshairSvg,
    iconSize: [18, 18],
    iconAnchor: [9, 9],
});

/** Inner popup component that fetches geofence events on mount */
const ZonePopupContent = ({ zoneId, zoneName, zoneAddress }: { zoneId: string; zoneName: string; zoneAddress?: string }) => {
    const { events } = useGeofenceEvents(zoneId);

    return (
        <div className="p-3 min-w-[220px]">
            <div className="flex items-center gap-2 mb-3 pb-2 border-b border-hud-border">
                <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                    <circle cx="5" cy="5" r="3" stroke="#00f5d4" strokeWidth="1" fill="none" />
                    <circle cx="5" cy="5" r="1" fill="#00f5d4" />
                </svg>
                <span className="text-[10px] font-bold tracking-hud-wide text-hud-accent uppercase">
                    ZONE INTEL
                </span>
            </div>

            <div className="space-y-2">
                <div>
                    <div className="hud-label">DESIGNATION</div>
                    <div className="hud-value text-[12px]">{zoneName.toUpperCase()}</div>
                </div>
                {zoneAddress && (
                    <div>
                        <div className="hud-label">SECTOR</div>
                        <div className="text-[10px] text-hud-text tracking-hud">{zoneAddress.toUpperCase()}</div>
                    </div>
                )}

                {events.length > 0 && (
                    <div className="pt-2 border-t border-hud-border">
                        <div className="hud-label mb-1">INCOMING VEHICLES</div>
                        {events.slice(0, 3).map((event, i) => (
                            <div key={event.id || i} className="flex justify-between items-center py-1 border-b border-hud-border/50 last:border-b-0">
                                <span className="text-[10px] text-hud-text tracking-hud">{event.deviceName || event.deviceId}</span>
                                <span className="text-hud-accent font-bold text-[12px]">
                                    {event.etaMinutes}<span className="text-[8px] text-hud-text-dim ml-0.5">MIN</span>
                                </span>
                            </div>
                        ))}
                    </div>
                )}

                {events.length === 0 && (
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

export const ZoneMarkers = () => {
    const { zones } = useZones();
    const setSelectedZoneId = useStore(state => state.setSelectedZoneId);
    const layerVisibility = useStore(state => state.layerVisibility);

    if (!layerVisibility.zones) return null;

    return (
        <>
            {zones.map((zone) => (
                <Marker
                    key={zone.id}
                    position={[zone.lat, zone.lng]}
                    icon={zoneIcon}
                    eventHandlers={{
                        click: () => setSelectedZoneId(zone.id),
                    }}
                >
                    <Popup>
                        <ZonePopupContent
                            zoneId={zone.id}
                            zoneName={zone.name}
                            zoneAddress={zone.address}
                        />
                    </Popup>
                </Marker>
            ))}
        </>
    );
};

export const StopMarkers = ZoneMarkers;
