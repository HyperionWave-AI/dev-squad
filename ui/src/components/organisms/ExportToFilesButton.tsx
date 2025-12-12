import React, { useState } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { Download, X, AlertTriangle, CheckCircle, AlertCircle, FolderOpen } from 'lucide-react';
import { Button } from '@/components/atoms/Button';
import { knowledgeService } from '@/services/knowledgeService';
import type { ExportReport } from '@/types/knowledge';

interface ExportToFilesButtonProps {
  onExportComplete?: () => void;
}

export const ExportToFilesButton: React.FC<ExportToFilesButtonProps> = ({ onExportComplete }) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [exportReport, setExportReport] = useState<ExportReport | null>(null);
  const [showReportDialog, setShowReportDialog] = useState(false);

  const handleExport = async () => {
    setLoading(true);
    setError(null);
    setExportReport(null);

    try {
      const report = await knowledgeService.exportToFiles();
      setExportReport(report);
      setShowReportDialog(true);

      // Call onExportComplete callback to refresh collections
      if (onExportComplete) {
        onExportComplete();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to export knowledge to files');
    } finally {
      setLoading(false);
    }
  };

  const handleCloseReport = () => {
    setShowReportDialog(false);
    setExportReport(null);
    setError(null);
  };

  const hasErrors = exportReport && exportReport.errors && exportReport.errors.length > 0;

  return (
    <>
      <Button
        variant="secondary"
        size="md"
        onClick={handleExport}
        disabled={loading}
        className="inline-flex items-center gap-2"
      >
        <Download className={`h-4 w-4 ${loading ? 'animate-bounce' : ''}`} />
        {loading ? 'Exporting...' : 'Export to Files'}
      </Button>

      {/* Error Toast */}
      {error && !showReportDialog && (
        <div className="fixed bottom-4 right-4 z-50 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 shadow-lg max-w-md">
          <div className="flex items-start gap-3">
            <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
            <div className="flex-1">
              <p className="text-sm font-medium text-red-900 dark:text-red-100">Export Failed</p>
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

      {/* Export Report Dialog */}
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
                  Knowledge Export Report
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
              {exportReport && (
                <>
                  {/* Success Summary */}
                  {!hasErrors && (
                    <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
                      <div className="flex items-start gap-3">
                        <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 mt-0.5 flex-shrink-0" />
                        <div>
                          <p className="text-sm font-medium text-green-900 dark:text-green-100">
                            Export completed successfully!
                          </p>
                          <p className="text-sm text-green-800 dark:text-green-200 mt-1">
                            All knowledge entries have been exported to markdown files.
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
                            Export completed with errors
                          </p>
                          <p className="text-sm text-orange-800 dark:text-orange-200 mt-1">
                            Some entries could not be exported. See details below.
                          </p>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* Output Path */}
                  <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                    <div className="flex items-center gap-3">
                      <div className="p-2 bg-indigo-100 dark:bg-indigo-900/30 rounded-lg">
                        <FolderOpen className="h-5 w-5 text-indigo-600 dark:text-indigo-400" />
                      </div>
                      <div>
                        <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Output Path</p>
                        <p className="text-base font-mono text-gray-900 dark:text-gray-100">{exportReport.outputPath}</p>
                      </div>
                    </div>
                  </div>

                  {/* Statistics Grid */}
                  <div className="grid grid-cols-2 gap-4">
                    {/* Collections Exported */}
                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                          <svg className="h-5 w-5 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                          </svg>
                        </div>
                        <div>
                          <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Collections</p>
                          <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{exportReport.collectionsExported}</p>
                        </div>
                      </div>
                    </div>

                    {/* Entries Exported */}
                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
                          <svg className="h-5 w-5 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                          </svg>
                        </div>
                        <div>
                          <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Entries Exported</p>
                          <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{exportReport.entriesExported}</p>
                        </div>
                      </div>
                    </div>

                    {/* Files Created */}
                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                          <Download className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                        </div>
                        <div>
                          <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Files Created</p>
                          <p className="text-2xl font-bold text-gray-900 dark:text-gray-100">{exportReport.filesCreated}</p>
                        </div>
                      </div>
                    </div>

                    {/* Errors Count */}
                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                      <div className="flex items-center gap-3">
                        <div className={`p-2 rounded-lg ${hasErrors ? 'bg-red-100 dark:bg-red-900/30' : 'bg-gray-100 dark:bg-gray-900/30'}`}>
                          <AlertCircle className={`h-5 w-5 ${hasErrors ? 'text-red-600 dark:text-red-400' : 'text-gray-600 dark:text-gray-400'}`} />
                        </div>
                        <div>
                          <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Errors</p>
                          <p className={`text-2xl font-bold ${hasErrors ? 'text-red-600 dark:text-red-400' : 'text-gray-900 dark:text-gray-100'}`}>
                            {exportReport.errors?.length || 0}
                          </p>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Collections List */}
                  {exportReport.collections && exportReport.collections.length > 0 && (
                    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                      <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-3">
                        Collections Exported ({exportReport.collections.length})
                      </h3>
                      <div className="flex flex-wrap gap-2">
                        {exportReport.collections.map((collection, index) => (
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
                        Errors ({exportReport.errors!.length})
                      </h3>
                      <div className="space-y-2 max-h-48 overflow-y-auto">
                        {exportReport.errors!.map((error, index) => (
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
