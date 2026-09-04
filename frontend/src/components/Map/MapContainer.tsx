/**
 * Map Container
 * Renders the main map view with CartoDB Dark Matter tiles
 */

import { MapContainer as LeafletMap, TileLayer, ZoomControl } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import L from 'leaflet';
import { DeviceMarkers } from './DeviceMarkers';
import { ZoneMarkers } from './ZoneMarkers';
import { MapClickHandler } from './MapClickHandler';
import { RouteCreatorMarkers } from './RouteCreatorMarkers';
import { MapCameraHandler } from './MapCameraHandler';

// Fix leaflet icon issue in React
import icon from 'leaflet/dist/images/marker-icon.png';
import iconShadow from 'leaflet/dist/images/marker-shadow.png';

const DefaultIcon = L.icon({
    iconUrl: icon,
    shadowUrl: iconShadow,
    iconSize: [25, 41],
    iconAnchor: [12, 41]
});

L.Marker.prototype.options.icon = DefaultIcon;

const center: [number, number] = [
    parseFloat(import.meta.env.VITE_MAP_CENTER_LAT || '37.7749'),
    parseFloat(import.meta.env.VITE_MAP_CENTER_LNG || '-122.4194')
];

export const MapContainer = () => {
    return (
        <LeafletMap
            center={center}
            zoom={13}
            style={{ height: '100%', width: '100%' }}
            zoomControl={false}
        >
            <TileLayer
                attribution='&copy; <a href="https://carto.com/">CARTO</a>'
                url="https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png"
                subdomains="abcd"
                maxZoom={20}
            />
            <ZoomControl position="bottomright" />

            <MapCameraHandler />

            {/* Map click handler for route creation */}
            <MapClickHandler />

            {/* Existing data layers */}
            <DeviceMarkers />
            <ZoneMarkers />

            {/* Route creator layer */}
            <RouteCreatorMarkers />
        </LeafletMap>
    );
};
