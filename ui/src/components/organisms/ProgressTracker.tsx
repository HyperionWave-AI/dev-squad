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
 * - Separate tool_call and tool_result message handling
 * - Progress event persistence
 * - Typing indicator support
 */

import React, { useState, useEffect } from 'react';
import {
  CheckCircle,
  XCircle,
  Clock,
  Loader2,
  Wrench,
  MessageSquare,
} from 'lucide-react';
import { cn } from '@/utils';

// Enhanced message types for tool calls and results
export interface ToolCallMessage {
  id: string;
  type: 'tool_call';
  toolName: string;
  parameters: Record<string, any>;
  timestamp: Date;
  status: 'pending' | 'in_progress' | 'completed' | 'error';
}

export interface ToolResultMessage {
  id: string;
  type: 'tool_result';
  toolCallId: string;
  result: any;
  error?: string;
  timestamp: Date;
  status: 'completed' | 'error';
}

// Enhanced progress event with dedicated progress type
export interface ProgressEvent {
  id: string;
  type: 'progress';
  step: number;
  totalSteps: number;
  description: string;
  status: 'pending' | 'in_progress' | 'completed' | 'error';
  timestamp: Date;
}

// Typing indicator event
export interface TypingEvent {
  id: string;
  type: 'typing';
  description: string;
  timestamp: Date;
}

// Union type for all trackable events
export type TrackableEvent = ToolCallMessage | ToolResultMessage | ProgressEvent | TypingEvent;

interface ProgressTrackerProps {
  events: TrackableEvent[];
  onClose?: () => void;
  className?: string;
  showTypingIndicator?: boolean;
  onPersistEvent?: (event: TrackableEvent) => void; // Callback for persistence
}

const EventIcon: React.FC<{
  event: TrackableEvent;
}> = ({ event }) => {
  switch (event.type) {
    case 'tool_call':
      return event.status === 'in_progress' ? (
        <div className="relative flex items-center justify-center">
          <Wrench className="w-3 h-3 text-blue-500 animate-pulse z-10" />
          <div className="absolute inset-0 w-3 h-3 bg-blue-500 rounded-full animate-ping opacity-75" />
        </div>
      ) : event.status === 'completed' ? (
        <CheckCircle className="w-4 h-4 text-green-500" />
      ) : event.status === 'error' ? (
        <XCircle className="w-4 h-4 text-red-500" />
      ) : (
        <Wrench className="w-3 h-3 text-gray-400" />
      );

    case 'tool_result':
      return event.status === 'error' ? (
        <XCircle className="w-4 h-4 text-red-500" />
      ) : (
        <CheckCircle className="w-4 h-4 text-green-500" />
      );

    case 'progress':
      return event.status === 'in_progress' ? (
        <div className="relative flex items-center justify-center">
          <div className="w-3 h-3 bg-blue-500 rounded-full animate-pulse z-10" />
          <div className="absolute inset-0 w-3 h-3 bg-blue-500 rounded-full animate-ping opacity-75" />
        </div>
      ) : event.status === 'completed' ? (
        <CheckCircle className="w-4 h-4 text-green-500" />
      ) : event.status === 'error' ? (
        <XCircle className="w-4 h-4 text-red-500" />
      ) : (
        <div className="w-3 h-3 bg-gray-300 dark:bg-gray-600 rounded-full" />
      );

    case 'typing':
      return (
        <div className="relative flex items-center justify-center">
          <MessageSquare className="w-3 h-3 text-blue-500 animate-pulse z-10" />
          <div className="absolute inset-0 w-3 h-3 bg-blue-500 rounded-full animate-ping opacity-75" />
        </div>
      );

    default:
      return <div className="w-3 h-3 bg-gray-300 dark:bg-gray-600 rounded-full" />;
  }
};

