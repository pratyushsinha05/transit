/**
 * Stop Details
 * Detailed view of a stop (can be combined with ArrivalsList)
 */

import type { Stop } from '../../types/domain';

interface Props {
    stop: Stop;
}

export const StopDetails = ({ stop }: Props) => {
    return (
        <div className="p-4 bg-white shadow-sm rounded-lg mb-4">
            <h3 className="font-bold">{stop.name}</h3>
            <p className="text-sm text-gray-600">{stop.address}</p>
            {stop.h3Hex && (
                <div className="mt-2 text-xs bg-gray-100 p-1 rounded inline-block">
                    H3: {stop.h3Hex}
                </div>
            )}
        </div>
    );
};
