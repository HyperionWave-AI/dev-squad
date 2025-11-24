/**
 * CodeChatPage
 *
 * Main chat interface with session management and WebSocket streaming.
 * Features:
 * - Two-column layout: SessionList + Chat area
 * - WebSocket real-time streaming
 * - Tool calls display with Radix Accordion
 * - Session management (create, rename, delete)
 * - Subchat support with intelligent interrupt categorization (STOP, MODIFY, CLARIFY, STATUS, CONTINUE)
 */

import React, { useState, useEffect, useRef, useCallback } from 'react';
import { AlertCircle, BarChart3, X, Shield, ShieldOff, GitBranch, GitMerge } from 'lucide-react';
import { SessionList } from '@/components/organisms/SessionList';
import { ChatMessage } from '@/components/organisms/ChatMessage';
import { ChatInput } from '@/components/organisms/ChatInput';
import { PerformanceMonitor } from '@/components/organisms/PerformanceMonitor';
import { ProgressTracker, type TrackableEvent } from '@/components/organisms/ProgressTracker';
import { MetricsDashboard } from '@/components/organisms/MetricsDashboard';
import { ConversationModeToggle } from '@/components/molecules/ConversationModeToggle';

import { useStreamingPerformance } from '@/hooks/useStreamingPerformance';
import { usePluginRegistry } from '@/hooks/usePluginRegistry';
import {
  createSession,
  getSessions,
  getMessages,
  deleteSession,
  updateSession,
  updateErrorPreventionMode,
  updateComplexityAnalysisMode,
  connectChatStream,
  type ChatMessage as ChatMessageType,
  type ChatStreamConnection,
  ConnectionState,
  type ToolCall,
  type ToolResult,
} from '@/services/chatService';


// Local interface for session display (matches SessionList expectations)
interface SessionItem {
  id: string;
  title: string;
  timestamp: Date | string;
  messageCount: number;
  lastMessage?: string;
  activeSubagentId?: string; // Indicates session is being processed by a user-created subagent
  activeSubagentName?: string; // Indicates session is with a system subagent (go-dev, ui-dev, etc.)
  parentSessionId?: string; // For subchats - links to parent session
  isSubchat?: boolean; // Indicates if this is a subchat
  errorPreventionMode?: boolean; // Error prevention toggle state
  complexityAnalysisMode?: boolean; // Complexity analysis toggle state
}

