import React, { useState } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { RefreshCw, X, AlertTriangle, CheckCircle, AlertCircle } from 'lucide-react';
import { Button } from '@/components/atoms/Button';
import { knowledgeService } from '@/services/knowledgeService';
import type { SyncReport } from '@/types/knowledge';

interface MarkdownSyncButtonProps {
  onSyncComplete?: () => void;
}

export const MarkdownSyncButton: React.FC<MarkdownSyncButtonProps> = ({ onSyncComplete }) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [syncReport, setSyncReport] = useState<SyncReport | null>(null);
  const [showReportDialog, setShowReportDialog] = useState(false);

  const handleSync = async () => {
    setLoading(true);
    setError(null);
    setSyncReport(null);

    try {
      const report = await knowledgeService.syncMarkdownKB();
      setSyncReport(report);
      setShowReportDialog(true);

      // Call onSyncComplete callback to refresh collections
      if (onSyncComplete) {
        onSyncComplete();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to sync markdown files');
    } finally {
      setLoading(false);
    }
  };

  const handleCloseReport = () => {
    setShowReportDialog(false);
    setSyncReport(null);
    setError(null);
  };

  const hasErrors = syncReport && syncReport.errors && syncReport.errors.length > 0;

  return (
    <>
      <Button
        variant="primary"
        size="md"
        onClick={handleSync}
        disabled={loading}
        className="inline-flex items-center gap-2"
      >
        <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        {loading ? 'Syncing...' : 'Sync KB Files'}
      </Button>

      {/* Error Toast */}
      {error && !showReportDialog && (
        <div className="fixed bottom-4 right-4 z-50 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 shadow-lg max-w-md">
          <div className="flex items-start gap-3">
            <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
            <div className="flex-1">
              <p className="text-sm font-medium text-red-900 dark:text-red-100">Sync Failed</p>
              <p className="text-sm text-red-800 dark:text-red-200 mt-1">{error}</p>
            </div>
            <button
              onClick={() => setError(null)}
              className="text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-200"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {/* Sync Report Dialog */}
      <Dialog.Root open={showReportDialog} onOpenChange={handleCloseReport}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 data-[state=open]:animate-fade-in z-50" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-2xl data-[state=open]:animate-scale-in z-50 max-h-[90vh] overflow-hidden flex flex-col">
            {/* Header */}
            <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4 flex items-center justify-between flex-shrink-0">
              <div className="flex items-center gap-2">
                {hasErrors ? (
                  <AlertTriangle className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                ) : (
                  <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400" />
                )}
                <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  Markdown Sync Report
                </Dialog.Title>
              </div>
              <Dialog.Close asChild>
                <button
                  className="p-1 rounded-md text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                  aria-label="Close"
                >
                  <X className="h-5 w-5" />
                </button>
              </Dialog.Close>
            </div>

            {/* Content */}
            <div className="px-6 py-6 space-y-4 overflow-y-auto flex-1">
              {syncReport && (
                <>
                  {/* Success Summary */}
                  {!hasErrors && (
                    <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
                      <div className="flex items-start gap-3">
                        <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5 flex-shrink-0" />
                        <div>
                          <p className="text-sm font-medium text-green-900 dark:text-green-100">
                            Sync completed successfully!
                          </p>
                          <p className="text-sm text-green-800 dark:text-green-200 mt-1">
                            All markdown files have been processed and imported.
                          </p>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Warning for errors */}
                  {hasErrors && (
                    <div className="bg-orange-50 dark:bg-orange-900/20 border border-orange-200 dark:border-orange-800 rounded-lg p-4">
                      <div className="flex items-start gap-3">
                        <AlertTriangle className="h-5 w-5 text-orange-600 dark:text-orange-400 mt-0.5 flex-shrink-0" />
                        <div>
                          <p className="text-sm font-medium text-orange-900 dark:text-orange-100">
                            Sync completed with errors
                          </p>
                          <p className="text-sm text-orange-800 dark:text-orange-200 mt-1">
                            Some files could not be processed. See details below.
                          </p>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Statistics Grid */}
                  <div className="grid grid-cols-2 gap-4">
                    {/* Files Processed */}
                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                          <RefreshCw className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                        </div>
                        <div>
                          <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Files Processed</p>
                          <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{syncReport.filesProcessed}</p>
                        </div>
                      </div>
                    </div>

                    {/* Entries Created */}
                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
                          <svg className="h-5 w-5 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                          </svg>
                        </div>
                        <div>
                          <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Created</p>
                          <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{syncReport.entriesCreated}</p>
                        </div>
                      </div>
                    </div>

                    {/* Entries Updated */}
                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                          <svg className="h-5 w-5 text-purple-600 dark:text-purple-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                          </svg>
                        </div>
                        <div>
                          <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Updated</p>
                          <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{syncReport.entriesUpdated}</p>
                        </div>
                      </div>
                    </div>

                    {/* Total Entries */}
                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-indigo-100 dark:bg-indigo-900/30 rounded-lg">
                          <svg className="h-5 w-5 text-indigo-600 dark:text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                          </svg>
                        </div>
                        <div>
                          <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Total Entries</p>
                          <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{syncReport.entriesCreated + syncReport.entriesUpdated}</p>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Collections List */}
                  {syncReport.collections && syncReport.collections.length > 0 && (
                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                      <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-3">
                        Collections Added/Updated ({syncReport.collections.length})
                      </h3>
                      <div className="flex flex-wrap gap-2">
                        {syncReport.collections.map((collection, index) => (
                          <span
                            key={index}
                            className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200 border border-blue-200 dark:border-blue-800"
                          >
                            {collection}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Errors List */}
                  {hasErrors && (
                    <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                      <h3 className="text-sm font-semibold text-red-900 dark:text-red-100 mb-3 flex items-center gap-2">
                        <AlertCircle className="h-4 w-4" />
                        Errors ({syncReport.errors.length})
                      </h3>
                      <div className="space-y-2 max-h-48 overflow-y-auto">
                        {syncReport.errors.map((error, index) => (
                          <div
                            key={index}
                            className="text-xs text-red-800 dark:text-red-200 bg-red-100 dark:bg-red-900/30 rounded px-3 py-2 font-mono"
                          >
                            {error}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </>
              )}
            </div>

            {/* Footer */}
            <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 px-6 py-4 flex justify-end gap-3 flex-shrink-0">
              <button
                onClick={handleCloseReport}
                className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors"
              >
                Close
              </button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </>
  );
};
