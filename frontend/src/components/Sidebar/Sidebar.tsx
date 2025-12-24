/**
 * Sidebar
 * Main navigation and info panel
 */

import { useStore } from '../../store';
import { ArrivalsList } from './ArrivalsList';
import { useStops } from '../../hooks/useStops';
import { Search } from 'lucide-react'; // assuming lucide-react is available, user plan mentioned it

export const Sidebar = () => {
    const selectedStopId = useStore(state => state.selectedStopId);
    const setSelectedStopId = useStore(state => state.setSelectedStopId);
    const { stops } = useStops(); // Ensure stops are loaded

    return (
        <div className="h-full flex flex-col bg-white border-r border-gray-200 w-80 shadow-xl z-[1000] relative">
            <div className="p-4 border-b shadow-sm z-10">
                <h1 className="text-xl font-bold text-indigo-600 flex items-center gap-2">
                    🚍 Transit Live
                </h1>

                {/* Simple Search - Future enhancement */}
                <div className="mt-4 relative">
                    <input
                        type="text"
                        placeholder="Search stops..."
                        className="w-full pl-9 pr-3 py-2 bg-gray-100 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    />
                    <Search className="w-4 h-4 text-gray-400 absolute left-3 top-2.5" />
                </div>
            </div>

            <div className="flex-1 overflow-hidden">
                {selectedStopId ? (
                    <div className="h-full flex flex-col">
                        <button
                            onClick={() => setSelectedStopId(null)}
                            className="p-2 text-sm text-blue-600 hover:bg-blue-50 text-left border-b"
                        >
                            ← Back to all stops
                        </button>
                        <ArrivalsList stopId={selectedStopId} />
                    </div>
                ) : (
                    <div className="h-full overflow-y-auto p-2">
                        <h2 className="text-xs font-bold text-gray-400 uppercase tracking-wider mb-2 px-2">Nearby Stops</h2>
                        {stops.map(stop => (
                            <div
                                key={stop.id}
                                onClick={() => setSelectedStopId(stop.id)}
                                className="p-3 hover:bg-gray-50 rounded-lg cursor-pointer transition-colors group"
                            >
                                <div className="font-medium text-gray-800 group-hover:text-indigo-600">{stop.name}</div>
                                <div className="text-xs text-gray-500 truncate">{stop.address}</div>
                            </div>
                        ))}
                    </div>
                )}
            </div>
        </div>
    );
};
