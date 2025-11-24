/**
 * Correlation ID Utility
 * 
 * Provides utilities for generating and managing correlation IDs
 * for end-to-end request tracing across frontend and backend.
 * 
 * Usage:
 * - generateCorrelationId(): Generate a new correlation ID
 * - getCorrelationId(): Get current correlation ID from context
 * - setCorrelationId(id): Set correlation ID in context
 * - withCorrelationId(id, fn): Execute function with specific correlation ID
 */

let currentCorrelationId: string | null = null;

/**
 * Generate a new correlation ID (UUID v4)
 */
export function generateCorrelationId(): string {
  return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
}

/**
 * Get the current correlation ID
 */
export function getCorrelationId(): string {
  if (!currentCorrelationId) {
    currentCorrelationId = generateCorrelationId();
  }
  return currentCorrelationId;
}

/**
 * Set the correlation ID
 */
export function setCorrelationId(id: string): void {
  currentCorrelationId = id;
}

/**
 * Execute a function with a specific correlation ID
 */
export async function withCorrelationId<T>(
  id: string,
  fn: () => Promise<T>
): Promise<T> {
  const previousId = currentCorrelationId;
  currentCorrelationId = id;
  try {
    return await fn();
  } finally {
    currentCorrelationId = previousId;
  }
}

/**
 * Add correlation ID to log messages
 */
export function logWithCorrelationId(
  level: 'log' | 'warn' | 'error' | 'info',
  message: string,
  data?: any
): void {
  const correlationId = getCorrelationId();
  const prefix = `[${correlationId}]`;
  
  const logFn = console[level] || console.log;
  if (data) {
    logFn(`${prefix} ${message}`, data);
  } else {
    logFn(`${prefix} ${message}`);
  }
}

/**
 * Add correlation ID to fetch headers
 */
export function addCorrelationIdToHeaders(headers: Record<string, string>): Record<string, string> {
  return {
    ...headers,
    'X-Correlation-ID': getCorrelationId(),
  };
}
