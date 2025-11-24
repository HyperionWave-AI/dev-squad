/**
 * Message Retry Utility
 * 
 * Provides retry logic with exponential backoff for failed message sends.
 * Handles transient errors gracefully with configurable retry strategy.
 * 
 * Usage:
 * const retryConfig = createRetryConfig({
 *   maxAttempts: 3,
 *   initialDelayMs: 1000,
 *   maxDelayMs: 10000,
 * });
 * 
 * await retryWithBackoff(
 *   () => sendMessage(content),
 *   retryConfig
 * );
 */

export interface RetryConfig {
  maxAttempts: number;
  initialDelayMs: number;
  maxDelayMs: number;
  backoffMultiplier: number;
  jitterFactor: number; // 0-1, adds randomness to prevent thundering herd
}

export const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxAttempts: 3,
  initialDelayMs: 1000,
  maxDelayMs: 10000,
  backoffMultiplier: 2,
  jitterFactor: 0.1,
};

/**
 * Create a custom retry configuration
 */
export function createRetryConfig(overrides: Partial<RetryConfig>): RetryConfig {
  return { ...DEFAULT_RETRY_CONFIG, ...overrides };
}

/**
 * Calculate delay with exponential backoff and jitter
 */
function calculateBackoffDelay(
  attempt: number,
  config: RetryConfig
): number {
  const exponentialDelay = Math.min(
    config.initialDelayMs * Math.pow(config.backoffMultiplier, attempt - 1),
    config.maxDelayMs
  );

  // Add jitter to prevent thundering herd
  const jitter = exponentialDelay * config.jitterFactor * Math.random();
  return exponentialDelay + jitter;
}

/**
 * Check if an error is retryable
 */
export function isRetryableError(error: Error): boolean {
  const message = error.message.toLowerCase();

  // Retryable errors
  const retryablePatterns = [
    'timeout',
    'econnrefused',
    'econnreset',
    'ehostunreach',
    'enetunreach',
    'network',
    'temporarily unavailable',
    'service unavailable',
    'too many requests',
    'connection closed',
    'connection refused',
  ];

  return retryablePatterns.some(pattern => message.includes(pattern));
}

/**
 * Retry a function with exponential backoff
 */
export async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  config: RetryConfig = DEFAULT_RETRY_CONFIG,
  onRetry?: (attempt: number, error: Error, nextDelayMs: number) => void
): Promise<T> {
  let lastError: Error | null = null;

  for (let attempt = 1; attempt <= config.maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error instanceof Error ? error : new Error(String(error));

      // Check if error is retryable
      if (!isRetryableError(lastError)) {
        throw lastError;
      }

      // If this was the last attempt, throw
      if (attempt === config.maxAttempts) {
        throw lastError;
      }

      // Calculate delay for next attempt
      const delayMs = calculateBackoffDelay(attempt, config);

      // Call retry callback if provided
      onRetry?.(attempt, lastError, delayMs);

      // Wait before retrying
      await new Promise(resolve => setTimeout(resolve, delayMs));
    }
  }

  // Should never reach here, but just in case
  throw lastError || new Error('Retry failed');
}

/**
 * Retry a function with a maximum timeout
 */
export async function retryWithTimeout<T>(
  fn: () => Promise<T>,
  timeoutMs: number = 30000,
  config: RetryConfig = DEFAULT_RETRY_CONFIG
): Promise<T> {
  const startTime = Date.now();

  return retryWithBackoff(
    async () => {
      const elapsed = Date.now() - startTime;
      if (elapsed > timeoutMs) {
        throw new Error(`Operation timeout after ${elapsed}ms`);
      }
      return fn();
    },
    config
  );
}

/**
 * Create a retry-aware fetch function
 */
export async function fetchWithRetry<T>(
  url: string,
  options?: RequestInit,
  config: RetryConfig = DEFAULT_RETRY_CONFIG
): Promise<T> {
  return retryWithBackoff(
    async () => {
      const response = await fetch(url, options);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      return response.json() as Promise<T>;
    },
    config
  );
}
