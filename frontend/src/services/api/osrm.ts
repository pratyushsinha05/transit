/**
 * OSRM API Service
 * Fetches driving routing data from the free public OSRM server to snap map points to actual roads
 */

import axios from 'axios';
import { logger } from '../logger';

const OSRM_BASE_URL = 'https://router.project-osrm.org/route/v1/driving';

/**
 * Fetches an actual driving route polyline connecting the given waypoints.
 * @param points Array of {lat, lng} coordinates
 * @returns Array of [lat, lng] pairs representing the snapped polyline road path
 */
export const getRouteGeometry = async (points: {lat: number, lng: number}[]): Promise<[number, number][]> => {
    if (points.length < 2) return [];
    
    // OSRM expects coordinate strings in longitude,latitude order separated by semicolons
    const coordinates = points.map(p => `${p.lng},${p.lat}`).join(';');
    const url = `${OSRM_BASE_URL}/${coordinates}?overview=full&geometries=geojson`;

    try {
        const response = await axios.get(url);
        
        if (response.data && response.data.code === 'Ok' && response.data.routes.length > 0) {
            // OSRM returning GeoJSON LineString => coordinates are [lng, lat]
            const coords = response.data.routes[0].geometry.coordinates as [number, number][];
            
            // Leaflet Polyline expects [lat, lng], so we flip them here
            return coords.map(c => [c[1], c[0]] as [number, number]);
        }
        
        throw new Error('OSRM API returned invalid or empty route');
    } catch (error) {
        logger.error('Failed to fetch OSRM road route. Falling back to straight lines.', { error });
        // Fallback to straight lines connecting the raw points if API fails
        return points.map(p => [p.lat, p.lng]);
    }
};
