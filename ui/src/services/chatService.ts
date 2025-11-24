/**
 * Chat Service API Client
 *
 * Provides REST API and WebSocket interface for chat functionality.
 * Follows existing restClient.ts patterns with typed responses.
 * 
 * INTEGRATION: Now uses WebSocketManager for robust connection handling with:
 * - Atomic state transitions
 * - Bounded message queue (prevents memory leaks)
 * - Exponential backoff with intelligent retry logic
 * - User-visible connection status indicators
 * - Graceful degradation when WebSocket fails
 * - Correlation IDs for end-to-end request tracing
 */

import { WebSocketManager, ConnectionState } from './WebSocketManager';
import { getCorrelationId, addCorrelationIdToHeaders } from '@/utils/correlationId';
import { retryWithBackoff, DEFAULT_RETRY_CONFIG } from '@/utils/retry';

const BASE_URL = '/api/v1';

// ============================================================
// TYPE DEFINITIONS

export interface ChatSession {
  id: string;
  userId: string;
  companyId: string;
  title: string;
  parentChatId?: string; // For subchats - links to parent session
  activeSubagentId?: string; // Indicates session is being processed by a subagent
  errorPreventionMode: boolean; // Toggle for validation plugin
  complexityAnalysisMode: boolean; // Toggle for complexity analysis and task splitting
  createdAt: string;
  updatedAt: string;
}

export interface ChatMessage {
  id: string;
  sessionId: string;
  role: 'user' | 'assistant' | 'system' | 'tool_call' | 'tool_result';
  content: string;
  timestamp: string;
  // Plural arrays for backward compatibility with WebSocket streaming
  toolCalls?: ToolCall[];
  toolResults?: Map<string, ToolResult>;
  // Singular objects for API responses (tool_call and tool_result messages)
  toolCall?: {
    id: string;
    name: string;
    args: Record<string, any>;
  };
  toolResult?: {
    id: string;
    name: string;
    output: any;
    error: string | null;
    durationMs: number;
  };
  // Plugin metadata
  metadata?: Record<string, any>;
}

export interface ToolCall {
  id: string;
  tool: string;
  args: Record<string, any>;
  timestamp: Date;
}

export interface ToolResult {
  id: string;
  tool: string;
  result: any;
  error: string | null;
  durationMs: number;
  // Plugin metadata
  metadata?: Record<string, any>;
}

export interface StreamMessage {
  type: 'token' | 'tool_call' | 'tool_result' | 'done' | 'error' | 'message_saved' | 'user_message' | 'session_created';
  content?: string;
  toolCall?: {
    tool: string;
    args: Record<string, any>;
    id: string;
  };
  toolResult?: {
    id: string;
    result: any;
    error: string | null;
    durationMs: number;
  };
  error?: string;
}

// ============================================================
// REST API FUNCTIONS
// ============================================================

/**
 * Generic fetch wrapper with error handling and correlation IDs
 */
async function fetchJSON<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const url = `${BASE_URL}${endpoint}`;
  const correlationId = getCorrelationId();

  try {
    const baseHeaders: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    // Merge options headers if they exist
    if (options?.headers) {
      const optionsHeaders = new Headers(options.headers);
      optionsHeaders.forEach((value, key) => {
        baseHeaders[key] = value;
      });
    }

    const response = await fetch(url, {
      ...options,
      headers: addCorrelationIdToHeaders(baseHeaders),
    });

    if (!response.ok) {
      const errorText = await response.text();
      let errorMessage: string;
      try {
        const errorData = JSON.parse(errorText);
        errorMessage = errorData.error || errorData.message || `HTTP ${response.status}`;
      } catch {
        errorMessage = errorText || `HTTP ${response.status}`;
      }
      console.error(`[${correlationId}] API Error:`, errorMessage);
      throw new Error(`API Error: ${errorMessage}`);
    }

    return await response.json();
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error(`Request failed: ${String(error)}`);
  }
}

/**
 * Create a new chat session
 */
