/**
 * Error Handler
 * Central processing for application errors
 */

import { logger } from './logger';

export interface AppError {
    code: string;
    message: string;
    originalError?: any;
    context?: object;
    timestamp: Date;
}

export const handleError = (error: any, context?: object) => {
    const appError = normalizeError(error, context);

    // Log error
    logger.error(appError.message, { ...appError, context });

    // Connect to UI notification system (future: dispatch to store)
    // For now, we rely on UI components catching/displaying or store actions handling it

    return appError;
};

const normalizeError = (error: any, context?: object): AppError => {
    if (error instanceof Error) {
        return {
            code: 'UNKNOWN_ERROR',
            message: error.message,
            originalError: error,
            context,
            timestamp: new Date(),
        };
    }

    if (typeof error === 'string') {
        return {
            code: 'GENERIC_ERROR',
            message: error,
            context,
            timestamp: new Date(),
        };
    }

    return {
        code: 'UNKNOWN_ObJECT',
        message: 'An unknown error occurred',
        originalError: error,
        context,
        timestamp: new Date(),
    };
};