const EventItem: React.FC<{ 
  event: TrackableEvent; 
  index: number;
  totalEvents: number;
}> = ({ event, index, totalEvents }) => {
  const getEventDescription = () => {
    switch (event.type) {
      case 'tool_call':
        return `Calling ${event.toolName}`;
      case 'tool_result':
        return event.error ? `Tool failed: ${event.error}` : 'Tool completed';
      case 'progress':
        return event.description;
      case 'typing':
        return event.description;
      default:
        return 'Unknown event';
    }
  };

  const getEventStatus = () => {
    switch (event.type) {
      case 'tool_call':
      case 'progress':
        return event.status;
      case 'tool_result':
        return event.status;
      case 'typing':
        return 'in_progress';
      default:
        return 'pending';
    }
  };

  const status = getEventStatus();
  
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
        <EventIcon event={event} />
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className={cn('text-sm font-medium truncate', textColorClass)}>
          {getEventDescription()}
        </div>
        <div className="text-xs text-gray-500 dark:text-gray-500">
          {event.type === 'progress' 
            ? `Step ${event.step}/${event.totalSteps}`
            : `Event ${index + 1}/${totalEvents}`
          }
          {event.type === 'tool_call' && event.parameters && (
            <span className="ml-2 text-gray-400">
              {Object.keys(event.parameters).length} params
            </span>
          )}
        </div>
      </div>
    </div>
  );
};

// Progress persistence utilities
const PROGRESS_STORAGE_KEY = 'progress_tracker_events';

const saveEventsToStorage = (events: TrackableEvent[]) => {
  try {
    localStorage.setItem(PROGRESS_STORAGE_KEY, JSON.stringify(events.map(event => ({
      ...event,
      timestamp: event.timestamp.toISOString()
    }))));
  } catch (error) {
    console.warn('Failed to save progress events to localStorage:', error);
  }
};

const loadEventsFromStorage = (): TrackableEvent[] => {
  try {
    const stored = localStorage.getItem(PROGRESS_STORAGE_KEY);
    if (!stored) return [];
    
    const parsed = JSON.parse(stored);
    return parsed.map((event: any) => ({
      ...event,
      timestamp: new Date(event.timestamp)
    }));
  } catch (error) {
    console.warn('Failed to load progress events from localStorage:', error);
    return [];
  }
};

const clearEventsFromStorage = () => {
  try {
    localStorage.removeItem(PROGRESS_STORAGE_KEY);
  } catch (error) {
    console.warn('Failed to clear progress events from localStorage:', error);
  }
};

