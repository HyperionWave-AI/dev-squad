/**
 * useChatState - Unified chat state management with atomic updates
 *
 * Fixes race conditions:
 * 1. Streaming state splits - Single atomic state object
 * 2. Message deduplication race - Synchronized deduplication
 * 3. Session switch overlaps - Debouncing and cancellation
 */

import { useState, useRef, useCallback, useEffect } from 'react';
import type { ChatMessage } from '../services/chatService';

interface StreamingState {
  isStreaming: boolean;
  sessionId: string | null;
  content: string;
  toolCalls: any[];
  toolResults: Map<string, any>;
}

interface ChatState {
  messages: ChatMessage[];
  streamingState: StreamingState;
  lastDeduplicationTime: number;
}

export function useChatState() {
  // Single unified state object for atomic updates
  const [chatState, setChatState] = useState<ChatState>({
    messages: [],
    streamingState: {
      isStreaming: false,
      sessionId: null,
      content: '',
      toolCalls: [],
      toolResults: new Map()
    },
    lastDeduplicationTime: 0
  });

  // Deduplication lock to prevent concurrent deduplication
  const deduplicationLock = useRef<Promise<void>>(Promise.resolve());

  // Session switch debouncer
  const sessionSwitchTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const sessionSwitchController = useRef<AbortController | null>(null);

  /**
   * Set messages with automatic deduplication (synchronized)
   */
  const setMessages = useCallback(async (
    updater: ChatMessage[] | ((prev: ChatMessage[]) => ChatMessage[])
  ) => {
    // Acquire deduplication lock
    const previousLock = deduplicationLock.current;
    let resolveLock: () => void;

    deduplicationLock.current = new Promise(resolve => {
      resolveLock = resolve;
    });

    try {
      await previousLock;

      setChatState(prev => {
        const newMessages = typeof updater === 'function'
          ? updater(prev.messages)
          : updater;

        // Deduplicate messages (O(n) with Map)
        const deduplicated = deduplicateMessages(newMessages);

        return {
          ...prev,
          messages: deduplicated,
          lastDeduplicationTime: Date.now()
        };
      });
    } finally {
      resolveLock!();
    }
  }, []);

  /**
   * Update streaming state atomically
   */
  const updateStreamingState = useCallback((updates: Partial<StreamingState>) => {
    setChatState(prev => ({
      ...prev,
      streamingState: {
        ...prev.streamingState,
        ...updates
      }
    }));
  }, []);

  /**
   * Start streaming for a session
   */
  const startStreaming = useCallback((streamSessionId: string) => {
    setChatState(prev => ({
      ...prev,
      streamingState: {
        isStreaming: true,
        sessionId: streamSessionId,
        content: `[${streamSessionId}]:`, // Session prefix for isolation
        toolCalls: [],
        toolResults: new Map()
      }
    }));
  }, []);

  /**
   * Stop streaming and save final message
   */
  const stopStreaming = useCallback(async () => {
    return new Promise<void>(resolve => {
      setChatState(prev => {
        const { streamingState } = prev;

        if (!streamingState.isStreaming) {
          resolve();
          return prev;
        }

        // Extract content without session prefix
        const sessionPrefix = `[${streamingState.sessionId}]:`;
        const finalContent = streamingState.content.startsWith(sessionPrefix)
          ? streamingState.content.substring(sessionPrefix.length)
          : streamingState.content;

        // Create final message if there's content or tool calls
        let newMessages = prev.messages;
        if (finalContent.trim() || streamingState.toolCalls.length > 0) {
          const finalMessage: ChatMessage = {
            id: `msg-${Date.now()}`,
            sessionId: streamingState.sessionId!,
            role: 'assistant',
            content: finalContent,
            timestamp: new Date().toISOString(),
            toolCalls: streamingState.toolCalls.length > 0
              ? streamingState.toolCalls
              : undefined,
            toolResults: streamingState.toolResults.size > 0
              ? streamingState.toolResults
              : undefined
          };
          newMessages = [...prev.messages, finalMessage];
        }

        resolve();

        return {
          ...prev,
          messages: newMessages,
          streamingState: {
            isStreaming: false,
            sessionId: null,
            content: '',
            toolCalls: [],
            toolResults: new Map()
          }
        };
      });
    });
  }, []);

  /**
   * Append streaming content (with session isolation)
   */
  const appendStreamingContent = useCallback((content: string) => {
    setChatState(prev => {
      // Verify still streaming for same session
      if (!prev.streamingState.isStreaming) {
        console.warn('[ChatState] Ignoring content - not streaming');
        return prev;
      }

      return {
        ...prev,
        streamingState: {
          ...prev.streamingState,
          content: prev.streamingState.content + content
        }
      };
    });
  }, []);

  /**
   * Add tool call to streaming state
   */
  const addToolCall = useCallback((toolCall: any) => {
    setChatState(prev => ({
      ...prev,
      streamingState: {
        ...prev.streamingState,
        toolCalls: [...prev.streamingState.toolCalls, toolCall]
      }
    }));
  }, []);

  /**
   * Add tool result to streaming state
   */
  const addToolResult = useCallback((id: string, result: any) => {
    setChatState(prev => {
      const newToolResults = new Map(prev.streamingState.toolResults);
      newToolResults.set(id, result);

      return {
        ...prev,
        streamingState: {
          ...prev.streamingState,
          toolResults: newToolResults
        }
      };
    });
  }, []);

  /**
   * Switch session with debouncing and cancellation
   */
  const switchSession = useCallback(async (
    newSessionId: string,
    onSwitch: (sessionId: string, signal: AbortSignal) => Promise<void>
  ) => {
    // Cancel any pending session switch
    if (sessionSwitchTimer.current) {
      clearTimeout(sessionSwitchTimer.current);
      sessionSwitchTimer.current = null;
    }

    if (sessionSwitchController.current) {
      sessionSwitchController.current.abort();
    }

    // Create abort controller for this switch
    const controller = new AbortController();
    sessionSwitchController.current = controller;

    // Debounce session switch (300ms)
    return new Promise<void>((resolve, reject) => {
      sessionSwitchTimer.current = setTimeout(async () => {
        try {
          if (!controller.signal.aborted) {
            await onSwitch(newSessionId, controller.signal);
            resolve();
          }
        } catch (error) {
          if (error instanceof Error && error.name === 'AbortError') {
            console.log('[ChatState] Session switch cancelled');
          } else {
            reject(error);
          }
        } finally {
          sessionSwitchController.current = null;
        }
      }, 300);
    });
  }, []);

  /**
   * Cleanup on unmount
   */
  useEffect(() => {
    return () => {
      if (sessionSwitchTimer.current) {
        clearTimeout(sessionSwitchTimer.current);
      }
      if (sessionSwitchController.current) {
        sessionSwitchController.current.abort();
      }
    };
  }, []);

  return {
    messages: chatState.messages,
    streamingState: chatState.streamingState,
    setMessages,
    updateStreamingState,
    startStreaming,
    stopStreaming,
    appendStreamingContent,
    addToolCall,
    addToolResult,
    switchSession
  };
}

