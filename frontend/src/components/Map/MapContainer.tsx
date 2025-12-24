/**
 * Map Container
 * Renders the main map view
 */

import { MapContainer as LeafletMap, TileLayer, ZoomControl } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import L from 'leaflet';
import { BusMarkers } from './BusMarkers';
import { StopMarkers } from './StopMarkers';
import { RoutePolyline } from './RoutePolyline';

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
                attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
            />
            <ZoomControl position="bottomright" />

            <BusMarkers />
            <StopMarkers />
            <RoutePolyline />
        </LeafletMap>
    );
};
