import { useState, useCallback, useEffect } from 'react';
import { type ContextMetadata, type ContextError } from '../types/chat';

interface UseContextStatusOptions {
  sessionId?: string;
  pollInterval?: number; // in milliseconds, 0 to disable polling
}

interface UseContextStatusReturn {
  contextMetadata: ContextMetadata | null;
  contextError: ContextError | null;
  isLoading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
  startPolling: () => void;
  stopPolling: () => void;
}

/**
 * Hook for managing context status
 * Fetches and polls context usage for a session
 */
export const useContextStatus = (
  options: UseContextStatusOptions = {}
): UseContextStatusReturn => {
  const { sessionId, pollInterval = 5000 } = options;
  const [contextMetadata, setContextMetadata] = useState<ContextMetadata | null>(null);
  const [contextError, setContextError] = useState<ContextError | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pollingEnabled, setPollingEnabled] = useState(false);

  const fetchContextStatus = useCallback(async () => {
    if (!sessionId) {
      setError('Session ID is required');
      return;
    }

    try {
      setIsLoading(true);
      setError(null);

      const response = await fetch(
        `/api/v1/chat/sessions/${sessionId}/context-status`,
        {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to fetch context status: ${response.statusText}`);
      }

      const data = await response.json();
      setContextMetadata(data.contextStatus);

      // Check if we need to show an error
      if (data.contextStatus.percentageUsed >= 90) {
        setContextError({
          code: data.contextStatus.percentageUsed >= 100 ? 'CONTEXT_FULL' : 'CONTEXT_CRITICAL',
          message:
            data.contextStatus.percentageUsed >= 100
              ? 'Context limit reached. Archive or summarize messages immediately.'
              : 'Context usage is critical. Consider archiving or summarizing messages.',
          currentTokens: data.contextStatus.tokenCount,
          maxTokens: data.contextStatus.maxTokens,
          percentageUsed: data.contextStatus.percentageUsed,
          recoveryOptions: ['archive', 'summarize'],
          suggestedAction: 'archive',
          canArchiveMessages: true,
          canSummarize: true,
        });
      } else if (data.contextStatus.percentageUsed >= 80) {
        setContextError({
          code: 'CONTEXT_WARNING',
          message: 'Context usage is at warning level. Consider archiving or summarizing messages.',
          currentTokens: data.contextStatus.tokenCount,
          maxTokens: data.contextStatus.maxTokens,
          percentageUsed: data.contextStatus.percentageUsed,
          recoveryOptions: ['archive', 'summarize'],
          suggestedAction: 'summarize',
          canArchiveMessages: true,
          canSummarize: true,
        });
      } else {
        setContextError(null);
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch context status';
      setError(errorMessage);
      setContextMetadata(null);
    } finally {
      setIsLoading(false);
    }
  }, [sessionId]);

  const startPolling = useCallback(() => {
    setPollingEnabled(true);
  }, []);

  const stopPolling = useCallback(() => {
    setPollingEnabled(false);
  }, []);

  // Set up polling
  useEffect(() => {
    if (!pollingEnabled || !sessionId || pollInterval <= 0) {
      return;
    }

    // Fetch immediately
    fetchContextStatus();

    // Set up interval
    const interval = setInterval(fetchContextStatus, pollInterval);

    return () => clearInterval(interval);
  }, [pollingEnabled, sessionId, pollInterval, fetchContextStatus]);

  return {
    contextMetadata,
    contextError,
    isLoading,
    error,
    refetch: fetchContextStatus,
    startPolling,
    stopPolling,
  };
};

export default useContextStatus;