export const CodeChatPage: React.FC = () => {
  // Session state
  const [sessions, setSessions] = useState<SessionItem[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);

  // Messages state
  const [messages, setMessages] = useState<ChatMessageType[]>([]);

  // Streaming state
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamingSessionId, setStreamingSessionId] = useState<string | null>(null); // Track which session is streaming
  const [streamingContent, setStreamingContent] = useState('');
  const [pendingToolCalls, setPendingToolCalls] = useState<Set<string>>(new Set());
  const [streamingToolCalls, setStreamingToolCalls] = useState<ToolCall[]>([]);
  const [streamingToolResults, setStreamingToolResults] = useState<Map<string, ToolResult>>(
    new Map()
  );

  // Progress tracking state
  const [progressEvents, setProgressEvents] = useState<TrackableEvent[]>([]);

  // Metrics drawer state
  const [metricsDrawerOpen, setMetricsDrawerOpen] = useState(false);

  // Error prevention mode state
  const [errorPreventionMode, setErrorPreventionMode] = useState(false);

  // Complexity analysis mode state
  const [complexityAnalysisMode, setComplexityAnalysisMode] = useState(false);

  // Plugin registry hook (currently unused but available for future use)
  const {
    toggleErrorPrevention: _pluginToggleErrorPrevention,
    toggleComplexityAnalysis: _pluginToggleComplexityAnalysis,
    isErrorPreventionEnabled: _isErrorPreventionEnabled,
    isComplexityAnalysisEnabled: _isComplexityAnalysisEnabled,
  } = usePluginRegistry();

  // Error state
  const [error, setError] = useState<string | null>(null);

  // Performance monitoring
  const performance = useStreamingPerformance();

  // Refs for WebSocket and streaming content
  const wsConnectionRef = useRef<ChatStreamConnection | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // WebSocket connection state and message queue
  const wsConnectionStateRef = useRef<'disconnected' | 'connecting' | 'connected'>('disconnected');
  const messageQueueRef = useRef<string[]>([]);
  
  const streamingContentRef = useRef<string>('');
  const currentMessageToolsRef = useRef<{
    toolCalls: ToolCall[];
    toolResults: Map<string, ToolResult>;
  }>({ toolCalls: [], toolResults: new Map() });

  // Ref to track current active session (prevents stale closure in auto-refresh)
  const activeSessionIdRef = useRef<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const prevMessageCountRef = useRef(0);

  // Keep ref in sync with state
  useEffect(() => {
    activeSessionIdRef.current = activeSessionId;
  }, [activeSessionId]);

  // Auto-scroll to bottom ONLY when NEW messages are added (not on every poll)
  useEffect(() => {
    // Only auto-scroll if NEW messages were added (not just reloaded from polling)
    if (messages.length > prevMessageCountRef.current) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
    prevMessageCountRef.current = messages.length;
  }, [messages, streamingContent]);

  // Load sessions on mount (no polling - use WebSocket session_created events instead)
  useEffect(() => {
    loadSessions();
  }, []);

  // Auto-refresh messages for active session (Fix #2: Messages not appearing without refresh)
  useEffect(() => {
    if (!activeSessionId) return;

    // Poll messages every 3 seconds for active session
    const intervalId = setInterval(() => {
      // Fix Bug #8: Smarter WebSocket health check - only reconnect if truly CLOSED
      if (wsConnectionRef.current) {
        const state = wsConnectionRef.current.getState();
        // Only reconnect if connection is DISCONNECTED, not CONNECTING or DISCONNECTING
        if (state === ConnectionState.DISCONNECTED) {
          console.log('[CodeChatPage] WebSocket closed, reconnecting...');
          // Poll messages when WebSocket is disconnected (not streaming)
          if (!isStreaming && !streamingSessionId) {
            console.log('[CodeChatPage] Polling messages - WebSocket disconnected (state: CLOSED)');
            loadMessages(activeSessionId);
          }
        }
      } else {
        // No connection at all, establish one
        console.log('[CodeChatPage] No WebSocket connection, establishing...');
        connectWebSocket(activeSessionId);

        // Poll messages when WebSocket doesn't exist (not streaming)
        if (!isStreaming && !streamingSessionId) {
          console.log('[CodeChatPage] Polling messages - No WebSocket connection exists');
          loadMessages(activeSessionId);
        }
      }
    }, 3000);

    return () => clearInterval(intervalId);
  }, [activeSessionId, isStreaming, streamingSessionId]);

  // Connect WebSocket when active session changes
  useEffect(() => {
    if (activeSessionId) {
      connectWebSocket(activeSessionId);
    }

    // Cleanup on session change or unmount
    return () => {
      if (wsConnectionRef.current) {
        wsConnectionRef.current.disconnect().catch(err => console.error('[CodeChatPage] Error disconnecting:', err));
        wsConnectionRef.current = null;
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, [activeSessionId]);

  // Load error prevention mode and complexity analysis mode when session changes
  useEffect(() => {
    if (activeSessionId && sessions.length > 0) {
      const currentSession = sessions.find((s) => s.id === activeSessionId);
      if (currentSession) {
        // Access modes from the session data
        const sessionData = (currentSession as any);
        setErrorPreventionMode(sessionData.errorPreventionMode || false);
        setComplexityAnalysisMode(sessionData.complexityAnalysisMode || false);
      }
    }
  }, [activeSessionId, sessions]);

  // Load sessions from API
  const loadSessions = async () => {
    try {
      const fetchedSessions = await getSessions();
      // Map chatService.ChatSession to SessionList's expected format
      const mappedSessions: SessionItem[] = fetchedSessions.map((session) => ({
        id: session.id,
        title: session.title,
        timestamp: session.updatedAt || session.createdAt,
        messageCount: 0, // Will be populated when we load messages
        lastMessage: undefined,
        activeSubagentId: session.activeSubagentId, // User-created subagent
        activeSubagentName: (session as any).activeSubagentName, // System subagent
        parentSessionId: session.parentChatId, // Map parentChatId to parentSessionId
        isSubchat: !!session.parentChatId, // Session is a subchat if it has a parent
        errorPreventionMode: session.errorPreventionMode, // Error prevention mode toggle
        complexityAnalysisMode: session.complexityAnalysisMode, // Complexity analysis mode toggle
      }));
      setSessions(mappedSessions);

      // Auto-select first session if none active
      if (!activeSessionIdRef.current && fetchedSessions.length > 0) {
        setActiveSessionId(fetchedSessions[0].id);
        await loadMessages(fetchedSessions[0].id);
      }
    } catch (err) {
      console.error('[CodeChatPage] Error loading sessions:', err);
      setError(err instanceof Error ? err.message : 'Failed to load sessions');
    }
  };

  // Enhanced deduplication with O(n) complexity and ID reconciliation
  const deduplicateMessages = (messages: ChatMessageType[]): ChatMessageType[] => {
    const messageMap = new Map<string, ChatMessageType>();
    const optimisticToDatabaseMap = new Map<string, string>(); // optimistic ID → database ID

    // First pass: identify optimistic → database ID mappings
    for (const msg of messages) {
      if (msg.role === 'user' && msg.id.startsWith('msg-')) {
        // Find corresponding database version of this optimistic message
        const dbVersion = messages.find(m =>
          m.role === 'user' &&
          !m.id.startsWith('msg-') &&
          m.content === msg.content &&
          m.sessionId === msg.sessionId &&
          Math.abs(new Date(m.timestamp).getTime() - new Date(msg.timestamp).getTime()) < 5000
        );
        if (dbVersion) {
          optimisticToDatabaseMap.set(msg.id, dbVersion.id);
        }
      }
    }

    // Second pass: deduplicate with ID reconciliation (O(n))
    for (const msg of messages) {
      // If this is an optimistic message that has a database version, skip it
      if (optimisticToDatabaseMap.has(msg.id)) {
        continue; // Skip optimistic version, keep database version
      }

      const effectiveId = msg.id;

      // Always prefer database messages over optimistic ones
      if (!messageMap.has(effectiveId) || !msg.id.startsWith('msg-')) {
        messageMap.set(effectiveId, msg);
      }
    }

    // Return sorted by timestamp
    return Array.from(messageMap.values()).sort((a, b) =>
      new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    );
  };

  // Load messages for a session
  const loadMessages = async (sessionId: string) => {
    try {
      const fetchedMessages = await getMessages(sessionId);

      // Bug #7 Fix: Convert toolResults from plain objects to Maps for consistency
      const normalizedMessages = fetchedMessages.map(msg => {
        if (msg.toolResults && !(msg.toolResults instanceof Map)) {
          // Convert plain object or array to Map
          const toolResultsMap = new Map<string, ToolResult>();
          const results: any = msg.toolResults;

          if (Array.isArray(results)) {
            // If it's an array, convert each item
            results.forEach((result: any) => {
              if (result.id) {
                toolResultsMap.set(result.id, result as ToolResult);
              }
            });
          } else if (typeof results === 'object') {
            // If it's a plain object, convert entries to Map
            Object.entries(results).forEach(([id, result]) => {
              toolResultsMap.set(id, result as ToolResult);
            });
          }
          return { ...msg, toolResults: toolResultsMap };
        }
        return msg;
      });

      setMessages((prev) => {
        // Merge fetched messages with existing, deduplicate
        const merged = [...prev, ...normalizedMessages];
        return deduplicateMessages(merged);
      });
    } catch (err) {
      console.error('[CodeChatPage] Error loading messages:', err);
      setError(err instanceof Error ? err.message : 'Failed to load messages');
    }
  };

  // Connect to WebSocket for streaming
  const connectWebSocket = useCallback((sessionId: string) => {
    // Fix Bug #8: Guard against redundant connections
    if (wsConnectionRef.current &&
        wsConnectionRef.current.isConnected() &&
        activeSessionIdRef.current === sessionId) {
      console.log('[CodeChatPage] Already connected to session, skipping reconnect');
      return;
    }
    // Bug #5 Fix: Proper connection cleanup before creating new connection
    // 1. Clear any pending reconnection attempts
    if (reconnectTimeoutRef.current) {
      reconnectTimeoutRef.current = null;
    }

    // 2. Disconnect existing connection
    if (wsConnectionRef.current) {
      wsConnectionRef.current.disconnect().catch(err => console.error('[CodeChatPage] Error disconnecting:', err));
      wsConnectionRef.current = null;
    }

    // 3. Reset streaming state for the session being disconnected
    if (streamingSessionId === sessionId) {
      setStreamingSessionId(null);
    }
    
    // Reset connection state
    wsConnectionStateRef.current = 'connecting';

    // 4. Reset all streaming refs and state
    streamingContentRef.current = '';
    currentMessageToolsRef.current = { toolCalls: [], toolResults: new Map() };
    setPendingToolCalls(new Set());
    setStreamingToolCalls([]);
    setStreamingToolResults(new Map());

    // Connect to WebSocket
    const connection = connectChatStream(sessionId, {
      onMessage: (content: string, done: boolean) => {
        // Fix Bug #8: Use ref to avoid stale closure issues with React state
        if (sessionId !== activeSessionIdRef.current) {
          console.log('[CodeChatPage] Message for different session, ignoring');
          return;
        }

        // RACE CONDITION FIX: Verify streamingContent belongs to current session
        // Store session ID with content to prevent stale closure bugs
        if (!streamingContentRef.current.startsWith(`[${sessionId}]:`)) {
          // First message for this session - prefix with session ID
          if (!done) {
            streamingContentRef.current = `[${sessionId}]:`;
          }
        }

        if (done) {
          // Stream complete - save final AI message if there's any remaining content
          // Extract content after session ID prefix
          const prefixedContent = streamingContentRef.current;
          const sessionPrefix = `[${sessionId}]:`;

          // Verify content belongs to this session before using it
          if (!prefixedContent.startsWith(sessionPrefix)) {
            console.warn('[CodeChatPage] RACE CONDITION DETECTED: Content belongs to different session, discarding');
            streamingContentRef.current = '';
            setStreamingContent('');
            setIsStreaming(false);
            setStreamingSessionId(null);
            return;
          }

          const finalContent = prefixedContent.substring(sessionPrefix.length);
          const tools = currentMessageToolsRef.current;

          if (finalContent || tools.toolCalls.length > 0) {
            const newMessage: ChatMessageType = {
              id: `msg-${Date.now()}`,
              sessionId,
              role: 'assistant',
              content: finalContent,
              timestamp: new Date().toISOString(),
              toolCalls: tools.toolCalls.length > 0 ? tools.toolCalls : undefined,
              toolResults: tools.toolResults.size > 0 ? tools.toolResults : undefined,
            };
            setMessages((prev) => {
              const updated = [...prev, newMessage];
              console.log('[CodeChatPage] ✅ Messages array updated. New length:', updated.length);
              return updated;
            });
          }

          // Refresh sessions in case AI created subchats
          console.log('[CodeChatPage] Refreshing sessions after AI response completion');
          loadSessions();

          // Clear streaming state (Bug #2 fix: clear streamingSessionId)
          streamingContentRef.current = '';
          currentMessageToolsRef.current = { toolCalls: [], toolResults: new Map() };
          setStreamingContent('');
          setIsStreaming(false);
          setStreamingSessionId(null); // Clear streaming session tracker
          setPendingToolCalls(new Set());
          setStreamingToolCalls([]);
          setStreamingToolResults(new Map());

          // Stop performance monitoring
          performance.stopMonitoring();
        } else {
          // Accumulate streaming content (with session ID prefix for race condition safety)
          streamingContentRef.current += content;
          // Display content without session ID prefix
          const sessionPrefix = `[${sessionId}]:`;
          const displayContent = streamingContentRef.current.substring(sessionPrefix.length);
          setStreamingContent(displayContent);
          setIsStreaming(true);

          // Debug: Log streaming state updates
          console.log(`[CodeChatPage] 📝 Content: ${streamingContentRef.current.length} bytes`);

          // Mark this session as streaming (Bug #2 fix)
          if (!streamingSessionId) {
            setStreamingSessionId(sessionId);
          }

          // Record chunk for performance monitoring
          performance.recordChunk(content);
        }
      },
      onToolCall: (tool: string, args: Record<string, any>, id: string) => {
        console.log(`[CodeChatPage] 🔧 Tool: ${tool}`);

        // If we have accumulated content before the tool call, save it as a separate message
        // Extract content without session ID prefix
        const sessionPrefix = `[${sessionId}]:`;
        const contentWithPrefix = streamingContentRef.current;
        const content = contentWithPrefix.startsWith(sessionPrefix)
          ? contentWithPrefix.substring(sessionPrefix.length)
          : contentWithPrefix;

        if (content.trim()) {
          const messageBeforeToolCall: ChatMessageType = {
            id: `msg-${Date.now()}`,
            sessionId,
            role: 'assistant',
            content: content,
            timestamp: new Date().toISOString(),
          };
          setMessages((prev) => [...prev, messageBeforeToolCall]);
          console.log('[CodeChatPage] Saved message before tool call');

          // Clear streaming content for next message (keep session prefix)
          streamingContentRef.current = sessionPrefix;
          setStreamingContent('');
        }

        const toolCall: ToolCall = {
          id,
          tool,
          args,
          timestamp: new Date(),
        };
        currentMessageToolsRef.current.toolCalls.push(toolCall);
        setPendingToolCalls((prev) => new Set(prev).add(id));
        setStreamingToolCalls((prev) => [...prev, toolCall]);
      },
      onToolResult: (
        id: string,
        tool: string,
        result: any,
        error: string | null,
        durationMs: number
      ) => {
        const toolResult: ToolResult = {
          id,
          tool,
          result,
          error,
          durationMs,
        };

        // Debug logging to diagnose tool result flow
        const resultSize = JSON.stringify(result).length;
        const hasErrorFlag = !!error;
        console.log(`[CodeChatPage] ${hasErrorFlag ? '❌' : '✅'} ${tool} (${resultSize > 1024 ? Math.round(resultSize/1024) + 'KB' : resultSize + 'B'})`);

        currentMessageToolsRef.current.toolResults.set(id, toolResult);

        setPendingToolCalls((prev) => {
          const updated = new Set(prev);
          updated.delete(id);
          return updated;
        });

        setStreamingToolResults((prev) => new Map(prev).set(id, toolResult));
      },
      onMessageSaved: (databaseId: string) => {
        // Check if this is a WebSocket disconnection notification (new backend feature)
        if (databaseId.includes('AI response saved')) {
          console.log('[CodeChatPage] WebSocket disconnected but message saved - refetching messages');
          // Refetch messages to get the complete AI response that was saved
          if (activeSessionId) {
            loadMessages(activeSessionId);
          }
          // Stop showing streaming state
          setIsStreaming(false);
          streamingContentRef.current = '';
          setStreamingContent('');
          return;
        }

        // Bug #3 Fix: Properly replace optimistic message ID with database ID
        console.log('[CodeChatPage] Message saved with database ID:', databaseId);
        setMessages((prev) => {
          // Find the most recent user message with optimistic ID
          const optimisticIndex = prev.findIndex((_msg, idx) => {
            // Search from end to find most recent
            const reverseIdx = prev.length - 1 - idx;
            return prev[reverseIdx].role === 'user' && prev[reverseIdx].id.startsWith('msg-');
          });

          if (optimisticIndex === -1) {
            console.warn('[CodeChatPage] No optimistic message found to update');
            return prev;
          }

          const actualIndex = prev.length - 1 - optimisticIndex;
          console.log('[CodeChatPage] Updating message ID:', prev[actualIndex].id, '→', databaseId);

          // Create new array with updated message (immutable update)
          const updated = prev.map((msg, idx) =>
            idx === actualIndex ? { ...msg, id: databaseId } : msg
          );

          return updated;
        });
      },
      onError: (err: Error) => {
        setError(`Connection error: ${err.message}`);
        setIsStreaming(false);
        streamingContentRef.current = '';
        setStreamingContent('');

        // Attempt reconnect after 3 seconds
        reconnectTimeoutRef.current = setTimeout(() => {
          console.log('[CodeChatPage] Attempting to reconnect...');
          connectWebSocket(sessionId);
        }, 3000);
      },
      onOpen: () => {
        console.log('[CodeChatPage] WebSocket connected');
        wsConnectionStateRef.current = 'connected';
        setError(null);
        
        // Process any queued messages
        if (messageQueueRef.current.length > 0) {
          console.log(`[CodeChatPage] Processing ${messageQueueRef.current.length} queued messages`);
          const queuedMessages = [...messageQueueRef.current];
          messageQueueRef.current = [];
          queuedMessages.forEach(message => {
            sendMessageInternal(message);
          });
        }
      },
      onClose: () => {
        console.log('[CodeChatPage] WebSocket disconnected');
        wsConnectionStateRef.current = 'disconnected';
      },
      onSessionCreated: (subchatId: string) => {
        console.log('[CodeChatPage] New subchat created, refreshing sessions list', subchatId);
        // Refresh sessions list to show the new subchat
        loadSessions();
      },
    });

    wsConnectionRef.current = connection;
  }, [activeSessionId, streamingSessionId, performance]);

  // Session management handlers
  const handleNewChat = async () => {
    const newSession = await createSession('New Chat');
    const sessionItem: SessionItem = {
      id: newSession.id,
      title: newSession.title,
      timestamp: newSession.updatedAt || newSession.createdAt,
      messageCount: 0,
      lastMessage: undefined,
      activeSubagentId: newSession.activeSubagentId,
      activeSubagentName: (newSession as any).activeSubagentName,
      parentSessionId: newSession.parentChatId,
      isSubchat: !!newSession.parentChatId,
    };
    setSessions((prev) => [sessionItem, ...prev]);
    setActiveSessionId(newSession.id);
    setMessages([]);
    setStreamingContent('');
  };

  const handleSessionSelect = async (sessionId: string) => {
    if (sessionId !== activeSessionId) {
      setActiveSessionId(sessionId);
      // Clear old messages and streaming state immediately before loading new session
      setMessages([]);
      setStreamingContent('');
      setIsStreaming(false);
      setStreamingToolCalls([]);
      setStreamingToolResults(new Map());
      setPendingToolCalls(new Set());
      await loadMessages(sessionId);
    }
  };

  const handleDeleteSession = async (sessionId: string) => {
    await deleteSession(sessionId);
    setSessions((prev) => prev.filter((s) => s.id !== sessionId));

    // If deleted active session, select another
    if (sessionId === activeSessionId) {
      const remainingSessions = sessions.filter((s) => s.id !== sessionId);
      if (remainingSessions.length > 0) {
        setActiveSessionId(remainingSessions[0].id);
        await loadMessages(remainingSessions[0].id);
      } else {
        setActiveSessionId(null);
        setMessages([]);
      }
    }
  };

  const handleDeleteAllSessions = async () => {
    await Promise.all(sessions.map((session) => deleteSession(session.id)));
    setSessions([]);
    setActiveSessionId(null);
    setMessages([]);
  };

  const handleRenameSession = async (sessionId: string, newTitle: string) => {
    const updatedSession = await updateSession(sessionId, newTitle);
    const sessionItem: SessionItem = {
      id: updatedSession.id,
      title: updatedSession.title,
      timestamp: updatedSession.updatedAt || updatedSession.createdAt,
      messageCount: 0,
      lastMessage: undefined,
      activeSubagentId: updatedSession.activeSubagentId,
      activeSubagentName: (updatedSession as any).activeSubagentName,
      parentSessionId: updatedSession.parentChatId,
      isSubchat: !!updatedSession.parentChatId,
    };
    setSessions((prev) => prev.map((s) => (s.id === sessionId ? sessionItem : s)));
  };

  // Internal message sending function (used by queue processing)
  const sendMessageInternal = useCallback((text: string) => {
    if (!activeSessionId || !wsConnectionRef.current) return;

    // Optimistically add user message
    const userMessage: ChatMessageType = {
      id: `msg-${Date.now()}`,
      sessionId: activeSessionId,
      role: 'user',
      content: text,
      timestamp: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, userMessage]);

    // Reset streaming state
    setIsStreaming(true);
    setStreamingContent('');
    streamingContentRef.current = '';

    // Start performance monitoring
    performance.resetMetrics();
    performance.startMonitoring();

    // Send message via WebSocket
    wsConnectionRef.current.sendMessage(text).catch(err => {
      setError(err instanceof Error ? err.message : 'Failed to send message');
      setIsStreaming(false);
      setStreamingContent('');
      streamingContentRef.current = '';
      performance.stopMonitoring();
    });
  }, [activeSessionId, performance]);

  // Message sending handler
  const handleSendMessage = (text: string) => {
    if (!activeSessionId || isStreaming) return;

    // Check connection state and handle accordingly
    const connectionState = wsConnectionStateRef.current;

    if (connectionState === 'connected' && wsConnectionRef.current?.isConnected()) {
      // Connection is ready, send immediately
      sendMessageInternal(text);
    } else if (connectionState === 'connecting') {
      // Connection is in progress, queue the message
      console.log('[CodeChatPage] Connection in progress, queueing message');
    } else {
      // Connection is disconnected, start connecting and queue the message
      console.log('[CodeChatPage] WebSocket disconnected, reconnecting and queueing message');
      messageQueueRef.current.push(text);
      connectWebSocket(activeSessionId);
    }
  };

  // Toggle error prevention mode
  const toggleErrorPrevention = async () => {
    if (!activeSessionId) return;

    const newMode = !errorPreventionMode;

    try {
      const result = await updateErrorPreventionMode(activeSessionId, newMode);
      setErrorPreventionMode(result.errorPreventionMode);

      // Update the session in the sessions list
      setSessions(prevSessions =>
        prevSessions.map(session =>
          session.id === activeSessionId
            ? { ...session, errorPreventionMode: result.errorPreventionMode }
            : session
        )
      );

      console.log(`[CodeChatPage] Error Prevention Mode: ${result.errorPreventionMode ? 'ON' : 'OFF'}`);
    } catch (error) {
      console.error('[CodeChatPage] Failed to toggle error prevention mode:', error);
      setError('Failed to toggle error prevention mode');
    }
  };

  // Toggle complexity analysis mode
  const toggleComplexityAnalysis = async () => {
    if (!activeSessionId) return;

    const newMode = !complexityAnalysisMode;

    try {
      const result = await updateComplexityAnalysisMode(activeSessionId, newMode);
      setComplexityAnalysisMode(result.complexityAnalysisMode);

      // Update the session in the sessions list
      setSessions(prevSessions =>
        prevSessions.map(session =>
          session.id === activeSessionId
            ? { ...session, complexityAnalysisMode: result.complexityAnalysisMode }
            : session
        )
      );

      console.log(`[CodeChatPage] Complexity Analysis Mode: ${result.complexityAnalysisMode ? 'ON' : 'OFF'}`);
    } catch (error) {
      console.error('[CodeChatPage] Failed to toggle complexity analysis mode:', error);
      setError('Failed to toggle complexity analysis mode');
    }
  };

  // Note: Subchats are now interruptible (supports intelligent interrupt categorization)
  // Backend handles STOP, MODIFY, CLARIFY, STATUS, CONTINUE categories

  return (
    <div className="flex h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      {/* Left Sidebar - Session List */}
      <div className="w-80 shrink-0 backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border-r border-white/30 dark:border-gray-700/30">
        <SessionList
          sessions={sessions}
          currentSessionId={activeSessionId ?? undefined}
          onSessionSelect={handleSessionSelect}
          onNewChat={handleNewChat}
          onDeleteSession={handleDeleteSession}
          onDeleteAllSessions={handleDeleteAllSessions}
          onRenameSession={handleRenameSession}
          onRefreshSessions={loadSessions}
        />
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Chat Header with Mode Toggle */}
        {activeSessionId && (
          <div className="shrink-0 px-6 py-3 border-b border-gray-200 dark:border-gray-700 bg-white/50 dark:bg-gray-800/50 backdrop-blur-sm">
            <div className="flex items-center justify-between">
              <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                {sessions.find((s) => s.id === activeSessionId)?.title || 'Chat'}
              </h1>
              <div className="flex items-center gap-3">
                <ConversationModeToggle showLabel={true} />

                {/* Error Prevention Mode Toggle */}
                <button
                  onClick={toggleErrorPrevention}
                  className={`p-2 rounded-lg transition-all ${
                    errorPreventionMode
                      ? 'bg-green-100 dark:bg-green-900/30 hover:bg-green-200 dark:hover:bg-green-900/50 border-2 border-green-500 dark:border-green-400'
                      : 'hover:bg-gray-100 dark:hover:bg-gray-700 border-2 border-gray-300 dark:border-gray-600'
                  }`}
                  title={
                    errorPreventionMode
                      ? 'Error Prevention: ON - AI validates code and fixes errors automatically'
                      : 'Error Prevention: OFF - No validation (useful for debugging)'
                  }
                >
                  {errorPreventionMode ? (
                    <Shield className="w-5 h-5 text-green-600 dark:text-green-400" />
                  ) : (
                    <ShieldOff className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                  )}
                </button>

                {/* Complexity Analysis Mode Toggle */}
                <button
                  onClick={toggleComplexityAnalysis}
                  className={`p-2 rounded-lg transition-all ${
                    complexityAnalysisMode
                      ? 'bg-purple-100 dark:bg-purple-900/30 hover:bg-purple-200 dark:hover:bg-purple-900/50 border-2 border-purple-500 dark:border-purple-400'
                      : 'hover:bg-gray-100 dark:hover:bg-gray-700 border-2 border-gray-300 dark:border-gray-600'
                  }`}
                  title={
                    complexityAnalysisMode
                      ? 'Complexity Analysis: ON - AI analyzes task complexity and suggests splitting large tasks'
                      : 'Complexity Analysis: OFF - No complexity analysis (useful for debugging)'
                  }
                >
                  {complexityAnalysisMode ? (
                    <GitBranch className="w-5 h-5 text-purple-600 dark:text-purple-400" />
                  ) : (
                    <GitMerge className="w-5 h-5 text-gray-600 dark:text-gray-400" />
                  )}
                </button>

                <button
                  onClick={() => setMetricsDrawerOpen(!metricsDrawerOpen)}
                  className="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                  title="Toggle metrics dashboard"
                >
                  <BarChart3 className="w-5 h-5 text-gray-700 dark:text-gray-300" />
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Chat Messages Area */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">
          {!activeSessionId ? (
            // Empty State
            <div className="flex items-center justify-center h-full text-center">
              <div className="max-w-md">
                <div className="w-16 h-16 mx-auto mb-4 bg-primary-100 dark:bg-primary-900/30 rounded-full flex items-center justify-center">
                  <svg
                    className="w-8 h-8 text-primary-600 dark:text-primary-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z"
                    />
                  </svg>
                </div>
                <h2 className="text-2xl font-semibold text-gray-900 dark:text-gray-100 mb-2">
                  Welcome to Code Chat
                </h2>
                <p className="text-gray-600 dark:text-gray-400 mb-6">
                  Create a new chat session to start a conversation with the AI assistant.
                </p>
              </div>
            </div>
          ) : (
            // Messages
            <>
              {messages.map((message) => (
                <ChatMessage key={message.id} message={message} />
              ))}

              {/* AI Thinking Indicator (Fix #1: Show before content arrives) */}
              {/* Also show when session is being processed by a subagent */}
              {(() => {
                const activeSession = sessions.find((s) => s.id === activeSessionId);
                const isProcessing = activeSession?.activeSubagentId != null;
                const showIndicator = (isStreaming && !streamingContent) || isProcessing;

                if (!showIndicator) return null;

                return (
                  <div className="flex items-start gap-3 p-4 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm">
                    <div className="flex-shrink-0 mt-1">
                      <div className="relative flex items-center justify-center w-6 h-6">
                        <div className="w-3 h-3 bg-blue-500 rounded-full animate-pulse z-10" />
                        <div className="absolute inset-0 w-3 h-3 bg-blue-500 rounded-full animate-ping opacity-75" />
                      </div>
                    </div>
                    <div className="flex-1">
                      <div className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-1">
                        Assistant
                      </div>
                      <div className="text-sm text-gray-600 dark:text-gray-400 italic">
                        {isProcessing ? 'Agent is processing task...' : 'AI is thinking...'}
                      </div>
                    </div>
                  </div>
                );
              })()}

              {/* Streaming Assistant Message */}
              {isStreaming && (streamingContent || streamingToolCalls.length > 0) && (
                <ChatMessage
                  message={{
                    id: 'streaming',
                    sessionId: activeSessionId,
                    role: 'assistant',
                    content: '',
                    timestamp: new Date().toISOString(),
                  }}
                  isStreaming={true}
                  streamingContent={streamingContent}
                  pendingToolCalls={pendingToolCalls}
                  streamingToolCalls={streamingToolCalls}
                  streamingToolResults={streamingToolResults}
                />
              )}

              {/* Auto-scroll anchor */}
              <div ref={messagesEndRef} />
            </>
          )}
        </div>

        {/* Error Banner */}
        {error && (
          <div className="border-t border-gray-200 dark:border-gray-700 bg-red-50 dark:bg-red-900/20 px-6 py-3">
            <div className="flex items-center gap-3">
              <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400 shrink-0" />
              <p className="text-sm text-red-800 dark:text-red-300 flex-1">{error}</p>
              <button
                onClick={() => setError(null)}
                className="text-sm text-red-600 dark:text-red-400 hover:underline"
              >
                Dismiss
              </button>
            </div>
          </div>
        )}

        {/* Chat Input */}
        <ChatInput
          onSendMessage={handleSendMessage}
          disabled={!activeSessionId || isStreaming}
          placeholder={
            !activeSessionId
              ? 'Create a new chat to get started'
              : 'Type your message...'
          }
        />
      </div>

      {/* Performance Monitor - Fixed bottom-right */}
      <PerformanceMonitor
        stats={performance.stats}
        fpsHistory={performance.fpsHistory}
        isPerformanceGood={performance.isPerformanceGood}
        isStreaming={isStreaming}
      />

      {/* Progress Tracker - Fixed bottom-right (above performance monitor) */}
      <ProgressTracker
        events={progressEvents}
        onClose={() => setProgressEvents([])}
      />
      {/* Metrics Drawer - Slide in from right */}
      {metricsDrawerOpen && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 bg-black/20 dark:bg-black/40 z-40 backdrop-blur-sm"
            onClick={() => setMetricsDrawerOpen(false)}
          />

          {/* Drawer */}
          <div className="fixed right-0 top-0 h-full w-full max-w-4xl bg-white dark:bg-gray-900 shadow-2xl z-50 overflow-y-auto animate-in slide-in-from-right duration-300">
            {/* Drawer Header */}
            <div className="sticky top-0 z-10 flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700 bg-white/95 dark:bg-gray-900/95 backdrop-blur-sm">
              <h2 className="text-2xl font-bold text-gray-900 dark:text-white">
                System Metrics
              </h2>
              <button
                onClick={() => setMetricsDrawerOpen(false)}
                className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"
                aria-label="Close metrics drawer"
              >
                <X className="w-5 h-5 text-gray-700 dark:text-gray-300" />
              </button>
            </div>

            {/* Drawer Content */}
            <div className="p-6">
              <MetricsDashboard />
            </div>
          </div>
        </>
      )}
    </div>
  );
};

export default CodeChatPage;
