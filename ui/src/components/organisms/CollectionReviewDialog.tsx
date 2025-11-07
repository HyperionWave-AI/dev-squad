import React, { useState } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { X, ClipboardCheck, Download, Archive, AlertTriangle } from 'lucide-react';
import type { CollectionReviewResult } from '@/types/knowledge';

interface CollectionReviewDialogProps {
  open: boolean;
  onClose: () => void;
  result: CollectionReviewResult | null;
  onCompactSelected?: (entryIds: string[]) => void;
  onViewEntry?: (entryId: string) => void;
}

const getHealthColor = (score: number): string => {
  if (score >= 90) return 'bg-green-500';
  if (score >= 70) return 'bg-yellow-500';
  if (score >= 40) return 'bg-orange-500';
  return 'bg-red-500';
};

const getHealthBadgeColor = (score: number): string => {
  if (score >= 90) return 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200';
  if (score >= 70) return 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200';
  if (score >= 40) return 'bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-200';
  return 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-200';
};

export const CollectionReviewDialog: React.FC<CollectionReviewDialogProps> = ({
  open,
  onClose,
  result,
  onCompactSelected,
  onViewEntry,
}) => {
  const [selectedEntries, setSelectedEntries] = useState<string[]>([]);

  if (!result) {
    return null;
  }

  const { summary, entries } = result;
  const avgHealthColor = getHealthColor(summary.averageHealth);

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedEntries(entries.map((e) => e.entryId));
    } else {
      setSelectedEntries([]);
    }
  };

  const handleSelectEntry = (entryId: string) => {
    setSelectedEntries((prev) =>
      prev.includes(entryId)
        ? prev.filter((id) => id !== entryId)
        : [...prev, entryId]
    );
  };

  const handleCompactSelected = () => {
    if (onCompactSelected && selectedEntries.length > 0) {
      onCompactSelected(selectedEntries);
      setSelectedEntries([]);
    }
  };

  const handleExportReport = () => {
    const reportData = {
      collection: result.collection,
      summary,
      entries,
      generatedAt: new Date().toISOString(),
    };
    const blob = new Blob([JSON.stringify(reportData, null, 2)], {
      type: 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${result.collection}-review-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const allSelected = entries.length > 0 && selectedEntries.length === entries.length;
  const someSelected = selectedEntries.length > 0 && selectedEntries.length < entries.length;

  return (
    <Dialog.Root open={open} onOpenChange={onClose}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 data-[state=open]:animate-fade-in z-50" />
        <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-6xl max-h-[90vh] overflow-hidden data-[state=open]:animate-scale-in z-50">
          {/* Header */}
          <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <ClipboardCheck className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Collection Review: {result.collection}
              </Dialog.Title>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={handleExportReport}
                className="p-2 rounded-md text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                title="Export Report"
              >
                <Download className="h-4 w-4" />
              </button>
              <Dialog.Close asChild>
                <button
                  className="p-1 rounded-md text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                  aria-label="Close"
                >
                  <X className="h-5 w-5" />
                </button>
              </Dialog.Close>
            </div>
          </div>

          {/* Content */}
          <div className="overflow-y-auto max-h-[calc(90vh-180px)]">
            <div className="px-6 py-4 space-y-4">
              {/* Summary Statistics */}
              <div>
                <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                  Summary Statistics
                </h3>
                <div className="flex gap-3 flex-wrap mb-3">
                  <span className="inline-flex items-center px-3 py-1.5 rounded-md text-sm font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200 border border-blue-200 dark:border-blue-900/50">
                    Total Entries: {summary.totalEntries}
                  </span>
                  <span className="inline-flex items-center px-3 py-1.5 rounded-md text-sm font-medium bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-200 border border-purple-200 dark:border-purple-900/50">
                    Reviewed: {summary.entriesReviewed}
                  </span>
                  <span className="inline-flex items-center px-3 py-1.5 rounded-md text-sm font-medium bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 border border-yellow-200 dark:border-yellow-900/50">
                    Low Score: {summary.lowScoreCount}
                  </span>
                </div>
                <div>
                  <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                    Average Health Score: {Math.round(summary.averageHealth)}%
                  </p>
                  <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-2 overflow-hidden">
                    <div
                      className={`h-full ${avgHealthColor} transition-all duration-300`}
                      style={{ width: `${summary.averageHealth}%` }}
                    />
                  </div>
                </div>
              </div>

              {/* Low Score Warning */}
              {summary.lowScoreCount > 0 && (
                <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-900/50 rounded-lg p-3">
                  <div className="flex items-start gap-2">
                    <AlertTriangle className="h-4 w-4 text-yellow-600 dark:text-yellow-400 mt-0.5 flex-shrink-0" />
                    <p className="text-sm text-yellow-800 dark:text-yellow-200">
                      Found {summary.lowScoreCount} entries with low health scores that may need attention.
                    </p>
                  </div>
                </div>
              )}

              {/* Selection Info */}
              {selectedEntries.length > 0 && (
                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-900/50 rounded-lg p-3">
                  <p className="text-sm text-blue-800 dark:text-blue-200">
                    {selectedEntries.length} entry(ies) selected
                  </p>
                </div>
              )}

              {/* Entries Table */}
              <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                    <thead className="bg-gray-50 dark:bg-gray-900">
                      <tr>
                        <th className="w-12 px-4 py-3 text-left">
                          <input
                            type="checkbox"
                            checked={allSelected}
                            ref={(input) => {
                              if (input) input.indeterminate = someSelected;
                            }}
                            onChange={(e) => handleSelectAll(e.target.checked)}
                            className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                          />
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                          Entry ID
                        </th>
                        <th className="px-4 py-3 text-center text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                          Health Score
                        </th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                          Issues
                        </th>
                        <th className="px-4 py-3 text-center text-xs font-medium text-gray-700 dark:text-gray-300 uppercase tracking-wider">
                          Actions
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                      {entries.length === 0 ? (
                        <tr>
                          <td colSpan={5} className="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                            No entries to display
                          </td>
                        </tr>
                      ) : (
                        entries.map((entry) => (
                          <tr
                            key={entry.entryId}
                            className="hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
                          >
                            <td className="px-4 py-3">
                              <input
                                type="checkbox"
                                checked={selectedEntries.includes(entry.entryId)}
                                onChange={() => handleSelectEntry(entry.entryId)}
                                className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                              />
                            </td>
                            <td className="px-4 py-3">
                              <span className="text-xs font-mono text-gray-900 dark:text-gray-100 max-w-[200px] truncate block">
                                {entry.entryId}
                              </span>
                            </td>
                            <td className="px-4 py-3 text-center">
                              <span
                                className={`inline-flex items-center px-2.5 py-1 rounded-md text-xs font-semibold min-w-[3rem] justify-center ${getHealthBadgeColor(entry.healthScore)}`}
                              >
                                {Math.round(entry.healthScore)}
                              </span>
                            </td>
                            <td className="px-4 py-3">
                              <div className="flex gap-1.5 flex-wrap">
                                {entry.issues.length === 0 ? (
                                  <span className="text-xs text-gray-500 dark:text-gray-400">
                                    No issues
                                  </span>
                                ) : (
                                  entry.issues.map((issue, idx) => (
                                    <span
                                      key={idx}
                                      className="inline-flex items-center px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-600"
                                    >
                                      {issue}
                                    </span>
                                  ))
                                )}
                              </div>
                            </td>
                            <td className="px-4 py-3 text-center">
                              <button
                                onClick={() => onViewEntry?.(entry.entryId)}
                                disabled={!onViewEntry}
                                className="text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 disabled:opacity-50 disabled:cursor-not-allowed"
                              >
                                View
                              </button>
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 px-6 py-4 flex justify-end gap-3">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
            >
              Close
            </button>
            {onCompactSelected && (
              <button
                onClick={handleCompactSelected}
                disabled={selectedEntries.length === 0}
                className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <Archive className="h-4 w-4" />
                Compact Selected ({selectedEntries.length})
              </button>
            )}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
};
