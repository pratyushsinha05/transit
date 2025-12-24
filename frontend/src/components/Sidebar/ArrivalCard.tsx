/**
 * Arrival Card
 * Displays a single arrival prediction
 */

import type { Arrival } from '../../types/domain';
import clsx from 'clsx';
import { formatDistanceToNow } from 'date-fns';

interface Props {
    arrival: Arrival;
}

export const ArrivalCard = ({ arrival }: Props) => {
    const statusColors = {
        on_time: 'text-green-600',
        delayed: 'text-red-500',
        arriving: 'text-blue-600 animate-pulse',
        cancelled: 'text-gray-400 line-through',
        scheduled: 'text-gray-600',
    };

    return (
        <div className="p-3 bg-white rounded-lg shadow-sm border border-gray-100 flex justify-between items-center mb-2">
            <div>
                <div className="flex items-center gap-2">
                    <span className="font-bold text-lg">{arrival.route}</span>
                    <span className={clsx('text-xs font-medium uppercase px-1.5 py-0.5 rounded bg-gray-100', statusColors[arrival.status])}>
                        {arrival.status.replace('_', ' ')}
                    </span>
                </div>
                <div className="text-xs text-gray-500 mt-1">
                    Bus #{arrival.busId}
                </div>
            </div>

            <div className="text-right">
                <div className="text-2xl font-bold text-gray-800">
                    {arrival.eta}<span className="text-xs font-normal text-gray-500 ml-1">min</span>
                </div>
                <div className="text-[10px] text-gray-400">
                    Updated {formatDistanceToNow(arrival.timestamp, { addSuffix: true })}
                </div>
            </div>
        </div>
    );
};
