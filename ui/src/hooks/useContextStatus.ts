import { useState, useCallback, useEffect, useRef } from 'react';
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
  
  // Use ref to track current sessionId for fetch operations
  const sessionIdRef = useRef<string | undefined>(sessionId);
  
  // Update ref when sessionId changes
  useEffect(() => {
    sessionIdRef.current = sessionId;
  }, [sessionId]);

  const fetchContextStatus = useCallback(async () => {
    // Use ref to get current sessionId instead of closure
    const currentSessionId = sessionIdRef.current;
    
    if (!currentSessionId) {
      setError('Session ID is required');
      return;
    }

    try {
      setIsLoading(true);
      setError(null);

      const response = await fetch(
        `/api/v1/chat/sessions/${currentSessionId}/context-status`,
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

      // Map backend response to frontend ContextMetadata type
      const backendData = data.contextStatus;
      const mappedMetadata: ContextMetadata = {
        tokenCount: backendData.totalTokens || 0,
        maxTokens: backendData.maxTokens || 100000,
        percentageUsed: backendData.percentageUsed || 0,
        isWarning: backendData.isWarning || false,
        isCritical: backendData.isCritical || false,
        messageCount: backendData.messageCount || 0,
        lastUpdated: backendData.lastUpdated || new Date().toISOString(),
        canAddMessage: !backendData.isCritical,
      };

      // Only update state if values actually changed to prevent unnecessary re-renders
      setContextMetadata(prev => {
        if (!prev ||
            prev.tokenCount !== mappedMetadata.tokenCount ||
            prev.percentageUsed !== mappedMetadata.percentageUsed ||
            prev.messageCount !== mappedMetadata.messageCount ||
            prev.isCritical !== mappedMetadata.isCritical ||
            prev.isWarning !== mappedMetadata.isWarning) {
          return mappedMetadata;
        }
        return prev;
      });

      // Check if we need to show an error - only update if error code changes
      const newErrorCode = mappedMetadata.percentageUsed >= 100 ? 'CONTEXT_FULL'
                         : mappedMetadata.percentageUsed >= 90 ? 'CONTEXT_CRITICAL'
                         : mappedMetadata.percentageUsed >= 80 ? 'CONTEXT_WARNING'
                         : null;

      setContextError(prev => {
        // If error code hasn't changed, keep previous error (prevents re-renders)
        if (prev?.code === newErrorCode && newErrorCode !== null) {
          return prev;
        }

        // Create new error object only if code changed
        if (newErrorCode === 'CONTEXT_FULL' || newErrorCode === 'CONTEXT_CRITICAL') {
          return {
            code: newErrorCode,
            message: newErrorCode === 'CONTEXT_FULL'
              ? 'Context limit reached. Archive or summarize messages immediately.'
              : 'Context usage is critical. Consider archiving or summarizing messages.',
            currentTokens: mappedMetadata.tokenCount,
            maxTokens: mappedMetadata.maxTokens,
            percentageUsed: mappedMetadata.percentageUsed,
            recoveryOptions: ['archive', 'summarize'],
            suggestedAction: 'archive',
            canArchiveMessages: true,
            canSummarize: true,
          };
        } else if (newErrorCode === 'CONTEXT_WARNING') {
          return {
            code: 'CONTEXT_WARNING',
            message: 'Context usage is at warning level. Consider archiving or summarizing messages.',
            currentTokens: mappedMetadata.tokenCount,
            maxTokens: mappedMetadata.maxTokens,
            percentageUsed: mappedMetadata.percentageUsed,
            recoveryOptions: ['archive', 'summarize'],
            suggestedAction: 'summarize',
            canArchiveMessages: true,
            canSummarize: true,
          };
        }
        return null;
      });
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch context status';
      setError(errorMessage);
      setContextMetadata(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const startPolling = useCallback(() => {
    setPollingEnabled(true);
  }, []);

  const stopPolling = useCallback(() => {
    setPollingEnabled(false);
  }, []);

  // Set up polling
  useEffect(() => {
    if (!pollingEnabled || !sessionIdRef.current || pollInterval <= 0) {
      return;
    }

    // Fetch immediately
    fetchContextStatus();

    // Set up interval
    const interval = setInterval(fetchContextStatus, pollInterval);

    return () => clearInterval(interval);
  }, [pollingEnabled, pollInterval, fetchContextStatus]);

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
