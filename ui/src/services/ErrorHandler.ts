/**
 * Error Handler Utility
 * 
 * Standardizes error handling across the chat system.
 * Provides error type hierarchy, recovery strategies, and tracking.
 * 
 * Features:
 * - Error type classification (Network, Validation, Auth, etc.)
 * - Retryable vs permanent error detection
 * - User-friendly error messages
 * - Error recovery suggestions
 * - Error tracking and analytics
 */

export const ErrorType = {
  NETWORK: 'NETWORK',
  VALIDATION: 'VALIDATION',
  AUTHENTICATION: 'AUTHENTICATION',
  AUTHORIZATION: 'AUTHORIZATION',
  NOT_FOUND: 'NOT_FOUND',
  CONFLICT: 'CONFLICT',
  RATE_LIMIT: 'RATE_LIMIT',
  SERVER_ERROR: 'SERVER_ERROR',
  TIMEOUT: 'TIMEOUT',
  UNKNOWN: 'UNKNOWN',
} as const;

export type ErrorType = typeof ErrorType[keyof typeof ErrorType];

export interface ChatError {
  type: ErrorType;
  message: string;
  userMessage: string;
  recoveryAction?: string;
  isRetryable: boolean;
  originalError?: Error;
  timestamp: Date;
  context?: Record<string, any>;
}

export class ErrorHandler {
  private errors: ChatError[] = [];
  private readonly MAX_ERROR_HISTORY = 100;

  /**
   * Classify an error
   */
  classifyError(error: Error | string): ErrorType {
    const message = typeof error === 'string' ? error : error.message;
    const lowerMessage = message.toLowerCase();

    if (
      lowerMessage.includes('network') ||
      lowerMessage.includes('fetch') ||
      lowerMessage.includes('connection') ||
      lowerMessage.includes('econnrefused') ||
      lowerMessage.includes('econnreset')
    ) {
      return ErrorType.NETWORK;
    }

    if (
      lowerMessage.includes('validation') ||
      lowerMessage.includes('invalid') ||
      lowerMessage.includes('required')
    ) {
      return ErrorType.VALIDATION;
    }

    if (
      lowerMessage.includes('unauthorized') ||
      lowerMessage.includes('unauthenticated') ||
      lowerMessage.includes('401')
    ) {
      return ErrorType.AUTHENTICATION;
    }

    if (
      lowerMessage.includes('forbidden') ||
      lowerMessage.includes('permission') ||
      lowerMessage.includes('403')
    ) {
      return ErrorType.AUTHORIZATION;
    }

    if (lowerMessage.includes('404') || lowerMessage.includes('not found')) {
      return ErrorType.NOT_FOUND;
    }

    if (lowerMessage.includes('409') || lowerMessage.includes('conflict')) {
      return ErrorType.CONFLICT;
    }

    if (
      lowerMessage.includes('429') ||
      lowerMessage.includes('rate limit') ||
      lowerMessage.includes('too many requests')
    ) {
      return ErrorType.RATE_LIMIT;
    }

    if (
      lowerMessage.includes('500') ||
      lowerMessage.includes('502') ||
      lowerMessage.includes('503') ||
      lowerMessage.includes('server error')
    ) {
      return ErrorType.SERVER_ERROR;
    }

    if (
      lowerMessage.includes('timeout') ||
      lowerMessage.includes('timed out')
    ) {
      return ErrorType.TIMEOUT;
    }

    return ErrorType.UNKNOWN;
  }

  /**
   * Check if an error is retryable
   */
  isRetryable(error: Error | string): boolean {
    const type = this.classifyError(error);

    const retryableTypes: ErrorType[] = [
      ErrorType.NETWORK,
      ErrorType.TIMEOUT,
      ErrorType.RATE_LIMIT,
      ErrorType.SERVER_ERROR,
    ];

    return retryableTypes.includes(type);
  }

