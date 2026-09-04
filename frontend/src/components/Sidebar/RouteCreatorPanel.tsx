/**
 * Route Creator Panel
 * HUD-styled sidebar panel for building routes with zones
 */

import { useState } from 'react';
import { useStore } from '../../store';
import { createRoute } from '../../services/api/createRoute';
import { logger } from '../../services/logger';

export const RouteCreatorPanel = () => {
    const zones = useStore(state => state.routeCreatorZones);
    const routeName = useStore(state => state.routeCreatorName);
    const routeDescription = useStore(state => state.routeCreatorDescription);
    const setRouteCreatorName = useStore(state => state.setRouteCreatorName);
    const setRouteCreatorDescription = useStore(state => state.setRouteCreatorDescription);
    const removeCreatorZone = useStore(state => state.removeCreatorZone);
    const updateCreatorZoneName = useStore(state => state.updateCreatorZoneName);
    const reorderCreatorZones = useStore(state => state.reorderCreatorZones);
    const clearCreatorZones = useStore(state => state.clearCreatorZones);
    const toggleRouteCreator = useStore(state => state.toggleRouteCreator);

    const [submitting, setSubmitting] = useState(false);
    const [successMessage, setSuccessMessage] = useState<string | null>(null);
    const [errorMessage, setErrorMessage] = useState<string | null>(null);

    const canSubmit = routeName.trim().length > 0 && zones.length >= 2 && !submitting;

    const handleSubmit = async () => {
        if (!canSubmit) return;

        setSubmitting(true);
        setErrorMessage(null);
        setSuccessMessage(null);

        try {
            const result = await createRoute({
                name: routeName.trim(),
                description: routeDescription.trim(),
                stops: zones.map(z => ({
                    name: z.name,
                    latitude: z.lat,
                    longitude: z.lng,
                })),
            });

            setSuccessMessage(`ROUTE DEPLOYED: ${result.id} // ${result.stop_count} ZONES`);
            logger.info('Route created', { result });

            // Clear after short delay to show success
            setTimeout(() => {
                clearCreatorZones();
                toggleRouteCreator();
                setSuccessMessage(null);
            }, 2000);
        } catch (err: any) {
            setErrorMessage(err?.response?.data?.error || 'DEPLOYMENT FAILED');
            logger.error('Route creation failed', { error: err });
        } finally {
            setSubmitting(false);
        }
    };

    const moveZone = (index: number, direction: 'up' | 'down') => {
        const newIndex = direction === 'up' ? index - 1 : index + 1;
        if (newIndex >= 0 && newIndex < zones.length) {
            reorderCreatorZones(index, newIndex);
        }
    };

    return (
        <div className="h-full flex flex-col">
            {/* Header */}
            <div className="px-4 py-3 border-b border-hud-border bg-hud-panel/40">
                <div className="flex items-center gap-2 mb-2">
                    <span className="w-1.5 h-1.5 rounded-full bg-hud-warn animate-blink"></span>
                    <span className="text-[9px] font-bold tracking-hud-wide uppercase text-hud-warn">
                        ROUTE BUILDER ACTIVE
                    </span>
                </div>
                <p className="text-[9px] text-hud-text-dim tracking-hud uppercase">
                    CLICK MAP TO ADD WAYPOINTS // DRAG TO REPOSITION
                </p>
            </div>

            {/* Route metadata */}
            <div className="px-4 py-3 border-b border-hud-border space-y-2">
                <div>
                    <label className="hud-label block mb-1">ROUTE DESIGNATION</label>
                    <input
                        type="text"
                        value={routeName}
                        onChange={(e) => setRouteCreatorName(e.target.value)}
                        placeholder="E.G. DOWNTOWN EXPRESS"
                        className="w-full px-3 py-1.5 bg-hud-panel border border-hud-border rounded-none text-[11px] text-hud-accent tracking-hud uppercase font-mono placeholder:text-hud-text-dim focus:outline-none focus:border-hud-accent/40 transition-colors"
                    />
                </div>
                <div>
                    <label className="hud-label block mb-1">DESCRIPTION</label>
                    <input
                        type="text"
                        value={routeDescription}
                        onChange={(e) => setRouteCreatorDescription(e.target.value)}
                        placeholder="OPTIONAL ROUTE DESCRIPTION"
                        className="w-full px-3 py-1.5 bg-hud-panel border border-hud-border rounded-none text-[10px] text-hud-text tracking-hud uppercase font-mono placeholder:text-hud-text-dim focus:outline-none focus:border-hud-accent/40 transition-colors"
                    />
                </div>
            </div>

            {/* Zone list header */}
            <div className="px-4 py-2 border-b border-hud-border bg-hud-panel/20">
                <span className="text-[9px] font-bold text-hud-text-dim tracking-hud-wide uppercase">
                    WAYPOINTS // {zones.length} PLACED
                </span>
            </div>

            {/* Zones list */}
            <div className="flex-1 overflow-y-auto">
                {zones.length === 0 ? (
                    <div className="px-4 py-8 text-center">
                        <div className="text-[10px] text-hud-text-dim tracking-hud uppercase">
                            CLICK MAP TO PLACE FIRST WAYPOINT
                        </div>
                    </div>
                ) : (
                    zones.map((zone, index) => (
                        <div
                            key={zone.tempId}
                            className="px-4 py-2.5 border-b border-hud-border/50 hover:bg-hud-panel/40 transition-colors"
                        >
                            <div className="flex items-center gap-2">
                                {/* Index */}
                                <span className="text-[10px] text-hud-accent font-bold w-4 shrink-0 tabular-nums">
                                    {String(index + 1).padStart(2, '0')}
                                </span>

                                {/* Name input */}
                                <input
                                    type="text"
                                    value={zone.name}
                                    onChange={(e) => updateCreatorZoneName(zone.tempId, e.target.value)}
                                    className="flex-1 bg-transparent border-b border-hud-border/30 text-[10px] text-hud-text tracking-hud uppercase font-mono focus:outline-none focus:border-hud-accent/40 py-0.5 min-w-0"
                                />

                                {/* Reorder buttons */}
                                <div className="flex flex-col gap-0 shrink-0">
                                    <button
                                        onClick={() => moveZone(index, 'up')}
                                        disabled={index === 0}
                                        className="text-[8px] text-hud-text-dim hover:text-hud-accent disabled:opacity-20 transition-colors leading-none"
                                    >▲</button>
                                    <button
                                        onClick={() => moveZone(index, 'down')}
                                        disabled={index === zones.length - 1}
                                        className="text-[8px] text-hud-text-dim hover:text-hud-accent disabled:opacity-20 transition-colors leading-none"
                                    >▼</button>
                                </div>

                                {/* Delete */}
                                <button
                                    onClick={() => removeCreatorZone(zone.tempId)}
                                    className="text-[9px] text-hud-danger/60 hover:text-hud-danger transition-colors shrink-0"
                                >✕</button>
                            </div>

                            {/* Coordinates */}
                            <div className="flex gap-3 mt-1 ml-6">
                                <span className="text-[8px] text-hud-text-dim tracking-wider">
                                    LAT {zone.lat.toFixed(6)}
                                </span>
                                <span className="text-[8px] text-hud-text-dim tracking-wider">
                                    LNG {zone.lng.toFixed(6)}
                                </span>
                            </div>
                        </div>
                    ))
                )}
            </div>

            {/* Status messages */}
            {successMessage && (
                <div className="px-4 py-2 border-t border-hud-accent/30 bg-hud-accent/5">
                    <span className="text-[9px] text-hud-accent tracking-hud uppercase font-bold">
                        ✓ {successMessage}
                    </span>
                </div>
            )}
            {errorMessage && (
                <div className="px-4 py-2 border-t border-hud-danger/30 bg-hud-danger/5">
                    <span className="text-[9px] text-hud-danger tracking-hud uppercase font-bold">
                        ⚠ {errorMessage}
                    </span>
                </div>
            )}

            {/* Action buttons */}
            <div className="px-4 py-3 border-t border-hud-border space-y-2">
                {/* Validation hint */}
                {zones.length < 2 && zones.length > 0 && (
                    <div className="text-[8px] text-hud-warn tracking-hud uppercase text-center">
                        MINIMUM 2 WAYPOINTS REQUIRED
                    </div>
                )}
                {zones.length >= 2 && !routeName.trim() && (
                    <div className="text-[8px] text-hud-warn tracking-hud uppercase text-center">
                        ROUTE DESIGNATION REQUIRED
                    </div>
                )}

                <button
                    onClick={handleSubmit}
                    disabled={!canSubmit}
                    className="w-full py-2 text-[10px] font-bold tracking-hud-wide uppercase border transition-all disabled:opacity-30 disabled:cursor-not-allowed"
                    style={{
                        color: canSubmit ? '#0a0e1a' : '#576574',
                        backgroundColor: canSubmit ? '#00f5d4' : 'transparent',
                        borderColor: canSubmit ? '#00f5d4' : '#1a2332',
                        boxShadow: canSubmit ? '0 0 12px rgba(0, 245, 212, 0.3)' : 'none',
                    }}
                >
                    {submitting ? 'DEPLOYING...' : 'DEPLOY ROUTE'}
                </button>

                <button
                    onClick={() => {
                        clearCreatorZones();
                        toggleRouteCreator();
                    }}
                    className="w-full py-1.5 text-[9px] tracking-hud-wide uppercase text-hud-text-dim border border-hud-border hover:text-hud-danger hover:border-hud-danger/40 transition-colors"
                >
                    ABORT
                </button>
            </div>
        </div>
    );
};
