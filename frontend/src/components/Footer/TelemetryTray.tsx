/**
 * Telemetry Tray
 * Persistent bottom bar with system clock, event ticker, and throughput stats
 */

import { useEffect, useState } from 'react';
import { useStore } from '../../store';

export const TelemetryTray = () => {
    const [time, setTime] = useState(new Date());
    const connection = useStore(state => state.connection);
    const isConnected = connection.status === 'connected';

    // Simulated event log
    const events = [
        '[SYS] TRANSIT KERNEL ONLINE',
        '[NET] WEBSOCKET SECURE LINK ESTABLISHED',
        '[DAT] ACQUIRING LIVE FLEET TELEMETRY...',
        '[GEO] POSTGIS SPATIAL INDEX READY',
        '[OPS] AWAITING TARGET VECTORS',
    ];

    useEffect(() => {
        const timer = setInterval(() => setTime(new Date()), 1000);
        return () => clearInterval(timer);
    }, []);

    return (
        <div className="h-12 border-t border-hud-border bg-hud-bg flex items-center px-4 relative z-[2000] shrink-0"
            style={{ boxShadow: '0 -2px 10px rgba(0,0,0,0.5)' }}>

            <div className="scanline-overlay"></div>

            {/* Left: System Status */}
            <div className="flex items-center gap-4 shrink-0 pr-6 border-r border-hud-border/50 h-full">
                <div className="flex items-center gap-2">
                    <span className={`w-1.5 h-1.5 rounded-sm ${isConnected ? 'bg-hud-accent' : 'bg-hud-danger animate-blink'}`}></span>
                    <span className="text-[10px] font-bold text-hud-text tracking-hud-wide uppercase">
                        SYS.STATE
                    </span>
                </div>
                <div className="text-[10px] text-hud-accent tracking-wider font-mono">
                    {isConnected ? 'NOMINAL' : 'CRITICAL'}
                </div>
            </div>

            {/* Center: Event Ticker */}
            <div className="flex-1 overflow-hidden px-6 flex items-center h-full">
                <div className="flex gap-12 animate-[marquee_40s_linear_infinite] whitespace-nowrap">
                    {events.map((evt, i) => (
                        <span key={i} className="text-[10px] text-hud-text-dim tracking-hud uppercase font-mono">
                            {evt}
                        </span>
                    ))}
                    {/* Duplicate for infinite seamless scroll */}
                    {events.map((evt, i) => (
                        <span key={`dup-${i}`} className="text-[10px] text-hud-text-dim tracking-hud uppercase font-mono">
                            {evt}
                        </span>
                    ))}
                </div>
            </div>

            {/* Right: Telemetry & Clock */}
            <div className="flex items-center gap-6 shrink-0 pl-6 border-l border-hud-border/50 h-full">
                <div className="flex flex-col justify-center">
                    <span className="text-[8px] text-hud-text-dim tracking-wide uppercase">NET.THROUGHPUT</span>
                    <span className="text-[10px] text-hud-accent tracking-wider font-mono">
                        {isConnected ? (Math.random() * 2 + 1).toFixed(2) + ' KB/S' : '0.00 KB/S'}
                    </span>
                </div>

                <div className="flex flex-col justify-center items-end min-w-[100px]">
                    <span className="text-[8px] text-hud-text-dim tracking-wide uppercase">GLOBAL.TIME [UTC]</span>
                    <span className="text-[12px] font-bold text-hud-text-bright tracking-wider font-mono">
                        {time.toISOString().substring(11, 19)}
                    </span>
                </div>
            </div>

            {/* Required CSS for the ticker animation */}
            <style>{`
                @keyframes marquee {
                    0% { transform: translateX(0%); }
                    100% { transform: translateX(-50%); }
                }
            `}</style>
        </div>
    );
};
