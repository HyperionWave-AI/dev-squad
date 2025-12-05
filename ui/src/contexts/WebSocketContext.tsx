/**
 * WebSocket Context - App-level WebSocket connection management
 *
 * Provides a SINGLETON WebSocket connection that persists across navigation:
 * - Single WebSocketManager instance for the entire app lifecycle
 * - Connection survives component unmount/remount (navigation)
 * - Session switching handled gracefully (disconnect old, connect new)
 * - Callbacks updated without reconnecting when returning to same session
 * - Automatic message fetch on reconnect to catch missed messages
 *
 * This solves the issue where navigating away from chat page would
 * disconnect the WebSocket and lose real-time message streaming.
 */

import React, { createContext, useContext, useRef, useCallback, useState, useEffect } from 'react';
import type { ReactNode } from 'react';
import { WebSocketManager, ConnectionState } from '../services/WebSocketManager';
import type { StreamMessage, SystemNotification } from '../services/chatService';

// Callbacks that components can register for WebSocket events
export interface WebSocketCallbacks {
  onMessage?: (content: string, done: boolean) => void;
  onToolCall?: (tool: string, args: Record<string, unknown>, id: string) => void;
  onToolResult?: (id: string, tool: string, result: unknown, error: string | null, durationMs: number) => void;
  onMessageSaved?: (databaseId: string) => void;
  onUserMessage?: (content: string) => void;
  onSessionCreated?: (subchatId: string) => void;
  onSystemNotification?: (notification: SystemNotification) => void;
  onStreamingStarted?: (sessionId: string) => void; // Called when AI starts processing (shows "AI is thinking")
  onError?: (error: Error) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onReconnect?: () => void; // Called when reconnecting - component should fetch missed messages
}

interface WebSocketContextValue {
  // Connection management
  connect: (sessionId: string, callbacks: WebSocketCallbacks) => Promise<void>;
  disconnect: () => Promise<void>;
  switchSession: (newSessionId: string, callbacks: WebSocketCallbacks) => Promise<void>;
  sendMessage: (content: string) => Promise<void>;
  stopExecution: () => void;

  // State accessors
  connectionState: ConnectionState;
  currentSessionId: string | null;
  isConnected: boolean;

  // Update callbacks without reconnecting
  updateCallbacks: (callbacks: WebSocketCallbacks) => void;
}

const WebSocketContext = createContext<WebSocketContextValue | undefined>(undefined);

interface WebSocketProviderProps {
  children: ReactNode;
}

