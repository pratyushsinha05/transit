/**
 * Route Polyline
 * Renders route paths as glowing cyan/teal polylines with layered glow effect
 */

import { Polyline } from 'react-leaflet';
import { useRoutes } from '../../hooks/useRoutes';
import { useStore } from '../../store';
import type { Route } from '../../types/domain';

/** Extract coordinates from GeoJSON LineString */
const getCoordinates = (route: Route): [number, number][] => {
    if (!route.pattern?.coordinates) return [];
    // GeoJSON is [lng, lat], Leaflet needs [lat, lng]
    return route.pattern.coordinates.map(
        (coord: number[]) => [coord[1], coord[0]] as [number, number]
    );
};

export const RoutePolyline = () => {
    const { routes } = useRoutes();
    const hiddenRoutes = useStore(state => state.hiddenRoutes);

    return (
        <>
            {routes.map((route) => {
                if (hiddenRoutes.includes(route.id)) return null;
                const positions = getCoordinates(route);
                if (positions.length === 0) return null;

                return (
                    <span key={route.id}>
                        {/* Outer glow layer */}
                        <Polyline
                            positions={positions}
                            pathOptions={{
                                color: '#00f5d4',
                                weight: 12,
                                opacity: 0.12,
                                lineCap: 'round',
                                lineJoin: 'round',
                            }}
                        />
                        {/* Mid glow layer */}
                        <Polyline
                            positions={positions}
                            pathOptions={{
                                color: '#00f5d4',
                                weight: 6,
                                opacity: 0.25,
                                lineCap: 'round',
                                lineJoin: 'round',
                            }}
                        />
                        {/* Core line */}
                        <Polyline
                            positions={positions}
                            pathOptions={{
                                color: '#00f5d4',
                                weight: 2,
                                opacity: 0.9,
                                lineCap: 'round',
                                lineJoin: 'round',
                            }}
                        />
                    </span>
                );
            })}
        </>
    );
};
