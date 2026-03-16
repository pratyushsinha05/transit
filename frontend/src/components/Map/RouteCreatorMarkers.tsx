import { Marker, Polyline, Tooltip } from 'react-leaflet';
import L from 'leaflet';
import { useStore } from '../../store';
import { useCallback, useEffect, useState } from 'react';
import { getRouteGeometry } from '../../services/api/osrm';

/** Create a numbered marker icon for route builder stops */
const createNumberedIcon = (index: number) => L.divIcon({
    className: 'stop-crosshair-marker',
    html: `
        <div style="
            position: relative;
            width: 24px;
            height: 24px;
            display: flex;
            align-items: center;
            justify-content: center;
        ">
            <div style="
                width: 22px;
                height: 22px;
                border: 1.5px solid #00f5d4;
                border-radius: 50%;
                background: rgba(0, 245, 212, 0.15);
                display: flex;
                align-items: center;
                justify-content: center;
                font-family: 'Space Mono', monospace;
                font-size: 10px;
                font-weight: 700;
                color: #00f5d4;
                box-shadow: 0 0 8px rgba(0, 245, 212, 0.3);
            ">${index + 1}</div>
        </div>
    `,
    iconSize: [24, 24],
    iconAnchor: [12, 12],
});

export const RouteCreatorMarkers = () => {
    const routeCreatorMode = useStore(state => state.routeCreatorMode);
    const stops = useStore(state => state.routeCreatorStops);
    const updateCreatorStopPosition = useStore(state => state.updateCreatorStopPosition);

    // State to hold the snapped road polyline
    const [pathGeometry, setPathGeometry] = useState<[number, number][]>([]);

    // Fetch road route whenever stops change (drag ends, added, removed, reordered)
    useEffect(() => {
        let isMounted = true;

        if (stops.length < 2) {
            setPathGeometry([]);
            return;
        }

        const fetchRoute = async () => {
            const rawPoints = stops.map(s => ({ lat: s.lat, lng: s.lng }));
            
            // Temporarily connect with straight lines while loading fast
            if (pathGeometry.length === 0) {
                 setPathGeometry(rawPoints.map(p => [p.lat, p.lng]));
            }

            const snappedPath = await getRouteGeometry(rawPoints);
            
            if (isMounted) {
                setPathGeometry(snappedPath);
            }
        };

        fetchRoute();

        return () => {
            isMounted = false;
        };
    }, [stops]);

    const handleDragEnd = useCallback((tempId: string, e: L.DragEndEvent) => {
        const latlng = e.target.getLatLng();
        updateCreatorStopPosition(tempId, latlng.lat, latlng.lng);
    }, [updateCreatorStopPosition]);

    if (!routeCreatorMode || stops.length === 0) return null;

    return (
        <>
            {/* Connecting road-snapped polyline */}
            {pathGeometry.length >= 2 && (
                <>
                    <Polyline
                        positions={pathGeometry}
                        pathOptions={{
                            color: '#00f5d4',
                            weight: 8,
                            opacity: 0.15,
                            dashArray: '12, 8',
                            lineCap: 'round',
                        }}
                    />
                    <Polyline
                        positions={pathGeometry}
                        pathOptions={{
                            color: '#00f5d4',
                            weight: 2,
                            opacity: 0.7,
                            dashArray: '12, 8',
                            lineCap: 'round',
                        }}
                    />
                </>
            )}

            {/* Numbered stop markers */}
            {stops.map((stop, index) => (
                <Marker
                    key={stop.tempId}
                    position={[stop.lat, stop.lng]}
                    icon={createNumberedIcon(index)}
                    draggable={true}
                    eventHandlers={{
                        dragend: (e) => handleDragEnd(stop.tempId, e),
                    }}
                >
                    <Tooltip direction="right" offset={[15, 0]} permanent={false}>
                        {stop.name}
                    </Tooltip>
                </Marker>
            ))}
        </>
    );
};
