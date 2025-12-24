/**
 * Logger Service
 * Centralized logging for the application
 */

/* eslint-disable no-console */

type LogLevel = 'debug' | 'info' | 'warn' | 'error';

class Logger {
    private level: LogLevel = 'debug';

    constructor() {
        // Determine log level from env if needed
    }

    debug(message: string, context?: object) {
        if (this.shouldLog('debug')) {
            console.debug(`[DEBUG] ${message}`, context || '');
        }
    }

    info(message: string, context?: object) {
        if (this.shouldLog('info')) {
            console.info(`[INFO] ${message}`, context || '');
        }
    }

    warn(message: string, context?: object) {
        if (this.shouldLog('warn')) {
            console.warn(`[WARN] ${message}`, context || '');
        }
    }

    error(message: string, context?: object) {
        if (this.shouldLog('error')) {
            console.error(`[ERROR] ${message}`, context || '');
        }
    }

    private shouldLog(level: LogLevel): boolean {
        const levels: LogLevel[] = ['debug', 'info', 'warn', 'error'];
        const currentIdx = levels.indexOf(this.level);
        const msgIdx = levels.indexOf(level);
        return msgIdx >= currentIdx;
    }
}

export const logger = new Logger();
