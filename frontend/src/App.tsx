import { MapContainer } from './components/Map/MapContainer';
import { Sidebar } from './components/Sidebar/Sidebar';
import { OperationsPanel } from './components/Sidebar/OperationsPanel';
import { TelemetryTray } from './components/Footer/TelemetryTray';
import { ConnectionStatus } from './components/Header/ConnectionStatus';
import './styles/globals.css';

function App() {
  return (
    <div className="flex flex-col h-screen w-screen overflow-hidden bg-hud-bg text-hud-text font-mono selection:bg-hud-accent/30">
      
      {/* ── Main Work Area ── */}
      <div className="flex flex-1 overflow-hidden relative">
        
        {/* Left Panel: Navigation & Targets */}
        <Sidebar />

        {/* Center: Live Map view */}
        <div className="flex-1 relative bg-[#0a0e1a]">
          <MapContainer />

          {/* Floating Controls Overlay */}
          <div className="absolute top-4 right-4 z-[1000] pointer-events-none">
            <div className="pointer-events-auto">
              <ConnectionStatus />
            </div>
          </div>
        </div>

        {/* Right Panel: Operations & Layers */}
        <OperationsPanel />
      
      </div>

      {/* ── Bottom Telemetry ── */}
      <TelemetryTray />

    </div>
  );
}

export default App;
