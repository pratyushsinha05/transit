/**
 * Map Click Handler
 * Captures map clicks to add zones when route creator mode is active
 */

import { useMapEvents } from 'react-leaflet';
import { useStore } from '../../store';

export const MapClickHandler = () => {
    const routeCreatorMode = useStore(state => state.routeCreatorMode);
    const addCreatorZone = useStore(state => state.addCreatorZone);

    useMapEvents({
        click(e) {
            if (routeCreatorMode) {
                addCreatorZone(e.latlng.lat, e.latlng.lng);
            }
        },
    });

    return null;
};
