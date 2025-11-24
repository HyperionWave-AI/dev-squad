import React, { useEffect, useState } from 'react';
import { type ContextMetadata } from '../../types/chat';
import '../ContextIndicator.css';

interface ContextIndicatorProps {
  contextMetadata?: ContextMetadata;
  isLoading?: boolean;
  error?: string;
  onArchiveClick?: () => void;
  onSummarizeClick?: () => void;
  isModalOpen?: boolean;
  onModalOpenChange?: (open: boolean) => void;
}

/**
 * Compact button showing context status icon and token count
 */
export const ContextStatusButton: React.FC<{
  contextMetadata?: ContextMetadata;
  isLoading?: boolean;
  error?: string;
  onClick?: () => void;
}> = ({ contextMetadata, isLoading, error, onClick }) => {
  const [status, setStatus] = useState<'healthy' | 'warning' | 'critical'>('healthy');

  useEffect(() => {
    if (!contextMetadata) return;

    if (contextMetadata.percentageUsed >= 90) {
      setStatus('critical');
    } else if (contextMetadata.percentageUsed >= 80) {
      setStatus('warning');
    } else {
      setStatus('healthy');
    }
  }, [contextMetadata]);

  if (isLoading) {
    return (
      <button
        className="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
        disabled
        title="Loading context status..."
      >
        <div className="w-4 h-4 border-2 border-gray-300 border-t-gray-600 rounded-full animate-spin" />
      </button>
    );
  }

  if (error) {
    return (
      <button
        onClick={onClick}
        className="p-2 rounded-lg bg-red-100 dark:bg-red-900/30 hover:bg-red-200 dark:hover:bg-red-900/50 transition-colors"
        title={error}
      >
        <span className="text-lg">⚠️</span>
      </button>
    );
  }

  if (!contextMetadata) {
    return null;
  }

  const tokenCount = contextMetadata.tokenCount ?? 0;
  const maxTokens = contextMetadata.maxTokens ?? 100000;
  const percentageUsed = contextMetadata.percentageUsed ?? 0;

  const formatTokens = (tokens: number): string => {
    if (tokens >= 1000000) {
      return `${(tokens / 1000000).toFixed(1)}M`;
    }
    if (tokens >= 1000) {
      return `${(tokens / 1000).toFixed(1)}K`;
    }
    return tokens.toString();
  };

  const statusColors = {
    healthy: 'bg-green-100 dark:bg-green-900/30 hover:bg-green-200 dark:hover:bg-green-900/50 border-green-500 dark:border-green-400',
    warning: 'bg-yellow-100 dark:bg-yellow-900/30 hover:bg-yellow-200 dark:hover:bg-yellow-900/50 border-yellow-500 dark:border-yellow-400',
    critical: 'bg-red-100 dark:bg-red-900/30 hover:bg-red-200 dark:hover:bg-red-900/50 border-red-500 dark:border-red-400',
  };

  const statusIcons = {
    healthy: '✓',
    warning: '⚠️',
    critical: '🚨',
  };

  return (
    <button
      onClick={onClick}
      className={`px-3 py-2 rounded-lg border-2 transition-colors flex items-center gap-2 text-sm font-medium ${statusColors[status]}`}
      title={`Context: ${formatTokens(tokenCount)} / ${formatTokens(maxTokens)} (${percentageUsed.toFixed(1)}%)`}
    >
      <span>{statusIcons[status]}</span>
      <span className="font-mono text-xs">{formatTokens(tokenCount)}</span>
    </button>
  );
};

/**
 * ContextStatusModal Component
 * Full modal showing detailed context usage information
 */
