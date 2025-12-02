/**
 * Notification Service
 *
 * Handles system notifications for backend events such as:
 * - Context compaction (when context window is compressed)
 * - Tool result deflection (when large tool results are summarized)
 * - Message summarization (when old messages are summarized)
 * - Execution stopped (when user stops AI execution)
 *
 * Uses the sonner toast library for non-blocking, auto-dismissing notifications.
 */

import { toast } from 'sonner';

export interface SystemNotification {
  category: 'compaction' | 'deflection' | 'summarization' | 'execution_stopped';
  title: string;
  message: string;
  severity: 'info' | 'warning' | 'success';
  metadata?: Record<string, unknown>;
}

const ICONS: Record<SystemNotification['category'], string> = {
  compaction: '🗜️',
  deflection: '🛑',
  summarization: '📋',
  execution_stopped: '⏹️',
};

const DURATION_MS: Record<SystemNotification['severity'], number> = {
  info: 5000,
  warning: 6000,
  success: 4000,
};

/**
 * Display a system notification as a toast
 *
 * @param notification - The notification details to display
 */
export function showSystemNotification(notification: SystemNotification): void {
  const icon = ICONS[notification.category] || 'ℹ️';
  const fullTitle = `${icon} ${notification.title}`;
  const duration = DURATION_MS[notification.severity];

  switch (notification.severity) {
    case 'warning':
      toast.warning(fullTitle, {
        description: notification.message,
        duration,
      });
      break;
    case 'success':
      toast.success(fullTitle, {
        description: notification.message,
        duration,
      });
      break;
    case 'info':
    default:
      toast.info(fullTitle, {
        description: notification.message,
        duration,
      });
      break;
  }

  // Log notification for debugging
  console.log(
    `[NotificationService] ${notification.severity.toUpperCase()}: ${notification.title} - ${notification.message}`,
    notification.metadata
  );
}
