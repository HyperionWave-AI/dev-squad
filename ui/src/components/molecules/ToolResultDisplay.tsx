/**
 * ToolResultDisplay Component
 *
 * Displays tool execution results with size-aware rendering:
 * - Normal results: Display fully
 * - Large results: Show preview with "Show Full" button
 * - Suppressed results: Show backend-generated helpful message
 *
 * Implements Claude-style progressive disclosure for large tool outputs.
 */

import React, { useState } from 'react';
import { AlertTriangle, ChevronDown, ChevronUp } from 'lucide-react';
import { cn } from '@/utils';
import {
  getDisplayContent,
  isBackendSuppressedResult,
} from '@/utils/toolResultHelpers';
import type { ToolResult } from '@/services/chatService';

interface ToolResultDisplayProps {
  toolResult: ToolResult;
  hasError: boolean;
}

export const ToolResultDisplay: React.FC<ToolResultDisplayProps> = ({
  toolResult,
  hasError,
}) => {
  const [showFullResult, setShowFullResult] = useState(false);

  // Handle errors - always show fully
  if (hasError) {
    return (
      <>
        <div className="text-xs text-gray-700 dark:text-gray-300 font-semibold mb-1">
          Error:
        </div>
        <pre
          className={cn(
            'p-2 rounded text-xs overflow-x-auto max-h-96',
            'bg-red-900/20 text-red-300'
          )}
        >
          {toolResult.error}
        </pre>
      </>
    );
  }

  // Check if backend suppressed/truncated the result
  const isBackendSuppressed = isBackendSuppressedResult(toolResult.result);

  // Get display content based on size and user preference
  const display = getDisplayContent(toolResult.result, {
    showFull: showFullResult,
  });

  return (
    <>
      <div className="flex items-center justify-between text-xs text-gray-700 dark:text-gray-300 font-semibold mb-1">
        <span>Result:</span>
        {display.showWarning && !isBackendSuppressed && (
          <span className="flex items-center gap-1 text-yellow-600 dark:text-yellow-400">
            <AlertTriangle className="w-3 h-3" />
            {display.warningMessage}
          </span>
        )}
        {isBackendSuppressed && (
          <span className="flex items-center gap-1 text-orange-600 dark:text-orange-400">
            <AlertTriangle className="w-3 h-3" />
            Suppressed by backend
          </span>
        )}
      </div>

      <pre
        className={cn(
          'p-2 rounded text-xs overflow-x-auto',
          isBackendSuppressed
            ? 'max-h-none bg-orange-50 dark:bg-orange-900/20 text-gray-900 dark:text-gray-100'
            : 'max-h-96 bg-gray-900 text-gray-100',
          // If showing full result, allow more height
          showFullResult && 'max-h-none'
        )}
      >
        {display.content}
      </pre>

      {display.showExpandButton && !isBackendSuppressed && (
        <button
          onClick={() => setShowFullResult(!showFullResult)}
          className="mt-2 flex items-center gap-1 text-xs text-primary-600 dark:text-primary-400 hover:underline focus:outline-none"
          aria-label={showFullResult ? 'Show less' : 'Show full result'}
        >
          {showFullResult ? (
            <>
              <ChevronUp className="w-3 h-3" />
              Show Less
            </>
          ) : (
            <>
              <ChevronDown className="w-3 h-3" />
              Show Full Result
            </>
          )}
        </button>
      )}
    </>
  );
};

ToolResultDisplay.displayName = 'ToolResultDisplay';
