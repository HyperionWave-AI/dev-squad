/**
 * Progress Tracker Component
 *
 * Displays AI agent progress with pulsating icons and step tracking.
 * Fixed position in bottom-right corner.
 *
 * Features:
 * - Pulsating animations for active steps (Tailwind animate-pulse + animate-ping)
 * - Status indicators: pending, in_progress, completed, error
 * - Step numbers (e.g., "Step 2/5")
 * - Auto-hide when no progress
 * - Dark mode support
 */

import React from 'react';
import {
  CheckCircle,
  XCircle,
  Clock,
  Loader2,
} from 'lucide-react';
import { cn } from '@/utils';

export interface ProgressEvent {
  step: number;
  totalSteps: number;
  description: string;
  status: 'pending' | 'in_progress' | 'completed' | 'error';
  timestamp?: Date;
}

interface ProgressTrackerProps {
  progress: ProgressEvent[];
  onClose?: () => void;
  className?: string;
}

const StepIcon: React.FC<{
  status: ProgressEvent['status'];
}> = ({ status }) => {
  switch (status) {
    case 'in_progress':
      return (
        <div className="relative flex items-center justify-center">
          {/* Main pulsating dot */}
          <div className="w-3 h-3 bg-blue-500 rounded-full animate-pulse z-10" />
          {/* Ping effect */}
          <div className="absolute inset-0 w-3 h-3 bg-blue-500 rounded-full animate-ping opacity-75" />
        </div>
      );

    case 'completed':
      return <CheckCircle className="w-4 h-4 text-green-500" />;

    case 'error':
      return <XCircle className="w-4 h-4 text-red-500" />;

    case 'pending':
    default:
      return (
        <div className="w-3 h-3 bg-gray-300 dark:bg-gray-600 rounded-full" />
      );
  }
};

const ProgressStep: React.FC<{ event: ProgressEvent }> = ({ event }) => {
  const { step, totalSteps, description, status } = event;

  // Status-based text color
  const textColorClass =
    status === 'in_progress'
      ? 'text-gray-900 dark:text-gray-100'
      : status === 'completed'
      ? 'text-green-700 dark:text-green-300'
      : status === 'error'
      ? 'text-red-700 dark:text-red-300'
      : 'text-gray-500 dark:text-gray-400';

  return (
    <div className="flex items-center gap-3 py-2 px-1">
      {/* Icon */}
      <div className="flex-shrink-0">
        <StepIcon status={status} />
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className={cn('text-sm font-medium truncate', textColorClass)}>
          {description}
        </div>
        <div className="text-xs text-gray-500 dark:text-gray-500">
          Step {step}/{totalSteps}
        </div>
      </div>
    </div>
  );
};

export const ProgressTracker: React.FC<ProgressTrackerProps> = ({
  progress,
  onClose,
  className,
}) => {
  // Don't render if no progress
  if (progress.length === 0) {
    return null;
  }

  // Calculate overall progress
  const completedCount = progress.filter((p) => p.status === 'completed').length;
  const totalCount = progress.length;
  const progressPercent = totalCount > 0 ? (completedCount / totalCount) * 100 : 0;

  // Check if any errors
  const hasErrors = progress.some((p) => p.status === 'error');

  // Check if completed
  const isCompleted = progress.every(
    (p) => p.status === 'completed' || p.status === 'error'
  );

  return (
    <div
      className={cn(
        'fixed bottom-4 right-4 bg-white dark:bg-gray-800',
        'rounded-lg shadow-2xl border border-gray-200 dark:border-gray-700',
        'p-4 max-w-md w-96 z-50',
        'animate-in slide-in-from-bottom-4 duration-300',
        className
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-3 pb-3 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center gap-2">
          {isCompleted ? (
            <CheckCircle className="w-5 h-5 text-green-500" />
          ) : hasErrors ? (
            <XCircle className="w-5 h-5 text-red-500" />
          ) : (
            <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />
          )}
          <span className="font-semibold text-gray-900 dark:text-white">
            {isCompleted
              ? hasErrors
                ? 'Completed with Errors'
                : 'Completed'
              : 'In Progress'}
          </span>
        </div>

        {/* Close button (if onClose provided) */}
        {onClose && isCompleted && (
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
            aria-label="Close progress tracker"
          >
            <XCircle className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Progress Bar */}
      <div className="mb-4">
        <div className="flex items-center justify-between text-xs text-gray-600 dark:text-gray-400 mb-1">
          <span>Overall Progress</span>
          <span className="font-mono">
            {completedCount}/{totalCount}
          </span>
        </div>
        <div className="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
          <div
            className={cn(
              'h-full transition-all duration-500 rounded-full',
              hasErrors
                ? 'bg-red-500'
                : isCompleted
                ? 'bg-green-500'
                : 'bg-blue-500'
            )}
            style={{ width: `${progressPercent}%` }}
          />
        </div>
      </div>

      {/* Progress Steps */}
      <div className="space-y-1 max-h-64 overflow-y-auto">
        {progress.map((event, index) => (
          <ProgressStep key={index} event={event} />
        ))}
      </div>

      {/* Footer (optional timestamp) */}
      {progress.length > 0 && progress[0].timestamp && (
        <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-500">
            <Clock className="w-3 h-3" />
            <span>
              Started {progress[0].timestamp.toLocaleTimeString()}
            </span>
          </div>
        </div>
      )}
    </div>
  );
};
