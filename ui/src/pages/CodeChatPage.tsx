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
import { ConnectionStatusIndicator, type ConnectionStatus } from '@/components/molecules/ConnectionStatusIndicator';
import { ErrorBoundary } from '@/components/molecules/ErrorBoundary';

import { ContextIndicator, ContextStatusModal } from '@/components/molecules/ContextIndicator';
import { ArchiveDialog } from '@/components/organisms/ArchiveDialog';
import { useStreamingPerformance } from '@/hooks/useStreamingPerformance';
import { usePluginRegistry } from '@/hooks/usePluginRegistry';
import { useContextStatus } from '@/hooks/useContextStatus';
import { useWebSocket, ConnectionState } from '@/contexts/WebSocketContext';
import {
  createSession,
  getSessions,
  getMessages,
  deleteSession,
  updateSession,
  updateErrorPreventionMode,
  updateComplexityAnalysisMode,
  archiveMessages,
  type ChatMessage as ChatMessageType,
  type ToolCall,
  type ToolResult,
  type SystemNotification,
} from '@/services/chatService';
import { showSystemNotification } from '@/services/notificationService';


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

  // Context management state
  const [archiveDialogOpen, setArchiveDialogOpen] = useState(false);
  const [contextModalOpen, setContextModalOpen] = useState(false);
  // Plugin registry hook (currently unused but available for future use)
  const {
    toggleErrorPrevention: _pluginToggleErrorPrevention,
    toggleComplexityAnalysis: _pluginToggleComplexityAnalysis,
    isErrorPreventionEnabled: _isErrorPreventionEnabled,
    isComplexityAnalysisEnabled: _isComplexityAnalysisEnabled,
  } = usePluginRegistry();

  // Error state
  const [error, setError] = useState<string | null>(null);

  // Connection status state
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('disconnected');

  // Performance monitoring
  const performance = useStreamingPerformance();
  const { resetMetrics, startMonitoring, stopMonitoring, recordChunk } = performance;

  // Context status polling
  const { contextMetadata, contextError, startPolling, stopPolling } = useContextStatus({
    sessionId: activeSessionId || undefined,
    pollInterval: 2000,
  });

  // Start polling when session is active
  useEffect(() => {
    if (activeSessionId) {
      startPolling();
    } else {
      stopPolling();
    }
  }, [activeSessionId, startPolling, stopPolling]);

  // App-level WebSocket context (singleton - persists across navigation!)
  const {
    connect: wsConnect,
    sendMessage: wsSendMessage,
    stopExecution: wsStopExecution,
    connectionState: wsConnectionState,
    isConnected: wsIsConnected,
    currentSessionId: wsCurrentSessionId,
  } = useWebSocket();

  // Refs for streaming content
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


  // Sync WebSocket connection state to local connectionStatus
  useEffect(() => {
    const stateMap: Record<string, ConnectionStatus> = {
      [ConnectionState.CONNECTED]: 'connected',
      [ConnectionState.CONNECTING]: 'connecting',
      [ConnectionState.RECONNECTING]: 'connecting',
      [ConnectionState.DISCONNECTED]: 'disconnected',
      [ConnectionState.DISCONNECTING]: 'disconnected',
      [ConnectionState.ERROR]: 'error',
    };
    setConnectionStatus(stateMap[wsConnectionState] || 'disconnected');
  }, [wsConnectionState]);

  // DON'T auto-connect WebSocket when switching chats!
  // WebSocket should only connect when:
  // 1. Sending a message (connectWebSocket called in handleSendMessage)
  // 2. There's an active stream for this session
  // This prevents disconnecting from a streaming session when just viewing another chat's history

  // Only connect WebSocket if this session is currently streaming or we don't have any connection
  useEffect(() => {
    if (activeSessionId) {
      // Check if we should connect:
      // - If not connected to anything, connect to view this session
      // - If connected to THIS session already, just update callbacks
      // - If connected to ANOTHER session that has active AI work, DON'T disconnect it!
      const currentWsSession = wsCurrentSessionId;

      // AI is "active" if streaming OR has pending tool calls OR has a streaming session marker
      const isAiActive = isStreaming || pendingToolCalls.size > 0 || streamingSessionId !== null;

      if (!currentWsSession) {
        // No connection at all - connect to active session
        console.log('[WS] 🔌 No connection, connecting to:', activeSessionId);
        connectWebSocket(activeSessionId);
      } else if (currentWsSession === activeSessionId) {
        // Already connected to this session - just update callbacks
        console.log('[WS] 🔄 Already connected to this session, updating callbacks');
        connectWebSocket(activeSessionId);
      } else if (!isAiActive) {
        // Connected to different session but AI is NOT active - safe to switch
        console.log('[WS] 🔀 AI not active, switching to:', activeSessionId);
        connectWebSocket(activeSessionId);
      } else {
        // Connected to different session AND AI is active - DON'T disconnect!
        console.log('[WS] ⏳ Keeping connection to active AI session:', currentWsSession, '(viewing:', activeSessionId, ')');
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeSessionId, wsCurrentSessionId, isStreaming, pendingToolCalls.size, streamingSessionId]);

  // Cleanup streaming state on component unmount (but NOT WebSocket!)
  useEffect(() => {
    return () => {
      // Clear streaming state on unmount
      setMessages([]);
      setStreamingContent('');
      setIsStreaming(false);
      setStreamingSessionId(null);
      setPendingToolCalls(new Set());
      setStreamingToolCalls([]);
      setStreamingToolResults(new Map());

      // Clear refs
      streamingContentRef.current = '';
      currentMessageToolsRef.current = { toolCalls: [], toolResults: new Map() };

      console.log('[CodeChatPage] Component unmounted, streaming state cleared (WebSocket persists at app level)');
    };
  }, []);

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

  // Efficient deduplication with O(n) complexity using Map-based lookup
  const deduplicateMessages = (messages: ChatMessageType[]): ChatMessageType[] => {
    const messageMap = new Map<string, ChatMessageType>();
    
    // Single pass: deduplicate by ID, preferring database IDs over optimistic ones
    for (const msg of messages) {
      const existingMsg = messageMap.get(msg.id);
      
      // Skip if we already have this exact ID
      if (existingMsg) {
        continue;
      }
      
      // For optimistic messages (msg-*), check if we have a database version
      if (msg.id.startsWith('msg-')) {
        // Look for a database version with matching content and timestamp
        let hasDatabaseVersion = false;
        for (const [, existingMsg] of messageMap) {
          if (
            existingMsg.role === msg.role &&
            existingMsg.content === msg.content &&
            existingMsg.sessionId === msg.sessionId &&
            !existingMsg.id.startsWith('msg-') &&
            Math.abs(new Date(existingMsg.timestamp).getTime() - new Date(msg.timestamp).getTime()) < 5000
          ) {
            hasDatabaseVersion = true;
            break;
          }
        }
        // Skip optimistic message if database version exists
        if (hasDatabaseVersion) {
          continue;
        }
      }
      
      messageMap.set(msg.id, msg);
    }

    // Return sorted by timestamp
    return Array.from(messageMap.values()).sort((a, b) =>
      new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    );
  };

  // Load messages for a session
  const loadMessages = async (sessionId: string) => {
    console.log('[WS] 📂 Loading messages from DB for session:', sessionId);
    try {
      const fetchedMessages = await getMessages(sessionId);
      console.log('[WS] 📂 Loaded', fetchedMessages.length, 'messages from DB');

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

  // Connect to WebSocket for streaming using app-level context
  // The WebSocket persists across navigation - only reconnects when session changes
  const connectWebSocket = useCallback((sessionId: string, forceReset: boolean = false) => {
    // Only reset streaming state if:
    // 1. forceReset is true (explicit new message being sent)
    // 2. OR this is NOT the session that's currently streaming (switching to idle session)
    const isCurrentlyStreamingSession = streamingSessionId === sessionId;
    const shouldReset = forceReset || (!isCurrentlyStreamingSession && !streamingSessionId);

    if (isCurrentlyStreamingSession && !forceReset) {
      // This IS the streaming session - preserve its state
      console.log('[WS] Preserving streaming state for active session:', sessionId);
    } else if (shouldReset) {
      // Reset streaming state for new session or when sending new message
      console.log('[WS] Resetting streaming state for session:', sessionId);
      streamingContentRef.current = '';
      currentMessageToolsRef.current = { toolCalls: [], toolResults: new Map() };
      setPendingToolCalls(new Set());
      setStreamingToolCalls([]);
      setStreamingToolResults(new Map());
    }

    // Connect using app-level WebSocket context
    // If already connected to same session, this just updates callbacks (no reconnect!)
    wsConnect(sessionId, {
      onMessage: (content: string, done: boolean) => {
        // IMPORTANT: Don't drop messages just because user is viewing a different session!
        // Messages belong to the WebSocket session, not the viewed session
        // Store them and show when user returns to this session
        if (sessionId !== activeSessionIdRef.current) {
          console.log('[WS] 📬 Receiving message for background session:', sessionId, '(viewing:', activeSessionIdRef.current, ')');
          // Continue processing - messages will be stored and shown when user returns
        }

        if (done) {
          // Stream complete - save final AI message
          const finalContent = streamingContentRef.current;
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

          // Clear streaming state
          streamingContentRef.current = '';
          currentMessageToolsRef.current = { toolCalls: [], toolResults: new Map() };
          setStreamingContent('');
          setIsStreaming(false);
          setStreamingSessionId(null);
          setPendingToolCalls(new Set());
          setStreamingToolCalls([]);
          setStreamingToolResults(new Map());

          // Stop performance monitoring
          stopMonitoring();
        } else {
          // Accumulate streaming content in ref (always - even for background sessions)
          streamingContentRef.current += content;

          // Only update UI state if this is the currently viewed session
          if (sessionId === activeSessionIdRef.current) {
            setStreamingContent(streamingContentRef.current);
            setIsStreaming(true);
          }

          console.log(`[CodeChatPage] 📝 Content: ${streamingContentRef.current.length} bytes (session: ${sessionId}, viewing: ${activeSessionIdRef.current})`);

          // Mark this session as streaming (always track which session is streaming)
          if (!streamingSessionId) {
            setStreamingSessionId(sessionId);
          }

          // Record chunk for performance monitoring
          recordChunk(content);
        }
      },
      onToolCall: (tool: string, args: Record<string, unknown>, id: string) => {
        console.log(`[CodeChatPage] 🔧 Tool: ${tool} (session: ${sessionId}, viewing: ${activeSessionIdRef.current})`);

        const isActiveSession = sessionId === activeSessionIdRef.current;

        // If we have accumulated content before the tool call, save it as a separate message
        const content = streamingContentRef.current;

        if (content.trim() && isActiveSession) {
          const messageBeforeToolCall: ChatMessageType = {
            id: `msg-${Date.now()}`,
            sessionId,
            role: 'assistant',
            content: content,
            timestamp: new Date().toISOString(),
          };
          setMessages((prev) => [...prev, messageBeforeToolCall]);
          console.log('[CodeChatPage] Saved message before tool call');
        }

        // Always clear the ref content after a tool call
        streamingContentRef.current = '';
        if (isActiveSession) {
          setStreamingContent('');
        }

        const toolCall: ToolCall = {
          id,
          tool,
          args: args as Record<string, any>,
          timestamp: new Date(),
        };

        // Always update refs (for background sessions)
        currentMessageToolsRef.current.toolCalls.push(toolCall);

        // Only update UI state for active session
        if (isActiveSession) {
          setPendingToolCalls((prev) => new Set(prev).add(id));
          setStreamingToolCalls((prev) => [...prev, toolCall]);
        }
      },
      onToolResult: (
        id: string,
        tool: string,
        result: unknown,
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

        const resultSize = JSON.stringify(result).length;
        const hasErrorFlag = !!error;
        console.log(`[CodeChatPage] ${hasErrorFlag ? '❌' : '✅'} ${tool} (${resultSize > 1024 ? Math.round(resultSize/1024) + 'KB' : resultSize + 'B'}) (session: ${sessionId}, viewing: ${activeSessionIdRef.current})`);

        // Always update refs (for background sessions)
        currentMessageToolsRef.current.toolResults.set(id, toolResult);

        // Only update UI state for active session
        if (sessionId === activeSessionIdRef.current) {
          setPendingToolCalls((prev) => {
            const updated = new Set(prev);
            updated.delete(id);
            return updated;
          });

          setStreamingToolResults((prev) => new Map(prev).set(id, toolResult));
        }
      },
      onMessageSaved: (databaseId: string) => {
        // Check if this is a WebSocket disconnection notification
        if (databaseId.includes('AI response saved')) {
          console.log('[CodeChatPage] WebSocket disconnected but message saved - refetching messages');
          if (activeSessionIdRef.current) {
            loadMessages(activeSessionIdRef.current);
          }
          setIsStreaming(false);
          streamingContentRef.current = '';
          setStreamingContent('');
          return;
        }

        const isAssistantMessage = databaseId.startsWith('assistant:');
        const actualId = isAssistantMessage ? databaseId.substring('assistant:'.length) : databaseId;
        const targetRole = isAssistantMessage ? 'assistant' : 'user';

        console.log(`[CodeChatPage] ${targetRole} message saved with database ID:`, actualId);

        setMessages((prev) => {
          let targetIndex = -1;
          for (let i = prev.length - 1; i >= 0; i--) {
            if (prev[i].role === targetRole && prev[i].id.startsWith('msg-')) {
              targetIndex = i;
              break;
            }
          }

          if (targetIndex === -1) {
            console.warn(`[CodeChatPage] No optimistic ${targetRole} message found to update`);
            return prev;
          }

          console.log(`[CodeChatPage] Updating ${targetRole} message ID:`, prev[targetIndex].id, '→', actualId);

          return prev.map((msg, idx) =>
            idx === targetIndex ? { ...msg, id: actualId } : msg
          );
        });
      },
      onError: (err: Error) => {
        setError(`Connection error: ${err.message}`);
        setIsStreaming(false);
        streamingContentRef.current = '';
        setStreamingContent('');
        // WebSocketManager handles reconnection automatically
      },
      onOpen: () => {
        console.log('[CodeChatPage] WebSocket connected');
        setError(null);
      },
      onClose: () => {
        console.log('[CodeChatPage] WebSocket disconnected');
      },
      onReconnect: () => {
        // Called when returning to same session - fetch any missed messages
        console.log('[CodeChatPage] Reconnected to session, fetching any missed messages');
        loadMessages(sessionId);
      },
      onSessionCreated: (subchatId: string) => {
        console.log('[CodeChatPage] New subchat created, refreshing sessions list', subchatId);
        loadSessions();
      },
      onSystemNotification: (notification: SystemNotification) => {
        showSystemNotification(notification);

        if (notification.category === 'execution_stopped') {
          console.log('[CodeChatPage] 🛑 Execution stopped, resetting streaming state');
          setIsStreaming(false);
          streamingContentRef.current = '';
          setStreamingContent('');
        }
      },
    }).catch((err) => {
      console.error('[CodeChatPage] Failed to connect WebSocket:', err);
      setError(`Failed to connect: ${err instanceof Error ? err.message : 'Unknown error'}`);
    });
  }, [wsConnect, streamingSessionId, stopMonitoring, recordChunk]);

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
      // Clear old messages for the UI - but DON'T clear streamingSessionId!
      // We need to track which session is actually streaming to prevent WebSocket disconnect
      setMessages([]);

      // Only clear streaming UI state if:
      // 1. No active streaming at all (!streamingSessionId)
      // 2. Switching to a DIFFERENT session than the one streaming (streamingSessionId !== sessionId)
      // Do NOT clear if switching BACK to the streaming session!
      if (!streamingSessionId) {
        // No active streaming - safe to clear everything
        setStreamingContent('');
        setIsStreaming(false);
        setStreamingToolCalls([]);
        setStreamingToolResults(new Map());
        setPendingToolCalls(new Set());
        streamingContentRef.current = '';
        currentMessageToolsRef.current = { toolCalls: [], toolResults: new Map() };
      } else if (streamingSessionId !== sessionId) {
        // Switching AWAY from streaming session - clear UI state but NOT refs
        // The refs hold the accumulated content for the streaming session
        setStreamingContent('');
        setIsStreaming(false);
        setStreamingToolCalls([]);
        setStreamingToolResults(new Map());
        setPendingToolCalls(new Set());
        // DON'T clear refs - they hold data for the streaming session
      } else {
        // streamingSessionId === sessionId: switching BACK to streaming session
        // Restore UI state from refs
        setStreamingContent(streamingContentRef.current);
        setIsStreaming(true);
        setStreamingToolCalls(currentMessageToolsRef.current.toolCalls);
        setStreamingToolResults(currentMessageToolsRef.current.toolResults);
        setPendingToolCalls(new Set(
          currentMessageToolsRef.current.toolCalls
            .filter(tc => !currentMessageToolsRef.current.toolResults.has(tc.id))
            .map(tc => tc.id)
        ));
      }

      // Note: streamingSessionId is NOT cleared here - it tracks actual AI activity
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

  // Internal message sending function
  const sendMessageInternal = useCallback((text: string) => {
    if (!activeSessionId) return;

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
    resetMetrics();
    startMonitoring();

    // Send message via app-level WebSocket context
    wsSendMessage(text).catch(err => {
      setError(err instanceof Error ? err.message : 'Failed to send message');
      setIsStreaming(false);
      setStreamingContent('');
      streamingContentRef.current = '';
      stopMonitoring();
    });
  }, [activeSessionId, wsSendMessage, resetMetrics, startMonitoring, stopMonitoring]);

  // Memoized callbacks to prevent re-render loops
  const handleArchiveDialogOpen = useCallback(() => {
    setArchiveDialogOpen(true);
  }, []);

  const handleArchiveDialogClose = useCallback(() => {
    setArchiveDialogOpen(false);
  }, []);

  const handleContextModalClose = useCallback(() => {
    setContextModalOpen(false);
  }, []);

  const handleProgressClose = useCallback(() => {
    setProgressEvents([]);
  }, []);

  // Message sending handler
  const handleSendMessage = (text: string) => {
    if (!activeSessionId || isStreaming) return;

    // Check connection state and handle accordingly
    if (wsIsConnected) {
      // Connection is ready, send immediately
      sendMessageInternal(text);
    } else if (wsConnectionState === ConnectionState.CONNECTING) {
      // Connection is in progress, WebSocketManager will queue
      console.log('[CodeChatPage] Connection in progress, sending (will be queued)');
      sendMessageInternal(text);
    } else {
      // Connection is disconnected, reconnect and send
      console.log('[CodeChatPage] WebSocket disconnected, reconnecting...');
      connectWebSocket(activeSessionId);
      sendMessageInternal(text);
    }
  };

  // Stop execution handler
  const handleStopExecution = useCallback(() => {
    console.log('[CodeChatPage] 🛑 Stop button clicked!', {
      isConnected: wsIsConnected,
      isStreaming,
    });

    if (!wsIsConnected) {
      console.log('[CodeChatPage] No WebSocket connection');
      return;
    }

    if (!isStreaming) {
      console.log('[CodeChatPage] Not streaming, nothing to stop');
      return;
    }

    console.log('[CodeChatPage] 🛑 Calling stopExecution...');
    wsStopExecution();
  }, [wsIsConnected, wsStopExecution, isStreaming]);

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
    <ErrorBoundary
      onError={(error, errorInfo) => {
        console.error('[CodeChatPage] Error boundary caught:', error, errorInfo);
        setError(`Application error: ${error.message}`);
      }}
    >
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
              <div className="flex items-center gap-3">
                <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                  {sessions.find((s) => s.id === activeSessionId)?.title || 'Chat'}
                </h1>
                <ConnectionStatusIndicator status={connectionStatus} />
                <ContextIndicator
                  contextMetadata={contextMetadata ?? undefined}
                  isLoading={false}
                  error={contextError?.message}
                  isModalOpen={contextModalOpen}
                  onModalOpenChange={setContextModalOpen}
                  onArchiveClick={handleArchiveDialogOpen}
                />
              </div>
              <div className="flex items-center gap-3">
                <ConversationModeToggle showLabel={true} />

                {/* Archive Messages Button */}
                {(contextError?.code === 'CONTEXT_WARNING' || contextError?.code === 'CONTEXT_CRITICAL' || contextError?.code === 'CONTEXT_FULL') && (
                  <button
                    onClick={handleArchiveDialogOpen}
                    className="p-2 rounded-lg transition-all bg-orange-100 dark:bg-orange-900/30 hover:bg-orange-200 dark:hover:bg-orange-900/50 border-2 border-orange-500 dark:border-orange-400"
                    title="Archive messages to free up context"
                  >
                    <AlertCircle className="w-5 h-5 text-orange-600 dark:text-orange-400" />
                  </button>
                )}
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
                // Only show thinking indicator for THIS session, not when streaming in another session
                const isStreamingThisSession = isStreaming && streamingSessionId === activeSessionId;
                const showIndicator = (isStreamingThisSession && !streamingContent) || isProcessing;

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

              {/* Streaming Assistant Message - only show if this is the streaming session */}
              {isStreaming && streamingSessionId === activeSessionId && (streamingContent || streamingToolCalls.length > 0) && (
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

              {/* Indicator when viewing different session but AI is active elsewhere */}
              {streamingSessionId && streamingSessionId !== activeSessionId && (
                <div className="flex items-center gap-3 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
                  <div className="flex-shrink-0">
                    <div className="relative flex items-center justify-center w-6 h-6">
                      <div className="w-3 h-3 bg-blue-500 rounded-full animate-pulse z-10" />
                      <div className="absolute inset-0 w-3 h-3 bg-blue-500 rounded-full animate-ping opacity-75" />
                    </div>
                  </div>
                  <div className="flex-1">
                    <p className="text-sm text-blue-800 dark:text-blue-200">
                      AI is responding in another chat.
                      <button
                        onClick={() => handleSessionSelect(streamingSessionId)}
                        className="ml-2 underline hover:no-underline font-medium"
                      >
                        Switch to active chat
                      </button>
                    </p>
                  </div>
                </div>
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
          onStopExecution={handleStopExecution}
          isStreaming={isStreaming}
          disabled={!activeSessionId}
          placeholder={
            !activeSessionId
              ? 'Create a new chat to get started'
              : isStreaming
                ? 'AI is responding... Click stop to cancel'
                : 'Type your message...'
          }
        />
      </div>

      {/* Archive Dialog */}
      <ArchiveDialog
        isOpen={archiveDialogOpen}
        messages={messages.filter(m => ['user', 'assistant', 'system'].includes(m.role)) as any}
        onClose={handleArchiveDialogClose}
        onArchive={async (request, sessionId) => {
          try {
            const result = await archiveMessages(sessionId, request.messageIds, request.reason);
            console.log('Messages archived:', result);
            // Refresh messages to show updated context
            if (activeSessionId) {
              await loadMessages(activeSessionId);
            }
          } catch (error) {
            console.error('Failed to archive messages:', error);
            setError(error instanceof Error ? error.message : 'Failed to archive messages');
          }
        }}
        sessionId={activeSessionId || ''}
      />

      {/* Context Status Modal - Rendered at page level */}
      <ContextStatusModal
        isOpen={contextModalOpen}
        contextMetadata={contextMetadata ?? undefined}
        contextError={contextError}
        isLoading={false}
        error={contextError?.message}
        onClose={handleContextModalClose}
        onArchiveClick={handleArchiveDialogOpen}
      />

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
        onClose={handleProgressClose}
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
    </ErrorBoundary>
  );
};

export default CodeChatPage;
