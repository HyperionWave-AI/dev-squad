import { useState, useCallback } from 'react';
import { type ArchiveRequest, type ArchiveResponse } from '../types/chat';

interface UseArchiveMessagesReturn {
  isLoading: boolean;
  error: string | null;
  success: boolean;
  archiveMessages: (request: ArchiveRequest, sessionId: string) => Promise<ArchiveResponse>;
  reset: () => void;
}

/**
 * Hook for archiving messages
 * Handles the API call to archive messages and free up context
 */
export const useArchiveMessages = (): UseArchiveMessagesReturn => {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const archiveMessages = useCallback(
    async (request: ArchiveRequest, sessionId: string): Promise<ArchiveResponse> => {
      try {
        setIsLoading(true);
        setError(null);
        setSuccess(false);

        const response = await fetch(
          `/api/v1/chat/sessions/${sessionId}/archive`,
          {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              messageIds: request.messageIds,
            }),
          }
        );

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.error || `Failed to archive messages: ${response.statusText}`
          );
        }

        const data = await response.json();
        setSuccess(true);

        return {
          success: true,
          archivedCount: data.archivedCount,
          archiveId: data.archiveId || `archive-${Date.now()}`,
          timestamp: new Date().toISOString(),
          message: data.message,
        };
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to archive messages';
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
    archiveMessages,
    reset,
  };
};

export default useArchiveMessages;
