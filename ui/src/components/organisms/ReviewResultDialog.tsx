import React from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { X, CheckCircle, AlertCircle, AlertTriangle } from 'lucide-react';
import type { ReviewResult } from '@/types/knowledge';

interface ReviewResultDialogProps {
  open: boolean;
  onClose: () => void;
  result: ReviewResult | null;
}

const getHealthColor = (score: number): string => {
  if (score >= 90) return 'bg-green-500';
  if (score >= 70) return 'bg-yellow-500';
  if (score >= 40) return 'bg-orange-500';
  return 'bg-red-500';
};

const getHealthTextColor = (score: number): string => {
  if (score >= 90) return 'text-green-600 dark:text-green-400';
  if (score >= 70) return 'text-yellow-600 dark:text-yellow-400';
  if (score >= 40) return 'text-orange-600 dark:text-orange-400';
  return 'text-red-600 dark:text-red-400';
};

const getHealthLabel = (score: number): string => {
  if (score >= 90) return 'Excellent';
  if (score >= 70) return 'Good';
  if (score >= 40) return 'Fair';
  return 'Poor';
};

export const ReviewResultDialog: React.FC<ReviewResultDialogProps> = ({
  open,
  onClose,
  result,
}) => {
  console.log('🔍 [REVIEW DIALOG] Rendered with props:', { open, hasResult: !!result });

  if (!result) {
    console.log('⚠️ [REVIEW DIALOG] No result provided, returning null');
    return null;
  }

  console.log('✅ [REVIEW DIALOG] Result received:', result);
  const { scores, verification, actions } = result;
  const healthColor = getHealthColor(scores.health);
  const healthTextColor = getHealthTextColor(scores.health);

  return (
    <Dialog.Root open={open} onOpenChange={onClose}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 data-[state=open]:animate-fade-in z-50" />
        <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-2xl max-h-[85vh] overflow-y-auto data-[state=open]:animate-scale-in z-50">
          {/* Header */}
          <div className="sticky top-0 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4 flex items-center justify-between">
            <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Entry Review Results
            </Dialog.Title>
            <div className="flex items-center gap-3">
              <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-semibold ${healthTextColor}`}>
                {getHealthLabel(scores.health)} ({Math.round(scores.health)})
              </span>
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
          <div className="px-6 py-4 space-y-6">
            {/* Overall Health Score */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Overall Health Score
              </h3>
              <div className="flex items-center gap-3">
                <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-3 overflow-hidden">
                  <div
                    className={`h-full ${healthColor} transition-all duration-300`}
                    style={{ width: `${scores.health}%` }}
                  />
                </div>
                <span className="text-sm font-semibold text-gray-900 dark:text-gray-100 min-w-[3rem] text-right">
                  {Math.round(scores.health)}%
                </span>
              </div>
            </div>

            <div className="border-t border-gray-200 dark:border-gray-700" />

            {/* Component Scores */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                Component Scores
              </h3>
              <div className="space-y-3">
                {[
                  { label: 'Alignment', value: scores.alignment },
                  { label: 'Freshness', value: scores.freshness },
                  { label: 'Verbosity', value: scores.verbosity },
                  { label: 'Uniqueness', value: scores.uniqueness },
                ].map((item) => (
                  <div key={item.label}>
                    <div className="flex justify-between items-center mb-1">
                      <span className="text-sm text-gray-600 dark:text-gray-400">
                        {item.label}
                      </span>
                      <span className="text-sm font-semibold text-gray-900 dark:text-gray-100">
                        {Math.round(item.value)}%
                      </span>
                    </div>
                    <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-2 overflow-hidden">
                      <div
                        className={`h-full ${getHealthColor(item.value)} transition-all duration-300`}
                        style={{ width: `${item.value}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div className="border-t border-gray-200 dark:border-gray-700" />

            {/* Reference Verification */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                Reference Verification
              </h3>
              <div className="flex gap-3 mb-3">
                <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200">
                  <CheckCircle className="h-4 w-4" />
                  {verification.validReferences} Valid
                </span>
                <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-200">
                  <AlertCircle className="h-4 w-4" />
                  {verification.brokenReferences.length} Broken
                </span>
              </div>

              {verification.brokenReferences.length > 0 && (
                <div className="space-y-2">
                  <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-900/50 rounded-lg p-3">
                    <div className="flex items-start gap-2">
                      <AlertTriangle className="h-4 w-4 text-yellow-600 dark:text-yellow-400 mt-0.5 flex-shrink-0" />
                      <p className="text-sm text-yellow-800 dark:text-yellow-200">
                        Found {verification.brokenReferences.length} broken reference(s)
                      </p>
                    </div>
                  </div>
                  <ul className="space-y-2">
                    {verification.brokenReferences.map((ref, index) => (
                      <li
                        key={index}
                        className="bg-gray-50 dark:bg-gray-900/50 rounded-lg p-3 border border-gray-200 dark:border-gray-700"
                      >
                        <div className="text-sm">
                          <span className="font-medium text-gray-900 dark:text-gray-100">
                            {ref.type}:
                          </span>{' '}
                          <span className="text-gray-700 dark:text-gray-300">
                            {ref.value}
                          </span>
                        </div>
                        <p className="text-xs text-red-600 dark:text-red-400 mt-1">
                          {ref.error}
                        </p>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
            </div>

            <div className="border-t border-gray-200 dark:border-gray-700" />

            {/* Suggested Actions */}
            <div>
              <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
                Suggested Actions
              </h3>
              {actions.length > 0 ? (
                <ul className="space-y-2">
                  {actions.map((action, index) => (
                    <li
                      key={index}
                      className="flex items-start gap-3 p-3 bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-700"
                    >
                      {action.applied ? (
                        <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
                      ) : (
                        <AlertTriangle className="h-5 w-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5" />
                      )}
                      <div className="flex-1">
                        <p className="text-sm text-gray-900 dark:text-gray-100">
                          <span className="font-medium">{action.type}:</span>{' '}
                          {action.description}
                        </p>
                      </div>
                      {action.applied && (
                        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200">
                          Applied
                        </span>
                      )}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  No actions suggested
                </p>
              )}
            </div>
          </div>

          {/* Footer */}
          <div className="sticky bottom-0 bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 px-6 py-4 flex justify-end">
            <button
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors"
            >
              Close
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
};
