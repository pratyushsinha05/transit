/**
 * Route Polyline
 * Render route paths using GeoJSON
 */

import { GeoJSON } from 'react-leaflet';
import { useRoutes } from '../../hooks/useRoutes';

export const RoutePolyline = () => {
    const { routes } = useRoutes();

    return (
        <>
            {routes.map((route) => {
                if (!route.pattern) return null;

                return (
                    <GeoJSON
                        key={route.id}
                        data={route.pattern}
                        style={() => ({
                            color: '#3b82f6',
                            weight: 3,
                            opacity: 0.6,
                        })}
                    />
                );
            })}
        </>
    );
};