export const WebSocketProvider: React.FC<WebSocketProviderProps> = ({ children }) => {
  // Single WebSocketManager instance for the ENTIRE app lifecycle
  // This ref persists across all renders and never gets recreated
  const managerRef = useRef<WebSocketManager | null>(null);

  // Lazy initialization of WebSocketManager
  const getManager = useCallback(() => {
    if (!managerRef.current) {
      console.log('[WebSocketContext] Creating singleton WebSocketManager');
      managerRef.current = new WebSocketManager();
    }
    return managerRef.current;
  }, []);

  // Current callbacks (updated by active component without reconnecting)
  const callbacksRef = useRef<WebSocketCallbacks>({});

  // Track if we've set up the adapter callbacks (only do once per connection)
  const adapterSetupRef = useRef<string | null>(null);

  // State for consumers
  const [connectionState, setConnectionState] = useState<ConnectionState>(ConnectionState.DISCONNECTED);
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);

  // Cleanup on unmount (app close)
  useEffect(() => {
    return () => {
      console.log('[WebSocketContext] App unmounting, cleaning up WebSocket');
      if (managerRef.current) {
        managerRef.current.disconnect().catch(console.error);
      }
    };
  }, []);

  /**
   * Create adapter callbacks that delegate to callbacksRef
   * These are stable and route to whatever callbacks are currently registered
   */
  const createAdapterCallbacks = useCallback((sessionId: string) => {
    return {
      onOpen: () => {
        console.log('[WS] ✅ Connected to session:', sessionId);
        setConnectionState(ConnectionState.CONNECTED);
        setCurrentSessionId(sessionId);
        callbacksRef.current.onOpen?.();
      },

      onMessage: (data: StreamMessage) => {
        // Only log session info for debugging
        if (data.type === 'token') {
          console.log('[WS] 📨 Message for session:', sessionId);
        }
        try {
          switch (data.type) {
            case 'error':
              callbacksRef.current.onError?.(new Error(data.error || 'Unknown error'));
              break;

            case 'token':
              callbacksRef.current.onMessage?.(data.content || '', false);
              break;

            case 'tool_call':
              if (data.toolCall) {
                callbacksRef.current.onToolCall?.(
                  data.toolCall.tool,
                  data.toolCall.args,
                  data.toolCall.id
                );
              }
              break;

            case 'tool_result':
              if (data.toolResult) {
                const toolName = (data.toolResult as { tool?: string }).tool || 'unknown';
                callbacksRef.current.onToolResult?.(
                  data.toolResult.id,
                  toolName,
                  data.toolResult.result,
                  data.toolResult.error,
                  data.toolResult.durationMs
                );
              }
              break;

            case 'done':
              callbacksRef.current.onMessage?.('', true);
              break;

            case 'message_saved':
              if (data.content) {
                callbacksRef.current.onMessageSaved?.(data.content);
              }
              break;

            case 'user_message':
              if (data.content) {
                callbacksRef.current.onUserMessage?.(data.content);
              }
              break;

            case 'session_created':
              if (data.content) {
                callbacksRef.current.onSessionCreated?.(data.content);
              }
              break;

            case 'system_notification':
              if (data.notification) {
                callbacksRef.current.onSystemNotification?.(data.notification);
              }
              break;

            case 'streaming_started':
              // Trigger streaming state immediately (before any content arrives)
              // This shows "AI is thinking" indicator right away
              callbacksRef.current.onStreamingStarted?.(data.content || sessionId);
              break;
          }
        } catch (error) {
          callbacksRef.current.onError?.(
            error instanceof Error ? error : new Error('Failed to parse message')
          );
        }
      },

      onError: (error: Error) => {
        console.error('[WebSocketContext] WebSocket error:', error);
        setConnectionState(ConnectionState.ERROR);
        callbacksRef.current.onError?.(error);
      },

      onClose: () => {
        console.log('[WebSocketContext] WebSocket closed');
        setConnectionState(ConnectionState.DISCONNECTED);
        callbacksRef.current.onClose?.();
      },
    };
  }, []);

  /**
   * Update callbacks without reconnecting
   * This is the KEY for navigation - when component remounts,
   * it just updates callbacks, doesn't reconnect
   */
  const updateCallbacks = useCallback((callbacks: WebSocketCallbacks) => {
    callbacksRef.current = callbacks;
    console.log('[WebSocketContext] Callbacks updated');
  }, []);

  /**
   * Connect to a session
   * If already connected to same session, just update callbacks (no reconnect!)
   * If connected to different session, disconnect first then connect
   */
  const connect = useCallback(async (sessionId: string, callbacks: WebSocketCallbacks): Promise<void> => {
    const manager = getManager();

    // Store callbacks first
    callbacksRef.current = callbacks;

    // If already connected to this session, update callbacks and notify
    if (manager.getSessionId() === sessionId && manager.isConnected()) {
      console.log('[WS] 🔄 Reusing connection, updating callbacks for session:', sessionId);

      // KEY FIX: Update the WebSocketManager's callbacks with fresh adapters!
      // This ensures the WebSocketManager routes messages to the new component's callbacks
      const adapterCallbacks = createAdapterCallbacks(sessionId);
      manager.updateCallbacks(adapterCallbacks);
      // Notify that we're connected (component may want to fetch messages)
      callbacks.onOpen?.();
      // Also trigger reconnect callback so component fetches any missed messages
      callbacks.onReconnect?.();
      return;
    }

    // If connected to different session, disconnect first
    if (manager.isConnected() && manager.getSessionId() !== sessionId) {
      console.log('[WS] 🔀 Switching session:', manager.getSessionId(), '→', sessionId);
      await manager.disconnect();
    }

    setConnectionState(ConnectionState.CONNECTING);
    adapterSetupRef.current = sessionId;

    try {
      // Create adapter callbacks for this session
      const adapterCallbacks = createAdapterCallbacks(sessionId);
      await manager.connect(sessionId, adapterCallbacks);
    } catch (error) {
      setConnectionState(ConnectionState.ERROR);
      adapterSetupRef.current = null;
      throw error;
    }
  }, [getManager, createAdapterCallbacks]);

  /**
   * Switch to a different session
   * Convenience method that handles disconnect + connect
   */
  const switchSession = useCallback(async (newSessionId: string, callbacks: WebSocketCallbacks): Promise<void> => {
    await connect(newSessionId, callbacks);
  }, [connect]);

  /**
   * Disconnect from current session
   */
  const disconnect = useCallback(async (): Promise<void> => {
    const manager = getManager();
    await manager.disconnect();
    setCurrentSessionId(null);
    setConnectionState(ConnectionState.DISCONNECTED);
    adapterSetupRef.current = null;
  }, [getManager]);

  /**
   * Send message through WebSocket
   */
  const sendMessage = useCallback(async (content: string): Promise<void> => {
    const manager = getManager();
    await manager.sendMessage(content);
  }, [getManager]);

  /**
   * Stop AI execution
   */
  const stopExecution = useCallback((): void => {
    const manager = getManager();
    manager.sendStopExecution();
  }, [getManager]);

  const value: WebSocketContextValue = {
    connect,
    disconnect,
    switchSession,
    sendMessage,
    stopExecution,
    connectionState,
    currentSessionId,
    isConnected: connectionState === ConnectionState.CONNECTED,
    updateCallbacks,
  };

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
};

/**
 * Hook to access WebSocket context
 * @throws Error if used outside WebSocketProvider
 */
export const useWebSocket = (): WebSocketContextValue => {
  const context = useContext(WebSocketContext);
  if (context === undefined) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
};

export { ConnectionState };
