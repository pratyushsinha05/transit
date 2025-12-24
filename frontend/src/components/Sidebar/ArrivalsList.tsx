/**
 * Arrivals List
 * Lists upcoming arrivals for a stop
 */

import { useArrivals } from '../../hooks/useArrivals';
import { ArrivalCard } from './ArrivalCard';
import { useStore } from '../../store';

interface Props {
    stopId: string;
}

export const ArrivalsList = ({ stopId }: Props) => {
    const { arrivals, loading, error, refresh } = useArrivals(stopId);
    const stops = useStore(state => state.stops);
    const stop = stops.find(s => s.id === stopId);

    if (loading && arrivals.length === 0) {
        return <div className="p-4 text-center text-gray-500">Loading arrivals...</div>;
    }

    if (error) {
        return (
            <div className="p-4 text-center text-red-500">
                <p>Error loading arrivals</p>
                <button onClick={refresh} className="mt-2 text-sm underline">Retry</button>
            </div>
        );
    }

    return (
        <div className="flex flex-col h-full">
            <div className="p-4 border-b bg-gray-50">
                <h2 className="font-bold text-lg">{stop?.name || 'Stop Details'}</h2>
                <p className="text-sm text-gray-500">{stop?.address || `ID: ${stopId}`}</p>
                {stop?.h3Hex && (
                    <code className="text-xs text-gray-400 mt-1 block">Hex: {stop.h3Hex}</code>
                )}
            </div>

            <div className="flex-1 overflow-y-auto p-3 bg-gray-50/50">
                {arrivals.length === 0 ? (
                    <div className="text-center text-gray-400 mt-10">
                        No upcoming arrivals found.
                    </div>
                ) : (
                    arrivals.map(arrival => (
                        <ArrivalCard key={arrival.id} arrival={arrival} />
                    ))
                )}
            </div>
        </div>
    );
};
