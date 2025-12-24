/**
 * Bus Markers
 * Render buses on the map with rotation
 */

import { Marker, Popup, Tooltip } from 'react-leaflet';
import L from 'leaflet';
import { useStore } from '../../store';
import type { BusLocation } from '../../types/domain';

// Custom bus icon factory (can be enhanced with SVG)
const createBusIcon = (heading: number) => L.divIcon({
    className: 'bus-marker',
    html: `<div class="bus-icon" style="transform: rotate(${heading}deg);">🚌</div>`,
    iconSize: [30, 30],
    iconAnchor: [15, 15],
});

export const BusMarkers = () => {
    const buses = useStore((state) => state.buses); // Map<string, BusLocation>

    return (
        <>
            {Array.from(buses.values()).map((bus: BusLocation) => (
                <Marker
                    key={bus.id}
                    position={[bus.lat, bus.lng]}
                    icon={createBusIcon(bus.heading)}
                >
                    <Tooltip direction="top" offset={[0, -15]} permanent={false}>
                        Route {bus.routeId}
                    </Tooltip>
                    <Popup>
                        <div className="p-2">
                            <h3 className="font-bold">Bus {bus.id}</h3>
                            <p>Route: {bus.routeId}</p>
                            <p>Speed: {Math.round(bus.speed)} km/h</p>
                            <p className="text-xs text-gray-500">
                                Updated: {bus.lastUpdate.toLocaleTimeString()}
                            </p>
                        </div>
                    </Popup>
                </Marker>
            ))}
        </>
    );
};