export const ContextStatusModal: React.FC<{
  isOpen: boolean;
  contextMetadata?: ContextMetadata;
  contextError?: any;
  isLoading?: boolean;
  error?: string;
  onClose: () => void;
  onArchiveClick?: () => void;
  onSummarizeClick?: () => void;
}> = ({
  isOpen,
  contextMetadata,
  contextError,
  isLoading,
  error,
  onClose,
  onArchiveClick,
  onSummarizeClick,
}) => {
  const [status, setStatus] = useState<'healthy' | 'warning' | 'critical'>('healthy');

  useEffect(() => {
    if (!contextMetadata) return;

    if (contextMetadata.percentageUsed >= 90) {
      setStatus('critical');
    } else if (contextMetadata.percentageUsed >= 80) {
      setStatus('warning');
    } else {
      setStatus('healthy');
    }
  }, [contextMetadata]);

  if (!isOpen) return null;

  if (isLoading) {
    return (
      <>
        <div
          className="fixed inset-0 bg-black/20 dark:bg-black/40 z-[1200] backdrop-blur-sm"
          onClick={onClose}
        />
        <div className="fixed inset-0 flex items-center justify-center z-[1300] p-4 pointer-events-none">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full p-6 pointer-events-auto">
            <div className="flex items-center justify-center gap-3">
              <div className="w-5 h-5 border-2 border-gray-300 border-t-gray-600 rounded-full animate-spin" />
              <span className="text-gray-600 dark:text-gray-400">Loading context status...</span>
            </div>
          </div>
        </div>
      </>
    );
  }

  if (error) {
    return (
      <>
        <div
          className="fixed inset-0 bg-black/20 dark:bg-black/40 z-[1200] backdrop-blur-sm"
          onClick={onClose}
        />
        <div className="fixed inset-0 flex items-center justify-center z-[1300] p-4 pointer-events-none">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full p-6 pointer-events-auto">
            <div className="flex items-start gap-3">
              <span className="text-2xl">⚠️</span>
              <div className="flex-1">
                <h3 className="font-semibold text-gray-900 dark:text-white mb-1">Error</h3>
                <p className="text-sm text-gray-600 dark:text-gray-400">{error}</p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="mt-4 w-full px-4 py-2 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition-colors text-sm font-medium"
            >
              Close
            </button>
          </div>
        </div>
      </>
    );
  }

  if (!contextMetadata) {
    return null;
  }

  const tokenCount = contextMetadata.tokenCount ?? 0;
  const maxTokens = contextMetadata.maxTokens ?? 100000;
  const percentageUsed = contextMetadata.percentageUsed ?? 0;

  const formatTokens = (tokens: number): string => {
    if (tokens >= 1000000) {
      return `${(tokens / 1000000).toFixed(1)}M`;
    }
    if (tokens >= 1000) {
      return `${(tokens / 1000).toFixed(1)}K`;
    }
    return tokens.toString();
  };

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/20 dark:bg-black/40 z-[1200] backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="fixed inset-0 flex items-center justify-center z-[1300] p-4 pointer-events-none">
        <div className={`bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full p-6 context-indicator context-${status} pointer-events-auto`}>
          {/* Header */}
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-white">Context Usage</h2>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
              aria-label="Close modal"
            >
              ✕
            </button>
          </div>

          {/* Token Count */}
          <div className="mb-4 p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div className="flex justify-between items-center">
              <span className="text-sm text-gray-600 dark:text-gray-400">Tokens Used</span>
              <span className="font-mono font-semibold text-gray-900 dark:text-white">
                {formatTokens(tokenCount)} / {formatTokens(maxTokens)}
              </span>
            </div>
          </div>

          {/* Progress Bar */}
          <div className="mb-4">
            <div className="flex justify-between items-center mb-2">
              <span className="text-sm text-gray-600 dark:text-gray-400">Usage</span>
              <span className="text-sm font-semibold text-gray-900 dark:text-white">
                {percentageUsed.toFixed(1)}%
              </span>
            </div>
            <div className="w-full h-3 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
              <div
                className={`h-full transition-all duration-300 ${
                  status === 'healthy'
                    ? 'bg-green-500'
                    : status === 'warning'
                    ? 'bg-yellow-500'
                    : 'bg-red-500'
                }`}
                style={{ width: `${Math.min(percentageUsed, 100)}%` }}
              />
            </div>
          </div>

          {/* Status Info */}
          <div className="mb-4 p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div className="flex items-center gap-2">
              <span className="text-lg">
                {status === 'healthy' ? '✓' : status === 'warning' ? '⚠️' : '🚨'}
              </span>
              <div>
                <div className="text-sm font-semibold text-gray-900 dark:text-white">
                  {status === 'healthy' ? 'Healthy' : status === 'warning' ? 'Warning' : 'Critical'}
                </div>
                <div className="text-xs text-gray-600 dark:text-gray-400">
                  {status === 'healthy'
                    ? 'Context usage is normal'
                    : status === 'warning'
                    ? 'Consider archiving or summarizing messages'
                    : 'Archive or summarize messages immediately'}
                </div>
              </div>
            </div>
          </div>

          {/* Message Count */}
          <div className="mb-4 p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
            <div className="flex justify-between items-center">
              <span className="text-sm text-gray-600 dark:text-gray-400">Messages</span>
              <span className="font-semibold text-gray-900 dark:text-white">
                {contextMetadata.messageCount ?? 0}
              </span>
            </div>
          </div>

          {/* Alert if needed */}
          {status !== 'healthy' && (
            <div className={`mb-4 p-3 rounded-lg ${
              status === 'warning'
                ? 'bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800'
                : 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
            }`}>
              <p className={`text-sm ${
                status === 'warning'
                  ? 'text-yellow-800 dark:text-yellow-200'
                  : 'text-red-800 dark:text-red-200'
              }`}>
                {contextError?.message || (status === 'warning'
                  ? 'Context usage is at warning level. Consider archiving or summarizing messages.'
                  : 'Context usage is critical. Archive or summarize messages immediately.')}
              </p>
            </div>
          )}

          {/* Action Buttons */}
          {(status === 'warning' || status === 'critical') && (
            <div className="flex gap-2 mb-4">
              {onArchiveClick && (
                <button
                  onClick={() => {
                    onArchiveClick();
                    onClose();
                  }}
                  className="flex-1 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg transition-colors text-sm font-medium"
                >
                  Archive Messages
                </button>
              )}
              {onSummarizeClick && (
                <button
                  onClick={() => {
                    onSummarizeClick();
                    onClose();
                  }}
                  className="flex-1 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors text-sm font-medium"
                >
                  Summarize
                </button>
              )}
            </div>
          )}

          {/* Close Button */}
          <button
            onClick={onClose}
            className="w-full px-4 py-2 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition-colors text-sm font-medium text-gray-900 dark:text-white"
          >
            Close
          </button>
        </div>
      </div>
    </>
  );
};

/**
 * ContextIndicator Component
 * Only renders the button - modal should be rendered at page level
 */
export const ContextIndicator: React.FC<ContextIndicatorProps> = ({
  contextMetadata,
  isLoading = false,
  error,
  onModalOpenChange,
}) => {
  // Only render the button - modal should be rendered at page level
  return (
    <ContextStatusButton
      contextMetadata={contextMetadata}
      isLoading={isLoading}
      error={error}
      onClick={() => onModalOpenChange?.(true)}
    />
  );
};

export default ContextIndicator;