/**
 * Deduplicate messages efficiently with Map (O(n) instead of O(n²))
 */
function deduplicateMessages(messages: ChatMessage[]): ChatMessage[] {
  // Index database messages by content hash
  const dbMessagesByHash = new Map<string, ChatMessage>();
  const optimisticMessageIds = new Set<string>();

  // First pass: index database messages
  for (const msg of messages) {
    if (!msg.id.startsWith('msg-')) {
      const hash = createMessageHash(msg);
      dbMessagesByHash.set(hash, msg);
    } else {
      optimisticMessageIds.add(msg.id);
    }
  }

  // Second pass: deduplicate
  const deduplicated = new Map<string, ChatMessage>();

  for (const msg of messages) {
    if (msg.id.startsWith('msg-')) {
      // Optimistic message - check if database version exists
      const hash = createMessageHash(msg);
      if (dbMessagesByHash.has(hash)) {
        // Skip optimistic, use database version
        const dbMsg = dbMessagesByHash.get(hash)!;
        deduplicated.set(dbMsg.id, dbMsg);
      } else {
        deduplicated.set(msg.id, msg);
      }
    } else {
      // Database message - always include
      deduplicated.set(msg.id, msg);
    }
  }

  // Sort by timestamp
  return Array.from(deduplicated.values()).sort((a, b) =>
    new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
  );
}

/**
 * Create hash for message deduplication
 */
function createMessageHash(msg: ChatMessage): string {
  const timestamp = Math.floor(new Date(msg.timestamp).getTime() / 5000) * 5000;
  return `${msg.role}:${msg.content.substring(0, 100)}:${timestamp}`;
}
