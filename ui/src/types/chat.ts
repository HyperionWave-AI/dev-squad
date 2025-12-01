/**
 * Chat-related type definitions
 */

export interface Message {
  id: string;
  content: string;
  role: 'user' | 'assistant' | 'system';
  timestamp: string;
  isPending?: boolean; // Whether this message is still being processed (optimistic update)
  metadata?: {
    tool?: string;
    [key: string]: any;
  };
}

export interface Session {
  id: string;
  name: string;
  messages: Message[];
  createdAt: string;
  updatedAt: string;
  parentSessionId?: string;
  isSubchat?: boolean;
  subchats?: Session[];
  metadata?: {
    [key: string]: any;
  };
}

export interface ChatState {
  sessions: Session[];
  currentSession: Session | null;
  isLoading: boolean;
  error: string | null;
}

export interface CreateSessionOptions {
  name?: string;
  parentId?: string;
}

export interface SendMessageOptions {
  sessionId: string;
  content: string;
  role: 'user' | 'assistant' | 'system';
  metadata?: {
    tool?: string;
    [key: string]: any;
  };
}

/**
 * Context metadata for tracking token usage and limits
 */
export interface ContextMetadata {
  tokenCount: number;           // Current tokens used
  maxTokens: number;            // Maximum allowed tokens (e.g., 100,000)
  percentageUsed: number;       // Percentage of max tokens used (0-100)
  isWarning: boolean;           // True if usage is between 80-90%
  isCritical: boolean;          // True if usage is above 90%
  messageCount: number;         // Number of messages in session
  lastUpdated: string;          // ISO timestamp of last update
  canAddMessage: boolean;       // Whether a new message can be added
}

/**
 * Extended stream message with context metadata
 */
export interface StreamMessage {
  type: string;
  content?: string;
  error?: string;
  contextMetadata?: ContextMetadata;
  [key: string]: any;
}

/**
 * Context error with recovery options
 */
export interface ContextError {
  code: string;
  message: string;
  currentTokens: number;
  maxTokens: number;
  percentageUsed: number;
  recoveryOptions: string[];
  suggestedAction: string;
  canArchiveMessages: boolean;
  canSummarize: boolean;
}

/**
 * Archive request for archiving messages
 */
export interface ArchiveRequest {
  sessionId: string;
  messageIds: string[];
  reason?: string;
}

/**
 * Archive response from server
 */
export interface ArchiveResponse {
  success: boolean;
  archivedCount: number;
  archiveId: string;
  timestamp: string;
  message?: string;
}

/**
 * Archived message information
 */
export interface ArchivedMessage {
  id: string;
  sessionId: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  timestamp: string;
  tokenCount: number;
}

/**
 * Archive metadata
 */
export interface ArchiveMetadata {
  archiveId: string;
  sessionId: string;
  messageCount: number;
  totalTokensFreed: number;
  archivedAt: string;
  reason?: string;
}
