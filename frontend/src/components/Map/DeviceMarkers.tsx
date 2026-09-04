/**
 * Device Markers
 * Renders devices on the map as glowing radar blips with ping animation
 */

import { Marker, Popup, Tooltip } from 'react-leaflet';
import L from 'leaflet';
import { useStore } from '../../store';
import type { DeviceLocation } from '../../types/domain';

// Radar blip icon — glowing dot with expanding ping rings
const createDeviceBlipIcon = () => L.divIcon({
    className: 'device-blip-marker',
    html: `
        <div class="device-blip">
            <div class="device-blip-core"></div>
            <div class="device-blip-ping"></div>
            <div class="device-blip-ping device-blip-ping-delayed"></div>
        </div>
    `,
    iconSize: [16, 16],
    iconAnchor: [8, 8],
});

const deviceBlipIcon = createDeviceBlipIcon();

export const DeviceMarkers = () => {
    const devices = useStore((state) => state.devices);
    const layerVisibility = useStore(state => state.layerVisibility);
    const hiddenRoutes = useStore(state => state.hiddenRoutes);

    if (!layerVisibility.devices) return null;

    const visibleDevices = Array.from(devices.values()).filter(
        (device: DeviceLocation) => !hiddenRoutes.includes(device.routeId)
    );

    return (
        <>
            {visibleDevices.map((device: DeviceLocation) => (
                <Marker
                    key={device.id}
                    position={[device.lat, device.lng]}
                    icon={deviceBlipIcon}
                >
                    <Tooltip direction="top" offset={[0, -12]} permanent={false}>
                        {device.id}
                    </Tooltip>
                    <Popup>
                        <div className="p-3 min-w-[200px]">
                            <div className="flex items-center gap-2 mb-3 pb-2 border-b border-hud-border">
                                <span className="w-2 h-2 rounded-full bg-hud-accent animate-glow-pulse"></span>
                                <span className="text-[10px] font-bold tracking-hud-wide text-hud-accent uppercase">
                                    DEVICE TELEMETRY
                                </span>
                            </div>

                            <div className="space-y-2">
                                <div>
                                    <div className="hud-label">DEVICE ID</div>
                                    <div className="hud-value">{device.id}</div>
                                </div>
                                <div className="grid grid-cols-2 gap-2">
                                    <div>
                                        <div className="hud-label">LAT</div>
                                        <div className="hud-value">{device.lat.toFixed(6)}</div>
                                    </div>
                                    <div>
                                        <div className="hud-label">LNG</div>
                                        <div className="hud-value">{device.lng.toFixed(6)}</div>
                                    </div>
                                </div>
                                <div className="grid grid-cols-2 gap-2">
                                    <div>
                                        <div className="hud-label">SPEED</div>
                                        <div className="hud-value">{Math.round(device.speed)} <span className="text-[9px] text-hud-text-dim">KM/H</span></div>
                                    </div>
                                    <div>
                                        <div className="hud-label">ROUTE</div>
                                        <div className="hud-value">{device.routeId}</div>
                                    </div>
                                </div>
                                <div className="pt-2 mt-2 border-t border-hud-border">
                                    <button 
                                        onClick={() => useStore.getState().setFollowedDeviceId(device.id)}
                                        className="w-full py-1 text-[9px] font-bold tracking-hud-wide uppercase text-hud-bg bg-hud-accent/90 hover:bg-hud-accent transition-colors"
                                    >
                                        [⌖] FOLLOW TARGET
                                    </button>
                                </div>
                                <div className="pt-1 mt-1 border-t border-hud-border/30">
                                    <div className="text-[9px] text-hud-text-dim tracking-hud">
                                        LAST UPDATE: {device.lastUpdate.toLocaleTimeString()}
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

export const BusMarkers = DeviceMarkers;