export async function createSession(title: string): Promise<ChatSession> {
  const response = await fetchJSON<{ session: ChatSession }>('/chat/sessions', {
    method: 'POST',
    body: JSON.stringify({ title }),
  });

  if (!response.session) {
    throw new Error('Failed to create session');
  }

  return response.session;
}

/**
 * Get all chat sessions for the current user
 */
export async function getSessions(): Promise<ChatSession[]> {
  const response = await fetchJSON<{ sessions: ChatSession[]; count: number }>('/chat/sessions', {
    method: 'GET',
  });

  return response.sessions || [];
}

/**
 * Get messages for a specific chat session
 */
export async function getMessages(
  sessionId: string,
  limit: number = 50,
  offset: number = 0
): Promise<ChatMessage[]> {
  const queryParams = new URLSearchParams({
    limit: limit.toString(),
    offset: offset.toString(),
  });

  const response = await fetchJSON<{ messages: ChatMessage[] | null; total: number; hasMore: boolean }>(
    `/chat/sessions/${sessionId}/messages?${queryParams}`,
    { method: 'GET' }
  );

  return response.messages || [];
}

/**
 * Delete a chat session
 */
export async function deleteSession(sessionId: string): Promise<void> {
  const response = await fetchJSON<{ success: boolean; message: string }>(
    `/chat/sessions/${sessionId}`,
    { method: 'DELETE' }
  );

  if (!response.success) {
    throw new Error('Failed to delete session');
  }
}

/**
 * Update a chat session title
 */
export async function updateSession(sessionId: string, title: string): Promise<ChatSession> {
  const response = await fetchJSON<{ session: ChatSession }>(
    `/chat/sessions/${sessionId}`,
    {
      method: 'PUT',
      body: JSON.stringify({ title }),
    }
  );

  if (!response.session) {
    throw new Error('Failed to update session');
  }

  return response.session;
}

/**
 * Update error prevention mode for a chat session
 */
export async function updateErrorPreventionMode(
  sessionId: string,
  enabled: boolean
): Promise<{ success: boolean; errorPreventionMode: boolean }> {
  const response = await fetchJSON<{
    success: boolean;
    errorPreventionMode: boolean;
    session: ChatSession;
  }>(`/chat/sessions/${sessionId}/error-prevention`, {
    method: 'PATCH',
    body: JSON.stringify({ enabled }),
  });

  return {
    success: response.success,
    errorPreventionMode: response.errorPreventionMode,
  };
}

/**
 * Update complexity analysis mode for a session
 */
export async function updateComplexityAnalysisMode(
  sessionId: string,
  enabled: boolean
): Promise<{ success: boolean; complexityAnalysisMode: boolean }> {
  const response = await fetchJSON<{
    success: boolean;
    complexityAnalysisMode: boolean;
    session: ChatSession;
  }>(`/chat/sessions/${sessionId}/complexity-analysis`, {
    method: 'PATCH',
    body: JSON.stringify({ enabled }),
  });

  return {
    success: response.success,
    complexityAnalysisMode: response.complexityAnalysisMode,
  };
}

// ============================================================
// WEBSOCKET STREAM CONNECTION (USING WEBSOCKETMANAGER)
// ============================================================

export interface StreamCallbacks {
  onMessage: (content: string, done: boolean) => void;
  onToolCall?: (tool: string, args: Record<string, any>, id: string) => void;
  onToolResult?: (id: string, tool: string, result: any, error: string | null, durationMs: number) => void;
  onMessageSaved?: (databaseId: string) => void;
  onUserMessage?: (content: string) => void; // Bug #1 fix: handle user message echo
  onSessionCreated?: (subchatId: string) => void; // Event-driven session updates
  onError: (error: Error) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onConnectionStateChange?: (state: ConnectionState) => void; // NEW: Connection state visibility
}

export interface ChatStreamConnection {
  manager: WebSocketManager;
  disconnect: () => Promise<void>;
  sendMessage: (content: string) => Promise<void>;
  getState: () => ConnectionState;
  isConnected: () => boolean;
}

