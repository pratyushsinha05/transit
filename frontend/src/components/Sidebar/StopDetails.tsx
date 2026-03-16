/**
 * Stop Details
 * HUD-styled detailed view of a stop
 */

import type { Stop } from '../../types/domain';

interface Props {
    stop: Stop;
}

export const StopDetails = ({ stop }: Props) => {
    return (
        <div className="p-4 border border-hud-border bg-hud-panel/30 rounded-sm hud-glow-border">
            <div className="hud-label mb-1">DESIGNATION</div>
            <h3 className="text-[13px] font-bold text-hud-accent tracking-hud uppercase">{stop.name}</h3>

            {stop.address && (
                <div className="mt-2">
                    <div className="hud-label">SECTOR</div>
                    <p className="text-[10px] text-hud-text tracking-hud uppercase">{stop.address}</p>
                </div>
            )}

            <div className="grid grid-cols-2 gap-2 mt-3 pt-3 border-t border-hud-border">
                <div>
                    <div className="hud-label">LAT</div>
                    <div className="hud-value text-[11px]">{stop.lat.toFixed(6)}</div>
                </div>
                <div>
                    <div className="hud-label">LNG</div>
                    <div className="hud-value text-[11px]">{stop.lng.toFixed(6)}</div>
                </div>
            </div>

            {stop.h3Hex && (
                <div className="mt-2 pt-2 border-t border-hud-border">
                    <div className="hud-label">H3 INDEX</div>
                    <code className="text-[9px] text-hud-accent/60 tracking-wider">{stop.h3Hex}</code>
                </div>
            )}
        </div>
    );
};
