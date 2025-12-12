/**
 * Chat State Manager
 * 
 * Centralizes all chat state management in a single, predictable store.
 * Eliminates scattered state across React, WebSocket, refs, and Maps.
 * 
 * Features:
 * - Single source of truth for all chat state
 * - State validation on every update
 * - Change listeners for React integration
 * - State synchronization between React and refs
 * - State rollback on errors
 * - Time-travel debugging support
 */

import { type ChatMessage as ChatMessageType, type ToolCall, type ToolResult, ConnectionState } from '@/services/chatService';
import { type ContextMetadata } from '@/types/chat';

export interface ChatState {
  // Session state
  sessions: SessionItem[];
  activeSessionId: string | null;

  // Messages state
  messages: ChatMessageType[];

  // Streaming state
  isStreaming: boolean;
  streamingSessionId: string | null;
  streamingContent: string;
  pendingToolCalls: Set<string>;
  streamingToolCalls: ToolCall[];
  streamingToolResults: Map<string, ToolResult>;

  // Connection state
  connectionState: ConnectionState;
  lastError: string | null;

  // Context state
  contextMetadata: ContextMetadata | null;
  contextWarningShown: boolean;
  contextCriticalShown: boolean;
  isContextFull: boolean;
  lastContextUpdate: number; // timestamp

  // UI state
  metricsDrawerOpen: boolean;
  errorPreventionMode: boolean;
  complexityAnalysisMode: boolean;
}

export interface SessionItem {
  id: string;
  title: string;
  timestamp: Date | string;
  messageCount: number;
  lastMessage?: string;
  activeSubagentId?: string;
  activeSubagentName?: string;
  parentSessionId?: string;
  isSubchat?: boolean;
  errorPreventionMode?: boolean;
  complexityAnalysisMode?: boolean;
}

export type StateChangeListener = (state: ChatState, previousState: ChatState) => void;

export class ChatStateManager {
  private state: ChatState;
  private previousStates: ChatState[] = [];
  private listeners: Set<StateChangeListener> = new Set();
  private maxHistorySize: number = 50;

  constructor(initialState?: Partial<ChatState>) {
    this.state = {
      sessions: [],
      activeSessionId: null,
      messages: [],
      isStreaming: false,
      streamingSessionId: null,
      streamingContent: '',
      pendingToolCalls: new Set(),
      streamingToolCalls: [],
      streamingToolResults: new Map(),
      connectionState: ConnectionState.DISCONNECTED,
      lastError: null,
      contextMetadata: null,
      contextWarningShown: false,
      contextCriticalShown: false,
      isContextFull: false,
      lastContextUpdate: 0,
      metricsDrawerOpen: false,
      errorPreventionMode: false,
      complexityAnalysisMode: false,
      ...initialState,
    };
  }

  /**
   * Get current state (immutable snapshot)
   */
  getState(): Readonly<ChatState> {
    return Object.freeze({ ...this.state });
  }

  /**
   * Update state with validation
   */
  setState(updates: Partial<ChatState>): void {
    const previousState = { ...this.state };

    // Validate updates
    this.validateUpdates(updates);

    // Apply updates
    this.state = {
      ...this.state,
      ...updates,
    };

    // Save to history for time-travel debugging
    this.previousStates.push(previousState);
    if (this.previousStates.length > this.maxHistorySize) {
      this.previousStates.shift();
    }

    // Notify listeners
    this.notifyListeners(previousState);
  }

  /**
   * Update a specific field
   */
  setField<K extends keyof ChatState>(key: K, value: ChatState[K]): void {
    this.setState({ [key]: value } as Partial<ChatState>);
  }

  /**
   * Add a state change listener
   */
  subscribe(listener: StateChangeListener): () => void {
    this.listeners.add(listener);

    // Return unsubscribe function
    return () => {
      this.listeners.delete(listener);
    };
  }

  /**
   * Get state history for debugging
   */
  getHistory(): ChatState[] {
    return [...this.previousStates, this.state];
  }

  /**
   * Rollback to previous state
   */
  rollback(): void {
    if (this.previousStates.length === 0) {
      console.warn('[ChatStateManager] No previous state to rollback to');
      return;
    }

    const previousState = this.state;
    this.state = this.previousStates.pop()!;
    this.notifyListeners(previousState);
  }

  /**
   * Clear history
   */
  clearHistory(): void {
    this.previousStates = [];
  }

  /**
   * Reset to initial state
   */
  reset(): void {
    const previousState = this.state;
    this.state = {
      sessions: [],
      activeSessionId: null,
      messages: [],
      isStreaming: false,
      streamingSessionId: null,
      streamingContent: '',
      pendingToolCalls: new Set(),
      streamingToolCalls: [],
      streamingToolResults: new Map(),
      connectionState: ConnectionState.DISCONNECTED,
      lastError: null,
      contextMetadata: null,
      contextWarningShown: false,
      contextCriticalShown: false,
      isContextFull: false,
      lastContextUpdate: 0,
      metricsDrawerOpen: false,
      errorPreventionMode: false,
      complexityAnalysisMode: false,
    };
    this.previousStates = [];
    this.notifyListeners(previousState);
  }

  // Private helper methods

  private validateUpdates(updates: Partial<ChatState>): void {
    // Validate sessions
    if (updates.sessions !== undefined) {
      if (!Array.isArray(updates.sessions)) {
        throw new Error('sessions must be an array');
      }
    }

    // Validate messages
    if (updates.messages !== undefined) {
      if (!Array.isArray(updates.messages)) {
        throw new Error('messages must be an array');
      }
    }

    // Validate streaming state consistency
    if (updates.isStreaming !== undefined && updates.isStreaming === false) {
      // When stopping streaming, clear streaming content
      if (updates.streamingContent === undefined) {
        updates.streamingContent = '';
      }
    }

    // Validate connection state
    if (updates.connectionState !== undefined) {
      const validStates = Object.values(ConnectionState);
      if (!validStates.includes(updates.connectionState)) {
        throw new Error(`Invalid connection state: ${updates.connectionState}`);
      }
    }
  }

  private notifyListeners(previousState: ChatState): void {
    this.listeners.forEach(listener => {
      try {
        listener(this.state, previousState);
      } catch (error) {
        console.error('[ChatStateManager] Error in listener:', error);
      }
    });
  }
}

/**
 * Create a singleton instance for the app
 */
let instance: ChatStateManager | null = null;

export function getChatStateManager(): ChatStateManager {
  if (!instance) {
    instance = new ChatStateManager();
  }
  return instance;
}

/**
 * Reset the singleton (useful for testing)
 */
export function resetChatStateManager(): void {
  instance = null;
}
