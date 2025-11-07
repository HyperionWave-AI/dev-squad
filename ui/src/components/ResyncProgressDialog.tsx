import React, { useState, useEffect } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { RefreshCw, X, CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import { knowledgeService } from '@/services/knowledgeService';

interface ResyncProgressDialogProps {
  open: boolean;
  onClose: () => void;
}

export const ResyncProgressDialog: React.FC<ResyncProgressDialogProps> = ({ open, onClose }) => {
  const [status, setStatus] = useState<{
    inProgress: boolean;
    totalEntries: number;
    processedEntries: number;
    percentage: number;
    estimatedTimeRemaining?: string;
    errorMessage?: string;
    completedTime?: string;
  } | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setStatus(null);
      setError(null);
      return;
    }

    // Start polling immediately
    let intervalId: ReturnType<typeof setTimeout> | null = null;

    const pollStatus = async () => {
      try {
        const result = await knowledgeService.getResyncStatus();
        setStatus(result);
        setError(null);

        // Stop polling if completed or error
        if (!result.inProgress) {
          if (intervalId) {
            clearInterval(intervalId);
            intervalId = null;
          }
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to get resync status');
        if (intervalId) {
          clearInterval(intervalId);
          intervalId = null;
        }
      }
    };

    // Poll immediately
    pollStatus();

    // Then poll every 2 seconds
    intervalId = setInterval(pollStatus, 2000);

    return () => {
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }, [open]);

  const handleClose = () => {
    // Only allow closing if not in progress or if there's an error
    if (!status?.inProgress || error || status?.errorMessage) {
      onClose();
    }
  };

  const isCompleted = status && !status.inProgress && !status.errorMessage;
  const hasFailed = status?.errorMessage || error;

  return (
    <Dialog.Root open={open} onOpenChange={handleClose}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 data-[state=open]:animate-fade-in z-50" />
        <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-lg data-[state=open]:animate-scale-in z-50">
          {/* Header */}
          <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4 flex items-center justify-between">
            <div className="flex items-center gap-2">
              {isCompleted ? (
                <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400" />
              ) : hasFailed ? (
                <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
              ) : (
                <RefreshCw className="h-5 w-5 text-blue-600 dark:text-blue-400 animate-spin" />
              )}
              <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                {isCompleted ? 'Resync Complete' : hasFailed ? 'Resync Failed' : 'Resyncing...'}
              </Dialog.Title>
            </div>
            <Dialog.Close asChild>
              <button
                className="p-1 rounded-md text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                aria-label="Close"
                disabled={status?.inProgress && !hasFailed}
              >
                <X className="h-5 w-5" />
              </button>
            </Dialog.Close>
          </div>

          {/* Content */}
          <div className="px-6 py-6 space-y-4">
            {!status && !error ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-8 w-8 text-blue-600 dark:text-blue-400 animate-spin" />
              </div>
            ) : error ? (
              <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                <div className="flex items-start gap-3">
                  <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
                  <div>
                    <p className="text-sm font-medium text-red-900 dark:text-red-100">Error</p>
                    <p className="text-sm text-red-800 dark:text-red-200 mt-1">{error}</p>
                  </div>
                </div>
              </div>
            ) : status?.errorMessage ? (
              <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                <div className="flex items-start gap-3">
                  <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
                  <div>
                    <p className="text-sm font-medium text-red-900 dark:text-red-100">Resync Failed</p>
                    <p className="text-sm text-red-800 dark:text-red-200 mt-1">{status.errorMessage}</p>
                  </div>
                </div>
              </div>
            ) : isCompleted ? (
              <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
                <div className="flex items-start gap-3">
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5 flex-shrink-0" />
                  <div>
                    <p className="text-sm font-medium text-green-900 dark:text-green-100">
                      Resync completed successfully!
                    </p>
                    <p className="text-sm text-green-800 dark:text-green-200 mt-1">
                      Processed {status.totalEntries.toLocaleString()} entries
                      {status.completedTime && ` in ${status.completedTime}`}
                    </p>
                  </div>
                </div>
              </div>
            ) : (
              <>
                {/* Progress Stats */}
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                    Processing entries...
                  </span>
                  <span className="text-sm font-semibold text-blue-600 dark:text-blue-400">
                    {status?.percentage.toFixed(1)}%
                  </span>
                </div>

                {/* Progress Bar */}
                <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3 overflow-hidden">
                  <div
                    className="bg-gradient-to-r from-blue-500 to-blue-600 h-full transition-all duration-300 ease-out rounded-full"
                    style={{ width: `${Math.min(100, Math.max(0, status?.percentage || 0))}%` }}
                  />
                </div>

                {/* Entry Count */}
                <div className="flex items-center justify-center gap-2 text-sm">
                  <span className="text-gray-600 dark:text-gray-400">
                    {status?.processedEntries.toLocaleString()} of {status?.totalEntries.toLocaleString()} entries
                  </span>
                </div>

                {/* ETA */}
                {status?.estimatedTimeRemaining && (
                  <div className="flex items-center justify-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                    <span>Estimated time remaining: {status.estimatedTimeRemaining}</span>
                  </div>
                )}

                {/* Info */}
                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-900/50 rounded-lg p-3 mt-4">
                  <p className="text-xs text-blue-800 dark:text-blue-200 text-center">
                    Please wait while the knowledge base is being resynced. This window will update automatically.
                  </p>
                </div>
              </>
            )}
          </div>

          {/* Footer */}
          <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 px-6 py-4 flex justify-end gap-3">
            <button
              onClick={handleClose}
              disabled={status?.inProgress && !hasFailed}
              className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isCompleted || hasFailed ? 'Close' : 'Please wait...'}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
};
