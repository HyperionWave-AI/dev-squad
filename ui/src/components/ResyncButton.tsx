import React, { useState } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { RefreshCw, X, AlertTriangle } from 'lucide-react';
import { Button } from '@/components/atoms/Button';
import { knowledgeService } from '@/services/knowledgeService';

interface ResyncButtonProps {
  onResyncStarted: () => void;
}

export const ResyncButton: React.FC<ResyncButtonProps> = ({ onResyncStarted }) => {
  const [showConfirmation, setShowConfirmation] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleConfirm = async () => {
    setLoading(true);
    setError(null);
    try {
      await knowledgeService.startResync();
      setShowConfirmation(false);
      onResyncStarted();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start resync');
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    if (!loading) {
      setShowConfirmation(false);
      setError(null);
    }
  };

  return (
    <>
      <Button
        variant="primary"
        size="md"
        onClick={() => setShowConfirmation(true)}
        className="inline-flex items-center gap-2"
      >
        <RefreshCw className="h-4 w-4" />
        Resync to Unified Collection
      </Button>

      <Dialog.Root open={showConfirmation} onOpenChange={handleClose}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 data-[state=open]:animate-fade-in z-50" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md data-[state=open]:animate-scale-in z-50">
            {/* Header */}
            <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <RefreshCw className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  Resync to Unified Collection
                </Dialog.Title>
              </div>
              <Dialog.Close asChild>
                <button
                  className="p-1 rounded-md text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                  aria-label="Close"
                  disabled={loading}
                >
                  <X className="h-5 w-5" />
                </button>
              </Dialog.Close>
            </div>

            {/* Content */}
            <div className="px-6 py-4 space-y-4">
              {/* Info Alert */}
              <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-900/50 rounded-lg p-4">
                <div className="flex items-start gap-3">
                  <AlertTriangle className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5 flex-shrink-0" />
                  <div className="space-y-2">
                    <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                      This will rebuild the knowledge base from MongoDB
                    </p>
                    <p className="text-sm text-blue-800 dark:text-blue-200">
                      All entries will be resynced to the unified Qdrant collection. This process may take a few moments depending on the number of entries.
                    </p>
                  </div>
                </div>
              </div>

              {/* Error Alert */}
              {error && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                  <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
                </div>
              )}

              <p className="text-sm text-gray-600 dark:text-gray-400">
                Do you want to continue?
              </p>
            </div>

            {/* Footer */}
            <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 px-6 py-4 flex justify-end gap-3">
              <button
                onClick={handleClose}
                disabled={loading}
                className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirm}
                disabled={loading}
                className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading ? (
                  <>
                    <RefreshCw className="h-4 w-4 animate-spin" />
                    Starting...
                  </>
                ) : (
                  <>
                    <RefreshCw className="h-4 w-4" />
                    Start Resync
                  </>
                )}
              </button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </>
  );
};