export const ProgressTracker: React.FC<ProgressTrackerProps> = ({
  events,
  onClose,
  className,
  showTypingIndicator = false,
  onPersistEvent,
}) => {
  const [persistedEvents, setPersistedEvents] = useState<TrackableEvent[]>([]);

  // Load persisted events on mount
  useEffect(() => {
    const stored = loadEventsFromStorage();
    setPersistedEvents(stored);
  }, []);

  // Combine current events with persisted events, avoiding duplicates
  const allEvents = React.useMemo(() => {
    const eventMap = new Map<string, TrackableEvent>();
    
    // Add persisted events first
    persistedEvents.forEach(event => {
      eventMap.set(event.id, event);
    });
    
    // Add current events, overwriting any duplicates
    events.forEach(event => {
      eventMap.set(event.id, event);
    });
    
    return Array.from(eventMap.values()).sort((a, b) => 
      a.timestamp.getTime() - b.timestamp.getTime()
    );
  }, [events, persistedEvents]);

  // Persist events when they change
  useEffect(() => {
    if (allEvents.length > 0) {
      saveEventsToStorage(allEvents);
      
      // Call external persistence callback if provided
      allEvents.forEach(event => {
        onPersistEvent?.(event);
      });
    }
  }, [allEvents, onPersistEvent]);

  // Show typing indicator if enabled and no events yet
  const shouldShowTypingIndicator = showTypingIndicator && allEvents.length === 0;

  // Don't render if no events and no typing indicator
  if (allEvents.length === 0 && !shouldShowTypingIndicator) {
    return null;
  }

  // Calculate overall progress
  const completedEvents = allEvents.filter((e) => 
    (e.type === 'progress' && e.status === 'completed') ||
    (e.type === 'tool_result' && e.status === 'completed') ||
    (e.type === 'tool_call' && e.status === 'completed')
  ).length;
  
  const totalEvents = allEvents.filter(e => e.type !== 'typing').length;
  const progressPercent = totalEvents > 0 ? (completedEvents / totalEvents) * 100 : 0;

  // Check if any errors
  const hasErrors = allEvents.some((e) => 
    (e.type === 'progress' && e.status === 'error') ||
    (e.type === 'tool_result' && e.status === 'error') ||
    (e.type === 'tool_call' && e.status === 'error')
  );

  // Check if completed
  const isCompleted = totalEvents > 0 && allEvents
    .filter(e => e.type !== 'typing')
    .every((e) => 
      (e.type === 'progress' && (e.status === 'completed' || e.status === 'error')) ||
      (e.type === 'tool_result' && (e.status === 'completed' || e.status === 'error')) ||
      (e.type === 'tool_call' && (e.status === 'completed' || e.status === 'error'))
    );

  const handleClose = () => {
    clearEventsFromStorage();
    onClose?.();
  };

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
          {shouldShowTypingIndicator ? (
            <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />
          ) : isCompleted ? (
            <CheckCircle className="w-5 h-5 text-green-500" />
          ) : hasErrors ? (
            <XCircle className="w-5 h-5 text-red-500" />
          ) : (
            <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />
          )}
          <span className="font-semibold text-gray-900 dark:text-white">
            {shouldShowTypingIndicator
              ? 'Starting...'
              : isCompleted
              ? hasErrors
                ? 'Completed with Errors'
                : 'Completed'
              : 'In Progress'}
          </span>
        </div>

        {/* Close button (if onClose provided) */}
        {onClose && (isCompleted || shouldShowTypingIndicator) && (
          <button
            onClick={handleClose}
            className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
            aria-label="Close progress tracker"
          >
            <XCircle className="w-4 h-4" />
          </button>
        )}
      </div>

      {/* Typing Indicator */}
      {shouldShowTypingIndicator && (
        <div className="flex items-center gap-3 py-4">
          <div className="relative flex items-center justify-center">
            <MessageSquare className="w-4 h-4 text-blue-500 animate-pulse z-10" />
            <div className="absolute inset-0 w-4 h-4 bg-blue-500 rounded-full animate-ping opacity-75" />
          </div>
          <span className="text-sm text-gray-600 dark:text-gray-400">
            Preparing response...
          </span>
        </div>
      )}

      {/* Progress Bar */}
      {totalEvents > 0 && (
        <div className="mb-4">
          <div className="flex items-center justify-between text-xs text-gray-600 dark:text-gray-400 mb-1">
            <span>Overall Progress</span>
            <span className="font-mono">
              {completedEvents}/{totalEvents}
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
      )}

      {/* Events List */}
      {allEvents.length > 0 && (
        <div className="space-y-1 max-h-64 overflow-y-auto">
          {allEvents.map((event, index) => (
            <EventItem 
              key={event.id} 
              event={event} 
              index={index}
              totalEvents={allEvents.length}
            />
          ))}
        </div>
      )}

      {/* Footer (optional timestamp) */}
      {allEvents.length > 0 && allEvents[0].timestamp && (
        <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700">
          <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-500">
            <Clock className="w-3 h-3" />
            <span>
              Started {allEvents[0].timestamp.toLocaleTimeString()}
            </span>
          </div>
        </div>
      )}
    </div>
  );
};

// Export utility functions for external use
export {
  saveEventsToStorage,
  loadEventsFromStorage,
  clearEventsFromStorage,
};