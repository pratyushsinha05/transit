/**
 * Operations Panel
 * Right-side HUD panel for toggling routes and map layers
 */

import { useStore } from '../../store';
import { useRoutes } from '../../hooks/useRoutes';

export const OperationsPanel = () => {
    const { routes } = useRoutes();
    const layerVisibility = useStore(state => state.layerVisibility);
    const hiddenRoutes = useStore(state => state.hiddenRoutes);
    const toggleLayer = useStore(state => state.toggleLayer);
    const toggleRouteVisibility = useStore(state => state.toggleRouteVisibility);

    return (
        <div className="h-full flex flex-col bg-hud-bg w-72 relative z-[1000] border-l border-hud-border shrink-0"
             style={{ boxShadow: '-1px 0 8px rgba(0,0,0,0.5), -1px 0 1px rgba(0, 245, 212, 0.05)' }}>
            
            <div className="scanline-overlay"></div>
            <div className="noise-overlay"></div>

            {/* Header */}
            <div className="relative z-20 p-4 border-b border-hud-border bg-hud-panel/40">
                <h2 className="text-[13px] font-bold text-hud-accent tracking-hud-wide uppercase leading-tight">
                    OPERATIONS
                </h2>
                <p className="text-[9px] text-hud-text-dim tracking-hud uppercase mt-1">
                    MAP LAYER & TARGET CONTROLS
                </p>
            </div>

            <div className="flex-1 overflow-y-auto relative z-20">
                
                {/* Global Layers */}
                <div className="p-4 border-b border-hud-border/50">
                    <h3 className="hud-label mb-3">GLOBAL LAYERS</h3>
                    <div className="space-y-2">
                        <label className="flex items-center gap-3 cursor-pointer group">
                            <div className={`w-3 h-3 border flex items-center justify-center transition-colors ${layerVisibility.zones ? 'bg-hud-accent/20 border-hud-accent' : 'border-hud-border'}`}>
                                {layerVisibility.zones && <div className="w-1.5 h-1.5 bg-hud-accent"></div>}
                            </div>
                            <span className={`text-[10px] tracking-hud uppercase font-mono transition-colors ${layerVisibility.zones ? 'text-hud-text-bright' : 'text-hud-text-dim'}`}>
                                WAYPOINTS [ZONES]
                            </span>
                            <input type="checkbox" className="hidden" checked={layerVisibility.zones} onChange={() => toggleLayer('zones')} />
                        </label>
                        
                        <label className="flex items-center gap-3 cursor-pointer group">
                            <div className={`w-3 h-3 border flex items-center justify-center transition-colors ${layerVisibility.devices ? 'bg-hud-accent/20 border-hud-accent' : 'border-hud-border'}`}>
                                {layerVisibility.devices && <div className="w-1.5 h-1.5 bg-hud-accent"></div>}
                            </div>
                            <span className={`text-[10px] tracking-hud uppercase font-mono transition-colors ${layerVisibility.devices ? 'text-hud-text-bright' : 'text-hud-text-dim'}`}>
                                VECTORS [DEVICES]
                            </span>
                            <input type="checkbox" className="hidden" checked={layerVisibility.devices} onChange={() => toggleLayer('devices')} />
                        </label>
                    </div>
                </div>

                {/* Active Routes */}
                <div className="p-4">
                    <div className="flex items-center justify-between mb-3">
                        <h3 className="hud-label">ACTIVE ROUTES</h3>
                        <span className="text-[9px] text-hud-text-dim tracking-wider font-mono">
                            {routes.length - hiddenRoutes.length} / {routes.length}
                        </span>
                    </div>

                    <div className="space-y-1">
                        {routes.map(route => {
                            const isVisible = !hiddenRoutes.includes(route.id);
                            return (
                                <div 
                                    key={route.id}
                                    onClick={() => toggleRouteVisibility(route.id)}
                                    className={`px-3 py-2 border cursor-pointer transition-all flex items-center justify-between ${isVisible ? 'border-hud-accent/30 bg-hud-panel/30 hover:border-hud-accent/60' : 'border-hud-border/50 bg-transparent hover:bg-hud-panel/10 opacity-50'}`}
                                >
                                    <div className="flex items-center gap-2 min-w-0">
                                        <div className={`w-1.5 h-1.5 rounded-full shrink-0 ${isVisible ? 'bg-hud-accent' : 'bg-hud-border'}`}></div>
                                        <span className={`text-[10px] tracking-hud uppercase font-mono truncate ${isVisible ? 'text-hud-text-bright' : 'text-hud-text-dim'}`}>
                                            {route.name}
                                        </span>
                                    </div>
                                    <span className="text-[8px] text-hud-text-dim ml-2 font-mono">
                                        {isVisible ? 'ON' : 'OFF'}
                                    </span>
                                </div>
                            );
                        })}

                        {routes.length === 0 && (
                            <div className="py-4 text-center">
                                <span className="text-[9px] text-hud-text-dim tracking-hud uppercase">AWAITING ROUTE DATA</span>
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
};
