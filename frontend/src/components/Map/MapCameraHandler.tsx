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
    const followedDeviceId = useStore(state => state.followedDeviceId);
    const devices = useStore(state => state.devices);
    const setFollowedDeviceId = useStore(state => state.setFollowedDeviceId);
    
    // Follow target logic
    useEffect(() => {
        if (!followedDeviceId) return;

        const targetDevice = devices.get(followedDeviceId);
        if (targetDevice) {
            // Smoothly pan map to device location
            map.panTo([targetDevice.lat, targetDevice.lng], { animate: true, duration: 1 });
        } else {
            // Device disappeared (e.g. disconnected), drop follow lock
            setFollowedDeviceId(null);
        }
    }, [followedDeviceId, devices, map, setFollowedDeviceId]);

    // Clear follow mode if user drastically interacts with the map (drags it)
    useEffect(() => {
        const handleDrag = () => {
            if (useStore.getState().followedDeviceId) {
                setFollowedDeviceId(null);
            }
        };

        map.on('dragstart', handleDrag);
        return () => {
            map.off('dragstart', handleDrag);
        };
    }, [map, setFollowedDeviceId]);

    return (
        <div className="leaflet-bottom leaflet-left pointer-events-none" style={{ bottom: 20, left: 10 }}>
            {/* Custom overlaid controls inside the Leaflet control pane area */}
            <div className="pointer-events-auto flex flex-col gap-2 mb-2 ml-2">
                <button
                    onClick={(e) => {
                        e.stopPropagation();
                        setFollowedDeviceId(null);
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

                {followedDeviceId && (
                    <div className="px-2 py-1 bg-hud-panel/90 border border-hud-accent text-[9px] font-bold text-hud-accent tracking-hud uppercase flex items-center gap-1.5"
                         style={{ backdropFilter: 'blur(4px)' }}>
                        <span className="w-1.5 h-1.5 rounded-full bg-hud-accent animate-glow-pulse"></span>
                        LOCK: {followedDeviceId}
                        <button
                            onClick={() => setFollowedDeviceId(null)}
                            className="ml-1 text-hud-text-dim hover:text-hud-danger text-[10px]"
                        >
                            ✕
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
};
