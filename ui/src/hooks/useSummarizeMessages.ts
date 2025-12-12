import { useState, useCallback } from 'react';

interface SummarizeResponse {
  success: boolean;
  message: string;
  summaryMessage: {
    id: string;
    content: string;
    role: 'system';
    timestamp: string;
  };
  summarizedCount: number;
  contextStatus: {
    tokenCount: number;
    percentageUsed: number;
  };
}

interface UseSummarizeMessagesReturn {
  isLoading: boolean;
  error: string | null;
  success: boolean;
  summarizeMessages: (
    sessionId: string,
    keepRecentMinutes: number
  ) => Promise<SummarizeResponse>;
  reset: () => void;
}

/**
 * Hook for summarizing messages
 * Handles the API call to summarize old messages and free up context
 */
export const useSummarizeMessages = (): UseSummarizeMessagesReturn => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const summarizeMessages = useCallback(
    async (sessionId: string, keepRecentMinutes: number): Promise<SummarizeResponse> => {
      try {
        setIsLoading(true);
        setError(null);
        setSuccess(false);

        const response = await fetch(
          `/api/v1/chat/sessions/${sessionId}/summarize`,
          {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              keepRecentMinutes,
            }),
          }
        );

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.error || `Failed to summarize messages: ${response.statusText}`
          );
        }

        const data = await response.json();
        setSuccess(true);

        return {
          success: true,
          message: data.message,
          summaryMessage: data.summaryMessage,
          summarizedCount: data.summarizedCount,
          contextStatus: data.contextStatus,
        };
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to summarize messages';
        setError(errorMessage);
        throw err;
      } finally {
        setIsLoading(false);
      }
    },
    []
  );

  const reset = useCallback(() => {
    setError(null);
    setSuccess(false);
  }, []);

  return {
    isLoading,
    error,
    success,
    summarizeMessages,
    reset,
  };
};

export default useSummarizeMessages;
