/**
 * H3 Utilities
 * Work with H3 hexagon data from backend
 */

import type { BusLocation } from '../types/domain';

/**
 * NOTE: Frontend RECEIVES h3_hex from backend
 * Frontend does NOT calculate H3 - that's backend's job
 * 
 * H3 data used for:
 * - Geospatial grouping visualization
 * - Performance optimization (filtering by hex level)
 * - Debug information
 * - Future: clustering by hex
 */

export interface H3Location {
    h3Hex: string;        // From backend, e.g., "8a2843b28dbffff"
    busId: string;
    lat: number;
    lng: number;
    timestamp: Date;
}

/**
 * Group buses by H3 hex level
 * Useful for showing "buses in this area" clustering
 */
export const groupBusesByH3 = (buses: Map<string, BusLocation>) => {
    const groups = new Map<string, BusLocation[]>();

    // NOTE: We don't calculate H3 ourselves
    // We use the h3Hex already in bus data from backend
    for (const [, bus] of buses) {
        if (bus.h3Hex) {
            if (!groups.has(bus.h3Hex)) {
                groups.set(bus.h3Hex, []);
            }
            groups.get(bus.h3Hex)!.push(bus);
        }
    }

    return groups;
};

/**
 * Filter buses in specific H3 hex
 * Backend already handles geofencing, this is just for display
 */
export const getBusesInHex = (buses: Map<string, BusLocation>, h3Hex: string) => {
    const result: BusLocation[] = [];

    for (const [, bus] of buses) {
        if (bus.h3Hex === h3Hex) {
            result.push(bus);
        }
    }

    return result;
};

/**
 * Debug: Show H3 hex information for a bus
 */
export const debugH3Info = (bus: BusLocation) => {
    /* eslint-disable no-console */
    console.group(`H3 Info for Bus ${bus.id}`);
    console.log('H3 Hex:', bus.h3Hex);
    console.log('Location:', `${bus.lat}, ${bus.lng}`);
    console.log('Last Update:', bus.lastUpdate);
    console.log('Speed:', `${bus.speed} km/h`);
    console.groupEnd();
    /* eslint-enable no-console */
};
