import { MapContainer } from './components/Map/MapContainer';
import { Sidebar } from './components/Sidebar/Sidebar';
import { ConnectionStatus } from './components/Header/ConnectionStatus';
import './styles/globals.css';

function App() {
  return (
    <div className="flex h-screen w-screen overflow-hidden bg-gray-100">
      {/* Sidebar */}
      <Sidebar />

      {/* Main Content (Map) */}
      <div className="flex-1 relative">
        <MapContainer />

        {/* Floating Header / Status */}
        <div className="absolute top-4 right-4 z-[1000]">
          <ConnectionStatus />
        </div>
      </div>
    </div>
  );
}

export default App;
