/**
 * Map Click Handler
 * Captures map clicks to add stops when route creator mode is active
 */

import { useMapEvents } from 'react-leaflet';
import { useStore } from '../../store';

export const MapClickHandler = () => {
    const routeCreatorMode = useStore(state => state.routeCreatorMode);
    const addCreatorStop = useStore(state => state.addCreatorStop);

    useMapEvents({
        click(e) {
            if (routeCreatorMode) {
                addCreatorStop(e.latlng.lat, e.latlng.lng);
            }
        },
    });

    return null;
};
