import React, { useState } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { X, Archive, CheckCircle, AlertTriangle, Loader2 } from 'lucide-react';
import type { CompactionResult } from '@/types/knowledge';

interface CompactionDialogProps {
  open: boolean;
  onClose: () => void;
  result: CompactionResult | null;
  onApprove: () => void;
  loading?: boolean;
}

export const CompactionDialog: React.FC<CompactionDialogProps> = ({
  open,
  onClose,
  result,
  onApprove,
  loading = false,
}) => {
  const [applying, setApplying] = useState(false);

  console.log('📦 [COMPACT DIALOG] Rendered with props:', { open, hasResult: !!result, loading });

  if (!result) {
    console.log('⚠️ [COMPACT DIALOG] No result provided, returning null');
    return null;
  }

  console.log('✅ [COMPACT DIALOG] Result received:', result);

  const handleApprove = async () => {
    console.log('💾 [COMPACT DIALOG] Approve button clicked');
    setApplying(true);
    try {
      await onApprove();
    } finally {
      setApplying(false);
    }
  };

  const { original, compacted, compressionRatio, preserved } = result;
  const wordReduction = original.wordCount - compacted.wordCount;
  const percentReduction = Math.round((wordReduction / original.wordCount) * 100);

  return (
    <Dialog.Root open={open} onOpenChange={onClose}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 data-[state=open]:animate-fade-in z-50" />
        <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-6xl max-h-[90vh] overflow-hidden data-[state=open]:animate-scale-in z-50">
          {/* Header */}
          <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Archive className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Compaction Preview
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
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 text-blue-600 dark:text-blue-400 animate-spin" />
            </div>
          ) : (
            <div className="overflow-y-auto max-h-[calc(90vh-180px)]">
              <div className="px-6 py-4 space-y-4">
                {/* Info Alert */}
                <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-900/50 rounded-lg p-3">
                  <p className="text-sm text-blue-800 dark:text-blue-200">
                    This is a preview. Click "Apply Compaction" to save the changes.
                  </p>
                </div>

                {/* Statistics */}
                <div className="flex gap-3 flex-wrap">
                  <span className="inline-flex items-center px-3 py-1.5 rounded-md text-sm font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200 border border-blue-200 dark:border-blue-900/50">
                    Word Count: {original.wordCount} → {compacted.wordCount}
                  </span>
                  <span className="inline-flex items-center px-3 py-1.5 rounded-md text-sm font-medium bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200 border border-green-200 dark:border-green-900/50">
                    Reduction: {wordReduction} words ({percentReduction}%)
                  </span>
                  <span className="inline-flex items-center px-3 py-1.5 rounded-md text-sm font-medium bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-200 border border-purple-200 dark:border-purple-900/50">
                    Compression Ratio: {Math.round(compressionRatio * 100)}%
                  </span>
                </div>

                {/* Preserved Elements */}
                <div>
                  <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Preserved Elements
                  </h3>
                  <div className="flex gap-3">
                    <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200">
                      <CheckCircle className="h-4 w-4" />
                      {preserved.filePaths} File Paths
                    </span>
                    <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200">
                      <CheckCircle className="h-4 w-4" />
                      {preserved.functionNames} Function Names
                    </span>
                  </div>
                </div>

                {/* Side-by-Side Comparison */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {/* Original */}
                  <div className="bg-gray-50 dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
                    <div className="bg-gray-100 dark:bg-gray-800 px-4 py-3 border-b border-gray-200 dark:border-gray-700 flex justify-between items-center">
                      <h4 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                        Original
                      </h4>
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300">
                        {original.wordCount} words
                      </span>
                    </div>
                    <div className="p-4 h-96 overflow-y-auto">
                      <pre className="text-xs text-gray-900 dark:text-gray-100 whitespace-pre-wrap break-words font-mono leading-relaxed">
                        {original.text}
                      </pre>
                    </div>
                  </div>

                  {/* Compacted */}
                  <div className="bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-900/50 overflow-hidden">
                    <div className="bg-green-100 dark:bg-green-900/30 px-4 py-3 border-b border-green-200 dark:border-green-900/50 flex justify-between items-center">
                      <h4 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                        Compacted
                      </h4>
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-200 dark:bg-green-900/50 text-green-800 dark:text-green-200">
                        {compacted.wordCount} words
                      </span>
                    </div>
                    <div className="p-4 h-96 overflow-y-auto">
                      <pre className="text-xs text-gray-900 dark:text-gray-100 whitespace-pre-wrap break-words font-mono leading-relaxed">
                        {compacted.text}
                      </pre>
                    </div>
                  </div>
                </div>

                {/* Warning for high compression */}
                {compressionRatio < 0.5 && (
                  <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-900/50 rounded-lg p-3">
                    <div className="flex items-start gap-2">
                      <AlertTriangle className="h-4 w-4 text-yellow-600 dark:text-yellow-400 mt-0.5 flex-shrink-0" />
                      <p className="text-sm text-yellow-800 dark:text-yellow-200">
                        This compaction achieves significant compression ({Math.round(compressionRatio * 100)}%).
                        Please review carefully to ensure important information is preserved.
                      </p>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Footer */}
          <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 px-6 py-4 flex justify-end gap-3">
            <button
              onClick={onClose}
              disabled={applying}
              className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Cancel
            </button>
            <button
              onClick={handleApprove}
              disabled={applying || loading}
              className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {applying ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Applying...
                </>
              ) : (
                <>
                  <Archive className="h-4 w-4" />
                  Apply Compaction
                </>
              )}
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
};
