import { Marker, Polyline, Tooltip } from 'react-leaflet';
import L from 'leaflet';
import { useStore } from '../../store';
import { useCallback, useEffect, useState } from 'react';
import { getRouteGeometry } from '../../services/api/osrm';

/** Create a numbered marker icon for route builder zones */
const createNumberedIcon = (index: number) => L.divIcon({
    className: 'zone-crosshair-marker',
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
    const zones = useStore(state => state.routeCreatorZones);
    const updateCreatorZonePosition = useStore(state => state.updateCreatorZonePosition);

    // State to hold the snapped road polyline
    const [pathGeometry, setPathGeometry] = useState<[number, number][]>([]);

    // Fetch road route whenever zones change (drag ends, added, removed, reordered)
    useEffect(() => {
        let isMounted = true;

        if (zones.length < 2) {
            setPathGeometry([]);
            return;
        }

        const fetchRoute = async () => {
            const rawPoints = zones.map(z => ({ lat: z.lat, lng: z.lng }));
            
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
    }, [zones]);

    const handleDragEnd = useCallback((tempId: string, e: L.DragEndEvent) => {
        const latlng = e.target.getLatLng();
        updateCreatorZonePosition(tempId, latlng.lat, latlng.lng);
    }, [updateCreatorZonePosition]);

    if (!routeCreatorMode || zones.length === 0) return null;

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

            {/* Numbered zone markers */}
            {zones.map((zone, index) => (
                <Marker
                    key={zone.tempId}
                    position={[zone.lat, zone.lng]}
                    icon={createNumberedIcon(index)}
                    draggable={true}
                    eventHandlers={{
                        dragend: (e) => handleDragEnd(zone.tempId, e),
                    }}
                >
                    <Tooltip direction="right" offset={[15, 0]} permanent={false}>
                        {zone.name}
                    </Tooltip>
                </Marker>
            ))}
        </>
    );
};
