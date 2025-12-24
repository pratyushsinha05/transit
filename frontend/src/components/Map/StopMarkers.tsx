/**
 * Stop Markers
 * Render stops on the map
 */

import { Marker, Popup } from 'react-leaflet';
import L from 'leaflet';
import { useStops } from '../../hooks/useStops';
import { useStore } from '../../store';

const stopIcon = L.divIcon({
    className: 'stop-marker',
    html: '<div class="w-3 h-3 bg-blue-500 rounded-full border-2 border-white shadow-md"></div>',
    iconSize: [12, 12],
    iconAnchor: [6, 6],
});

export const StopMarkers = () => {
    const { stops } = useStops(); // Uses hook which selects from store
    const setSelectedStopId = useStore(state => state.setSelectedStopId);

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
                        <div className="p-2">
                            <h3 className="font-bold text-sm">{stop.name}</h3>
                            <p className="text-xs text-gray-600">{stop.address}</p>
                            <button
                                className="mt-2 text-xs text-blue-600 hover:underline"
                                onClick={() => setSelectedStopId(stop.id)}
                            >
                                View Arrivals
                            </button>
                        </div>
                    </Popup>
                </Marker>
            ))}
        </>
    );
};