/**
 * Connect to chat stream WebSocket using WebSocketManager
 * 
 * Returns connection object with:
 * - manager: WebSocketManager instance for advanced control
 * - disconnect: Graceful disconnect function
 * - sendMessage: Send message with automatic queuing and retry logic
 * - getState: Get current connection state
 * - isConnected: Check if actively connected
 */
export function connectChatStream(
  sessionId: string,
  callbacks: StreamCallbacks
): ChatStreamConnection {
  const manager = new WebSocketManager();
  const correlationId = getCorrelationId();

  // Adapt WebSocketManager callbacks to StreamCallbacks
  const wsCallbacks: Record<string, Function> = {
    onOpen: () => {
      console.log(`[${correlationId}] WebSocket connected via WebSocketManager`);
      callbacks.onOpen?.();
      callbacks.onConnectionStateChange?.(ConnectionState.CONNECTED);
    },

    onMessage: (data: StreamMessage) => {
      try {
        switch (data.type) {
          case 'error':
            callbacks.onError(new Error(data.error || 'Unknown error'));
            break;

          case 'token':
            // Streaming token
            callbacks.onMessage(data.content || '', false);
            break;

          case 'tool_call':
            // Tool execution started
            if (data.toolCall && callbacks.onToolCall) {
              callbacks.onToolCall(
                data.toolCall.tool,
                data.toolCall.args,
                data.toolCall.id
              );
            }
            break;

          case 'tool_result':
            // Tool execution completed
            if (data.toolResult && callbacks.onToolResult) {
              // Note: tool name should be included in toolResult from backend
              const toolName = (data.toolResult as any).tool || 'unknown';
              callbacks.onToolResult(
                data.toolResult.id,
                toolName,
                data.toolResult.result,
                data.toolResult.error,
                data.toolResult.durationMs
              );
            }
            break;

          case 'done':
            // Stream complete
            callbacks.onMessage('', true);
            break;

          case 'message_saved':
            // Message saved with database ID
            if (data.content && callbacks.onMessageSaved) {
              callbacks.onMessageSaved(data.content);
            }
            break;

          case 'user_message':
            // User message echo from server
            if (callbacks.onUserMessage && data.content) {
              callbacks.onUserMessage(data.content);
            }
            break;

          case 'session_created':
            // New subchat session created - refresh sessions list
            if (callbacks.onSessionCreated && data.content) {
              callbacks.onSessionCreated(data.content);
            }
            break;
        }
      } catch (error) {
        callbacks.onError(
          error instanceof Error ? error : new Error('Failed to parse message')
        );
      }
    },

    onError: (error: Error) => {
      console.error(`[${correlationId}] WebSocket error via WebSocketManager:`, error);
      callbacks.onError(error);
      callbacks.onConnectionStateChange?.(ConnectionState.ERROR);
    },

    onClose: () => {
      console.log(`[${correlationId}] WebSocket closed via WebSocketManager`);
      callbacks.onClose?.();
      callbacks.onConnectionStateChange?.(ConnectionState.DISCONNECTED);
    },
  };

  // Connect using WebSocketManager with atomic state management
  manager.connect(sessionId, wsCallbacks).catch((error) => {
    console.error(`[${correlationId}] Failed to connect WebSocketManager:`, error);
    callbacks.onError(error);
  });

  // Return connection interface with retry-aware sendMessage
  return {
    manager,
    disconnect: async () => {
      await manager.disconnect();
    },
    sendMessage: async (content: string) => {
      try {
        // Retry with exponential backoff for transient errors
        await retryWithBackoff(
          () => manager.sendMessage(content),
          DEFAULT_RETRY_CONFIG,
          (attempt, error, nextDelayMs) => {
            console.warn(
              `[${correlationId}] Message send failed (attempt ${attempt}), retrying in ${nextDelayMs}ms:`,
              error.message
            );
          }
        );
      } catch (error) {
        throw error instanceof Error ? error : new Error('Failed to send message');
      }
    },
    getState: () => manager.getState(),
    isConnected: () => manager.isConnected(),
  };
}

/**
 * Export ConnectionState for UI components to use
 */
export { ConnectionState };
