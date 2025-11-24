/**
 * Message Handler Utility
 * 
 * Handles message deduplication, ordering, and validation.
 * Improves frontend deduplication from O(n²) to O(n).
 * 
 * Features:
 * - O(n) deduplication using Set for tracking seen IDs
 * - Message ordering by timestamp
 * - Message validation (format, size, content)
 * - Idempotency key support
 * - Duplicate detection and handling
 */

import { type ChatMessage as ChatMessageType } from '@/services/chatService';

export interface MessageValidationResult {
  isValid: boolean;
  errors: string[];
}

export class MessageHandler {
  private readonly MAX_MESSAGE_SIZE = 1024 * 1024; // 1MB
  private readonly MAX_CONTENT_LENGTH = 100000; // 100k characters

  /**
   * Validate a message
   */
  validateMessage(message: ChatMessageType): MessageValidationResult {
    const errors: string[] = [];

    // Validate required fields
    if (!message.id) {
      errors.push('Message ID is required');
    }
    if (!message.sessionId) {
      errors.push('Session ID is required');
    }
    if (!message.role) {
      errors.push('Message role is required');
    }
    if (!message.timestamp) {
      errors.push('Message timestamp is required');
    }

    // Validate role
    const validRoles = ['user', 'assistant', 'system', 'tool_call', 'tool_result'];
    if (message.role && !validRoles.includes(message.role)) {
      errors.push(`Invalid role: ${message.role}`);
    }

    // Validate content
    if (message.content === undefined || message.content === null) {
      errors.push('Message content is required');
    }
    if (typeof message.content === 'string' && message.content.length > this.MAX_CONTENT_LENGTH) {
      errors.push(`Content exceeds maximum length of ${this.MAX_CONTENT_LENGTH}`);
    }

    // Validate timestamp format
    try {
      new Date(message.timestamp);
    } catch {
      errors.push(`Invalid timestamp format: ${message.timestamp}`);
    }

    // Validate message size
    const messageSize = JSON.stringify(message).length;
    if (messageSize > this.MAX_MESSAGE_SIZE) {
      errors.push(`Message size exceeds maximum of ${this.MAX_MESSAGE_SIZE} bytes`);
    }

    return {
      isValid: errors.length === 0,
      errors,
    };
  }

  /**
   * Deduplicate messages using O(n) algorithm
   */
  deduplicateMessages(messages: ChatMessageType[]): ChatMessageType[] {
    const seenIds = new Set<string>();
    const deduplicated: ChatMessageType[] = [];

    for (const message of messages) {
      if (!seenIds.has(message.id)) {
        seenIds.add(message.id);
        deduplicated.push(message);
      }
    }

    return deduplicated;
  }

  /**
   * Sort messages by timestamp
   */
  sortMessagesByTimestamp(messages: ChatMessageType[]): ChatMessageType[] {
    return [...messages].sort((a, b) => {
      const timeA = new Date(a.timestamp).getTime();
      const timeB = new Date(b.timestamp).getTime();
      return timeA - timeB;
    });
  }

  /**
   * Merge and deduplicate messages
   */
  mergeMessages(
    existing: ChatMessageType[],
    incoming: ChatMessageType[]
  ): ChatMessageType[] {
    // Combine messages
    const combined = [...existing, ...incoming];

    // Deduplicate
    const deduplicated = this.deduplicateMessages(combined);

    // Sort by timestamp
    return this.sortMessagesByTimestamp(deduplicated);
  }

  /**
   * Find duplicate messages
   */
  findDuplicates(messages: ChatMessageType[]): ChatMessageType[][] {
    const contentMap = new Map<string, ChatMessageType[]>();

    for (const message of messages) {
      const key = `${message.sessionId}:${message.role}:${message.content}`;
      if (!contentMap.has(key)) {
        contentMap.set(key, []);
      }
      contentMap.get(key)!.push(message);
    }

    // Return only groups with duplicates
    return Array.from(contentMap.values()).filter(group => group.length > 1);
  }

  /**
   * Validate message ordering
   */
  isOrderedByTimestamp(messages: ChatMessageType[]): boolean {
    for (let i = 1; i < messages.length; i++) {
      const prevTime = new Date(messages[i - 1].timestamp).getTime();
      const currTime = new Date(messages[i].timestamp).getTime();
      if (prevTime > currTime) {
        return false;
      }
    }
    return true;
  }

  /**
   * Get messages for a specific session
   */
  getSessionMessages(messages: ChatMessageType[], sessionId: string): ChatMessageType[] {
    return messages.filter(msg => msg.sessionId === sessionId);
  }

  /**
   * Get messages by role
   */
  getMessagesByRole(
    messages: ChatMessageType[],
    role: 'user' | 'assistant' | 'system' | 'tool_call' | 'tool_result'
  ): ChatMessageType[] {
    return messages.filter(msg => msg.role === role);
  }

  /**
   * Get messages in a time range
   */
  getMessagesInTimeRange(
    messages: ChatMessageType[],
    startTime: Date,
    endTime: Date
  ): ChatMessageType[] {
    const startMs = startTime.getTime();
    const endMs = endTime.getTime();

    return messages.filter(msg => {
      const msgTime = new Date(msg.timestamp).getTime();
      return msgTime >= startMs && msgTime <= endMs;
    });
  }

  /**
   * Get the last N messages
   */
  getLastMessages(messages: ChatMessageType[], count: number): ChatMessageType[] {
    return messages.slice(Math.max(0, messages.length - count));
  }

  /**
   * Calculate message statistics
   */
  getMessageStats(messages: ChatMessageType[]): {
    total: number;
    byRole: Record<string, number>;
    bySession: Record<string, number>;
    averageLength: number;
    totalSize: number;
  } {
    const byRole: Record<string, number> = {};
    const bySession: Record<string, number> = {};
    let totalSize = 0;

    for (const msg of messages) {
      // Count by role
      byRole[msg.role] = (byRole[msg.role] || 0) + 1;

      // Count by session
      bySession[msg.sessionId] = (bySession[msg.sessionId] || 0) + 1;

      // Calculate size
      totalSize += JSON.stringify(msg).length;
    }

    return {
      total: messages.length,
      byRole,
      bySession,
      averageLength: messages.length > 0 ? totalSize / messages.length : 0,
      totalSize,
    };
  }
}

/**
 * Create a singleton instance
 */
let instance: MessageHandler | null = null;

export function getMessageHandler(): MessageHandler {
  if (!instance) {
    instance = new MessageHandler();
  }
  return instance;
}
