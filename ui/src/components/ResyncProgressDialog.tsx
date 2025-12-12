import React, { useState, useEffect } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { RefreshCw, X, CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import { knowledgeService } from '@/services/knowledgeService';
import { ProgressTracker, type TrackableEvent } from './organisms/ProgressTracker';

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
  const [progressEvents, setProgressEvents] = useState<TrackableEvent[]>([]);

  useEffect(() => {
    if (!open) {
      setStatus(null);
      setError(null);
      setProgressEvents([]);
      return;
    }

    // Start polling immediately
    let intervalId: ReturnType<typeof setTimeout> | null = null;

    const pollStatus = async () => {
      try {
        const result = await knowledgeService.getResyncStatus();
        setStatus(result);
        setError(null);

        // Convert status to progress events
        const events: TrackableEvent[] = [];
        
        if (result.inProgress) {
          // Add typing indicator if just started
          if (result.processedEntries === 0) {
            events.push({
              id: 'resync-typing',
              type: 'typing',
              description: 'Initializing resync...',
              timestamp: new Date(),
            });
          }
          
          // Add progress event
          events.push({
            id: 'resync-progress',
            type: 'progress',
            step: result.processedEntries,
            totalSteps: result.totalEntries,
            description: `Processing knowledge base entries (${result.processedEntries.toLocaleString()}/${result.totalEntries.toLocaleString()})`,
            status: 'in_progress',
            timestamp: new Date(),
          });
        } else {
          // Completed - add final progress event
          events.push({
            id: 'resync-progress',
            type: 'progress',
            step: result.totalEntries,
            totalSteps: result.totalEntries,
            description: result.errorMessage 
              ? `Resync failed: ${result.errorMessage}`
              : `Successfully processed ${result.totalEntries.toLocaleString()} entries`,
            status: result.errorMessage ? 'error' : 'completed',
            timestamp: new Date(),
          });
        }

        setProgressEvents(events);

        // Stop polling if completed or error
        if (!result.inProgress) {
          if (intervalId) {
            clearInterval(intervalId);
            intervalId = null;
          }
        }
      } catch (err) {
        const errorMessage = err instanceof Error ? err.message : 'Failed to get resync status';
        setError(errorMessage);
        
        // Add error event
        setProgressEvents([{
          id: 'resync-error',
          type: 'progress',
          step: 0,
          totalSteps: 1,
          description: `Resync failed: ${errorMessage}`,
          status: 'error',
          timestamp: new Date(),
        }]);
        
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
                {isCompleted ? 'Resync Complete' : hasFailed ? 'Resync Failed' : 'Resyncing Knowledge Base'}
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
                {/* Enhanced Progress Display using ProgressTracker */}
                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-900/50 rounded-lg p-4">
                  <div className="flex items-center justify-between mb-3">
                    <span className="text-sm font-medium text-blue-900 dark:text-blue-100">
                      Resync Progress
                    </span>
                    <span className="text-sm font-semibold text-blue-600 dark:text-blue-400">
                      {status?.percentage.toFixed(1)}%
                    </span>
                  </div>

                  {/* Progress Bar */}
                  <div className="w-full bg-blue-100 dark:bg-blue-800/30 rounded-full h-3 overflow-hidden mb-3">
                    <div
                      className="bg-gradient-to-r from-blue-500 to-blue-600 h-full transition-all duration-300 ease-out rounded-full"
                      style={{ width: `${Math.min(100, Math.max(0, status?.percentage || 0))}%` }}
                    />
                  </div>

                  {/* Entry Count */}
                  <div className="flex items-center justify-center gap-2 text-sm">
                    <span className="text-blue-800 dark:text-blue-200">
                      {status?.processedEntries.toLocaleString()} of {status?.totalEntries.toLocaleString()} entries
                    </span>
                  </div>

                  {/* ETA */}
                  {status?.estimatedTimeRemaining && (
                    <div className="flex items-center justify-center gap-2 text-xs text-blue-700 dark:text-blue-300 mt-2">
                      <span>Estimated time remaining: {status.estimatedTimeRemaining}</span>
                    </div>
                  )}
                </div>

                {/* Info */}
                <div className="bg-gray-50 dark:bg-gray-900/20 border border-gray-200 dark:border-gray-800 rounded-lg p-3">
                  <p className="text-xs text-gray-700 dark:text-gray-300 text-center">
                    Please wait while the knowledge base is being resynced. This window will update automatically.
                  </p>
                </div>
              </>
            )}
          </div>

          {/* Footer with enhanced progress tracking */}
          {progressEvents.length > 0 && (
            <div className="border-t border-gray-200 dark:border-gray-700 px-6 py-4">
              <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
                Detailed Progress:
              </div>
              <div className="max-h-32 overflow-y-auto">
                {progressEvents.map((event) => (
                  <div key={event.id} className="flex items-center gap-2 py-1 text-xs">
                    <div className="w-2 h-2 rounded-full bg-blue-500 flex-shrink-0" />
                    <span className="text-gray-600 dark:text-gray-400">
                      {event.type === 'progress' ? event.description : 
                       event.type === 'typing' ? event.description : 
                       'Processing...'}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </Dialog.Content>
      </Dialog.Portal>

      {/* Render ProgressTracker as overlay if in progress */}
      {status?.inProgress && progressEvents.length > 0 && (
        <ProgressTracker
          events={progressEvents}
          showTypingIndicator={status.processedEntries === 0}
          className="fixed bottom-4 left-4 z-[60]" // Higher z-index than dialog
        />
      )}
    </Dialog.Root>
  );
};