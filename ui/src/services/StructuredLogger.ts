/**
 * Structured Logger Utility
 * 
 * Provides structured logging with correlation IDs and consistent formatting.
 * Logs are output in JSON format for easy parsing and aggregation.
 * 
 * Features:
 * - Structured JSON logging
 * - Correlation ID tracking
 * - Log levels (debug, info, warn, error)
 * - Request/response logging
 * - Performance metrics logging
 * - Log filtering and searching
 */

import { getCorrelationId } from '@/utils/correlationId';

export const LogLevel = {
  DEBUG: 'DEBUG',
  INFO: 'INFO',
  WARN: 'WARN',
  ERROR: 'ERROR',
} as const;

export type LogLevel = typeof LogLevel[keyof typeof LogLevel];

export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  correlationId: string;
  message: string;
  context?: Record<string, any>;
  duration?: number;
  error?: {
    message: string;
    stack?: string;
  };
}

export class StructuredLogger {
  private logs: LogEntry[] = [];
  private readonly MAX_LOG_SIZE = 1000;
  private minLogLevel: LogLevel = LogLevel.DEBUG;

  constructor(minLogLevel: LogLevel = LogLevel.DEBUG) {
    this.minLogLevel = minLogLevel;
  }

  /**
   * Log a debug message
   */
  debug(message: string, context?: Record<string, any>): void {
    this.log(LogLevel.DEBUG, message, context);
  }

  /**
   * Log an info message
   */
  info(message: string, context?: Record<string, any>): void {
    this.log(LogLevel.INFO, message, context);
  }

  /**
   * Log a warning message
   */
  warn(message: string, context?: Record<string, any>): void {
    this.log(LogLevel.WARN, message, context);
  }

  /**
   * Log an error message
   */
  error(message: string, error?: Error, context?: Record<string, any>): void {
    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level: LogLevel.ERROR,
      correlationId: getCorrelationId(),
      message,
      context,
      error: error ? {
        message: error.message,
        stack: error.stack,
      } : undefined,
    };

    this.addLog(entry);
    console.error(`[${entry.correlationId}] ${message}`, {
      error: error?.message,
      context,
    });
  }

  /**
   * Log a request
   */
  logRequest(
    method: string,
    url: string,
    context?: Record<string, any>
  ): void {
    this.info(`${method} ${url}`, {
      type: 'request',
      method,
      url,
      ...context,
    });
  }

  /**
   * Log a response
   */
  logResponse(
    method: string,
    url: string,
    status: number,
    durationMs: number,
    context?: Record<string, any>
  ): void {
    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level: status >= 400 ? LogLevel.WARN : LogLevel.INFO,
      correlationId: getCorrelationId(),
      message: `${method} ${url} ${status}`,
      duration: durationMs,
      context: {
        type: 'response',
        method,
        url,
        status,
        ...context,
      },
    };

    this.addLog(entry);
    console.log(`[${entry.correlationId}] ${method} ${url} ${status} (${durationMs}ms)`);
  }

  /**
   * Log a performance metric
   */
  logMetric(
    name: string,
    value: number,
    unit: string = 'ms',
    context?: Record<string, any>
  ): void {
    this.info(`Metric: ${name}`, {
      type: 'metric',
      name,
      value,
      unit,
      ...context,
    });
  }

  /**
   * Log a state change
   */
  logStateChange(
    component: string,
    previousState: any,
    newState: any,
    context?: Record<string, any>
  ): void {
    this.debug(`State change in ${component}`, {
      type: 'state_change',
      component,
      previousState,
      newState,
      ...context,
    });
  }

  /**
   * Get all logs
   */
  getLogs(): LogEntry[] {
    return [...this.logs];
  }

  /**
   * Filter logs by level
   */
  filterByLevel(level: LogLevel): LogEntry[] {
    return this.logs.filter(log => log.level === level);
  }

  /**
   * Filter logs by correlation ID
   */
  filterByCorrelationId(correlationId: string): LogEntry[] {
    return this.logs.filter(log => log.correlationId === correlationId);
  }

  /**
   * Filter logs by time range
   */
  filterByTimeRange(startTime: Date, endTime: Date): LogEntry[] {
    const startMs = startTime.getTime();
    const endMs = endTime.getTime();

    return this.logs.filter(log => {
      const logTime = new Date(log.timestamp).getTime();
      return logTime >= startMs && logTime <= endMs;
    });
  }

  /**
   * Search logs by message
   */
  search(query: string): LogEntry[] {
    const lowerQuery = query.toLowerCase();
    return this.logs.filter(log =>
      log.message.toLowerCase().includes(lowerQuery)
    );
  }

  /**
   * Get log statistics
   */
  getStats(): {
    total: number;
    byLevel: Record<string, number>;
    averageDuration: number;
    errorCount: number;
  } {
    const byLevel: Record<string, number> = {};
    let totalDuration = 0;
    let durationCount = 0;
    let errorCount = 0;

    for (const log of this.logs) {
      byLevel[log.level] = (byLevel[log.level] || 0) + 1;

      if (log.duration !== undefined) {
        totalDuration += log.duration;
        durationCount++;
      }

      if (log.error) {
        errorCount++;
      }
    }

    return {
      total: this.logs.length,
      byLevel,
      averageDuration: durationCount > 0 ? totalDuration / durationCount : 0,
      errorCount,
    };
  }

  /**
   * Clear logs
   */
  clear(): void {
    this.logs = [];
  }

  /**
   * Export logs as JSON
   */
  exportAsJSON(): string {
    return JSON.stringify(this.logs, null, 2);
  }

  /**
   * Export logs as CSV
   */
  exportAsCSV(): string {
    if (this.logs.length === 0) {
      return '';
    }

    const headers = ['timestamp', 'level', 'correlationId', 'message', 'duration', 'error'];
    const rows = this.logs.map(log => [
      log.timestamp,
      log.level,
      log.correlationId,
      log.message,
      log.duration || '',
      log.error?.message || '',
    ]);

    const csvContent = [
      headers.join(','),
      ...rows.map(row => row.map(cell => `"${cell}"`).join(',')),
    ].join('\n');

    return csvContent;
  }

  // Private helper methods

  private log(level: LogLevel, message: string, context?: Record<string, any>): void {
    // Check if this log level should be logged
    if (!this.shouldLog(level)) {
      return;
    }

    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level,
      correlationId: getCorrelationId(),
      message,
      context,
    };

    this.addLog(entry);

    // Also log to console
    const logFn = console[level.toLowerCase() as keyof typeof console] || console.log;
    if (typeof logFn === 'function') {
      if (context) {
        (logFn as (msg: string, ctx: any) => void)(`[${entry.correlationId}] ${message}`, context);
      } else {
        (logFn as (msg: string) => void)(`[${entry.correlationId}] ${message}`);
      }
    }
  }

  private addLog(entry: LogEntry): void {
    this.logs.push(entry);

    // Keep logs bounded
    if (this.logs.length > this.MAX_LOG_SIZE) {
      this.logs.shift();
    }
  }

  private shouldLog(level: LogLevel): boolean {
    const levels = [LogLevel.DEBUG, LogLevel.INFO, LogLevel.WARN, LogLevel.ERROR];
    const minIndex = levels.indexOf(this.minLogLevel);
    const currentIndex = levels.indexOf(level);
    return currentIndex >= minIndex;
  }
}

/**
 * Create a singleton instance
 */
let instance: StructuredLogger | null = null;

export function getLogger(): StructuredLogger {
  if (!instance) {
    instance = new StructuredLogger();
  }
  return instance;
}