  /**
   * Get user-friendly error message
   */
  getUserMessage(error: Error | string): string {
    const type = this.classifyError(error);

    const messages: Record<ErrorType, string> = {
      [ErrorType.NETWORK]: 'Network connection failed. Please check your internet connection.',
      [ErrorType.VALIDATION]: 'Invalid input. Please check your message and try again.',
      [ErrorType.AUTHENTICATION]: 'Authentication failed. Please log in again.',
      [ErrorType.AUTHORIZATION]: 'You do not have permission to perform this action.',
      [ErrorType.NOT_FOUND]: 'The requested resource was not found.',
      [ErrorType.CONFLICT]: 'A conflict occurred. Please refresh and try again.',
      [ErrorType.RATE_LIMIT]: 'Too many requests. Please wait a moment and try again.',
      [ErrorType.SERVER_ERROR]: 'Server error. Please try again later.',
      [ErrorType.TIMEOUT]: 'Request timed out. Please try again.',
      [ErrorType.UNKNOWN]: 'An unexpected error occurred. Please try again.',
    };

    return messages[type];
  }

  /**
   * Get recovery action suggestion
   */
  getRecoveryAction(error: Error | string): string | undefined {
    const type = this.classifyError(error);

    const actions: Record<ErrorType, string | undefined> = {
      [ErrorType.NETWORK]: 'Check your internet connection and try again',
      [ErrorType.VALIDATION]: 'Review your input and try again',
      [ErrorType.AUTHENTICATION]: 'Log in again',
      [ErrorType.AUTHORIZATION]: 'Contact support for access',
      [ErrorType.NOT_FOUND]: 'Refresh the page and try again',
      [ErrorType.CONFLICT]: 'Refresh the page and try again',
      [ErrorType.RATE_LIMIT]: 'Wait a moment and try again',
      [ErrorType.SERVER_ERROR]: 'Try again later',
      [ErrorType.TIMEOUT]: 'Try again',
      [ErrorType.UNKNOWN]: 'Try again',
    };

    return actions[type];
  }

  /**
   * Create a structured error
   */
  createError(
    error: Error | string,
    context?: Record<string, any>
  ): ChatError {
    const type = this.classifyError(error);
    const originalError = error instanceof Error ? error : undefined;
    const message = typeof error === 'string' ? error : error.message;

    const chatError: ChatError = {
      type,
      message,
      userMessage: this.getUserMessage(error),
      recoveryAction: this.getRecoveryAction(error),
      isRetryable: this.isRetryable(error),
      originalError,
      timestamp: new Date(),
      context,
    };

    // Track error
    this.trackError(chatError);

    return chatError;
  }

  /**
   * Track an error for analytics
   */
  trackError(error: ChatError): void {
    this.errors.push(error);

    // Keep history bounded
    if (this.errors.length > this.MAX_ERROR_HISTORY) {
      this.errors.shift();
    }

    // Log error
    console.error(`[${error.type}] ${error.message}`, {
      userMessage: error.userMessage,
      recoveryAction: error.recoveryAction,
      context: error.context,
    });
  }

  /**
   * Get error history
   */
  getErrorHistory(): ChatError[] {
    return [...this.errors];
  }

  /**
   * Get error statistics
   */
  getErrorStats(): {
    total: number;
    byType: Record<string, number>;
    retryableCount: number;
    recentErrors: ChatError[];
  } {
    const byType: Record<string, number> = {};
    let retryableCount = 0;

    for (const error of this.errors) {
      byType[error.type] = (byType[error.type] || 0) + 1;
      if (error.isRetryable) {
        retryableCount++;
      }
    }

    return {
      total: this.errors.length,
      byType,
      retryableCount,
      recentErrors: this.errors.slice(-10),
    };
  }

  /**
   * Clear error history
   */
  clearHistory(): void {
    this.errors = [];
  }
}

/**
 * Create a singleton instance
 */
let instance: ErrorHandler | null = null;

export function getErrorHandler(): ErrorHandler {
  if (!instance) {
    instance = new ErrorHandler();
  }
  return instance;
}
