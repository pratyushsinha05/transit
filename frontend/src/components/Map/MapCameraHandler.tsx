/**
 * Map Camera Handler
 * Coordinates map interactions, recentering, and live vehicle tracking (Follow Mode).
 */

import { useMap } from 'react-leaflet';
import { useStore } from '../../store';
import { useEffect } from 'react';

const DEFAULT_CENTER: [number, number] = [
    parseFloat(import.meta.env.VITE_MAP_CENTER_LAT || '37.7749'),
    parseFloat(import.meta.env.VITE_MAP_CENTER_LNG || '-122.4194')
];

export const MapCameraHandler = () => {
    const map = useMap();
    const followedBusId = useStore(state => state.followedBusId);
    const buses = useStore(state => state.buses);
    const setFollowedBusId = useStore(state => state.setFollowedBusId);
    
    // Follow target logic
    useEffect(() => {
        if (!followedBusId) return;

        const targetBus = buses.get(followedBusId);
        if (targetBus) {
            // Smoothly pan map to bus location
            map.panTo([targetBus.lat, targetBus.lng], { animate: true, duration: 1 });
        } else {
            // Bus disappeared (e.g. disconnected), drop follow lock
            setFollowedBusId(null);
        }
    }, [followedBusId, buses, map, setFollowedBusId]);

    // Clear follow mode if user drastically interacts with the map (drags it)
    useEffect(() => {
        const handleDrag = () => {
            if (useStore.getState().followedBusId) {
                setFollowedBusId(null);
            }
        };

        map.on('dragstart', handleDrag);
        return () => {
            map.off('dragstart', handleDrag);
        };
    }, [map, setFollowedBusId]);

    return (
        <div className="leaflet-bottom leaflet-left pointer-events-none" style={{ bottom: 20, left: 10 }}>
            {/* Custom overlaid controls inside the Leaflet control pane area */}
            <div className="pointer-events-auto flex flex-col gap-2 mb-2 ml-2">
                <button
                    onClick={(e) => {
                        e.stopPropagation();
                        setFollowedBusId(null);
                        map.flyTo(DEFAULT_CENTER, 13, { duration: 1.5 });
                    }}
                    className="w-10 h-10 bg-hud-panel/80 border border-hud-border flex items-center justify-center text-hud-accent hover:border-hud-accent/60 hover:bg-hud-panel transition-all"
                    title="Recenter Map"
                    style={{ backdropFilter: 'blur(4px)' }}
                >
                    <svg viewBox="0 0 24 24" width="18" height="18" stroke="currentColor" strokeWidth="2" fill="none">
                        <circle cx="12" cy="12" r="3" />
                        <path d="M12 2v4M12 18v4M4 12H2M22 12h-2" />
                    </svg>
                </button>

                {followedBusId && (
                    <div className="bg-hud-accent/20 border border-hud-accent px-3 py-1.5 flex flex-col justify-center animate-pulse-slow backdrop-blur-sm">
                        <span className="text-[8px] font-bold tracking-hud-wide text-hud-bright uppercase">
                            TRACKING LOCK: {followedBusId}
                        </span>
                        <span className="text-[7px] text-hud-accent tracking-widest uppercase">
                            DRAG TO ABORT
                        </span>
                    </div>
                )}
            </div>
        </div>
    );
};
