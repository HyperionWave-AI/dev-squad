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
import { AlertCircle, BarChart3, X } from 'lucide-react';
import { SessionList } from '@/components/organisms/SessionList';
import { ChatMessage } from '@/components/organisms/ChatMessage';
import { ChatInput } from '@/components/organisms/ChatInput';
import { PerformanceMonitor } from '@/components/organisms/PerformanceMonitor';
import { ProgressTracker, type ProgressEvent } from '@/components/organisms/ProgressTracker';
import { MetricsDashboard } from '@/components/organisms/MetricsDashboard';
import { ConversationModeToggle } from '@/components/molecules/ConversationModeToggle';
import { useStreamingPerformance } from '@/hooks/useStreamingPerformance';
import {
  createSession,
  getSessions,
  getMessages,
  deleteSession,
  updateSession,
  connectChatStream,
  type ChatMessage as ChatMessageType,
  type ChatStreamConnection,
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
  activeSubagentId?: string; // Indicates session is being processed by a subagent
}

export const CodeChatPage: React.FC = () => {
  // Session state
  const [sessions, setSessions] = useState<SessionItem[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);

  // Messages state
  const [messages, setMessages] = useState<ChatMessageType[]>([]);

  // Streaming state
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamingContent, setStreamingContent] = useState('');
  const [pendingToolCalls, setPendingToolCalls] = useState<Set<string>>(new Set());
  const [streamingToolCalls, setStreamingToolCalls] = useState<ToolCall[]>([]);
  const [streamingToolResults, setStreamingToolResults] = useState<Map<string, ToolResult>>(
    new Map()
  );

  // Progress tracking state
  const [progressEvents, setProgressEvents] = useState<ProgressEvent[]>([]);

  // Metrics drawer state
  const [metricsDrawerOpen, setMetricsDrawerOpen] = useState(false);

  // Error state
  const [error, setError] = useState<string | null>(null);

  // Performance monitoring
  const performance = useStreamingPerformance();

  // Refs for WebSocket and streaming content
  const wsConnectionRef = useRef<ChatStreamConnection | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
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

  // Load sessions on mount and set up auto-refresh
  useEffect(() => {
    loadSessions();

    // Auto-refresh sessions every 5 seconds to catch new subchats
    const intervalId = setInterval(() => {
      loadSessions();
    }, 5000);

    return () => clearInterval(intervalId);
  }, []);

  // Auto-refresh messages for active session (Fix #2: Messages not appearing without refresh)
  useEffect(() => {
    if (!activeSessionId) return;

    // Poll messages every 3 seconds for active session
    const intervalId = setInterval(() => {
      // Check WebSocket health and reconnect if needed
      if (!wsConnectionRef.current || wsConnectionRef.current.ws.readyState !== WebSocket.OPEN) {
        console.log('[CodeChatPage] WebSocket disconnected, reconnecting...');
        connectWebSocket(activeSessionId);
      }

      // Only poll if not streaming
      if (!isStreaming) {
        loadMessages(activeSessionId);
      }
    }, 3000);

    return () => clearInterval(intervalId);
  }, [activeSessionId, isStreaming]);

  // Connect WebSocket when active session changes
  useEffect(() => {
    if (activeSessionId) {
      connectWebSocket(activeSessionId);
    }

    // Cleanup on session change or unmount
    return () => {
      if (wsConnectionRef.current) {
        wsConnectionRef.current.disconnect();
        wsConnectionRef.current = null;
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, [activeSessionId]);

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
        activeSubagentId: session.activeSubagentId, // Preserve processing indicator
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

  // Deduplicate messages by ID (keeps latest version of each message)
  const deduplicateMessages = (messages: ChatMessageType[]): ChatMessageType[] => {
    const seen = new Set<string>();
    const unique: ChatMessageType[] = [];

    // Process in reverse to keep the LATEST version of each message
    for (let i = messages.length - 1; i >= 0; i--) {
      const msg = messages[i];
      if (!seen.has(msg.id)) {
        seen.add(msg.id);
        unique.unshift(msg); // Add to front to maintain order
      }
    }

    return unique;
  };

  // Load messages for a session
  const loadMessages = async (sessionId: string) => {
    try {
      const fetchedMessages = await getMessages(sessionId);
      setMessages((prev) => {
        // Merge fetched messages with existing, deduplicate
        const merged = [...prev, ...fetchedMessages];
        return deduplicateMessages(merged);
      });
    } catch (err) {
      console.error('[CodeChatPage] Error loading messages:', err);
      setError(err instanceof Error ? err.message : 'Failed to load messages');
    }
  };

  // Connect to WebSocket for streaming
  const connectWebSocket = useCallback((sessionId: string) => {
    // Disconnect existing connection
    if (wsConnectionRef.current) {
      wsConnectionRef.current.disconnect();
      wsConnectionRef.current = null;
    }

    // Reset streaming state
    streamingContentRef.current = '';
    currentMessageToolsRef.current = { toolCalls: [], toolResults: new Map() };
    setPendingToolCalls(new Set());
    setStreamingToolCalls([]);
    setStreamingToolResults(new Map());

    // Connect to WebSocket
    const connection = connectChatStream(sessionId, {
      onMessage: (content: string, done: boolean) => {
        if (done) {
          // Stream complete - save final AI message if there's any remaining content
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
            console.log('[CodeChatPage] ✅ Creating completed AI message:', {
              id: newMessage.id,
              hasContent: !!finalContent,
              toolCallsCount: tools.toolCalls.length,
              toolResultsCount: tools.toolResults.size,
            });
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
          setPendingToolCalls(new Set());
          setStreamingToolCalls([]);
          setStreamingToolResults(new Map());

          // Stop performance monitoring
          performance.stopMonitoring();
        } else {
          // Accumulate streaming content
          streamingContentRef.current += content;
          setStreamingContent((prev) => prev + content);
          setIsStreaming(true);

          // Record chunk for performance monitoring
          performance.recordChunk(content);
        }
      },
      onToolCall: (tool: string, args: Record<string, any>, id: string) => {
        console.log('[CodeChatPage] Tool call received:', tool, id);

        // If we have accumulated content before the tool call, save it as a separate message
        if (streamingContentRef.current.trim()) {
          const messageBeforeToolCall: ChatMessageType = {
            id: `msg-${Date.now()}`,
            sessionId,
            role: 'assistant',
            content: streamingContentRef.current,
            timestamp: new Date().toISOString(),
          };
          setMessages((prev) => [...prev, messageBeforeToolCall]);
          console.log('[CodeChatPage] Saved message before tool call');

          // Clear streaming content for next message
          streamingContentRef.current = '';
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
        currentMessageToolsRef.current.toolResults.set(id, toolResult);
        setPendingToolCalls((prev) => {
          const updated = new Set(prev);
          updated.delete(id);
          return updated;
        });
        setStreamingToolResults((prev) => new Map(prev).set(id, toolResult));
      },
      onMessageSaved: (databaseId: string) => {
        // FIX: Reconcile optimistic message ID with database ID
        // Find the most recent user message and update its ID
        console.log('[CodeChatPage] Message saved with database ID:', databaseId);
        setMessages((prev) => {
          // Find most recent message with optimistic ID (msg-timestamp)
          const updated = [...prev];
          for (let i = updated.length - 1; i >= 0; i--) {
            if (updated[i].role === 'user' && updated[i].id.startsWith('msg-')) {
              console.log('[CodeChatPage] Updating message ID:', updated[i].id, '→', databaseId);
              updated[i] = { ...updated[i], id: databaseId };
              break;
            }
          }
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
        setError(null);
      },
      onClose: () => {
        console.log('[CodeChatPage] WebSocket disconnected');
      },
    });

    wsConnectionRef.current = connection;
  }, []);

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
    };
    setSessions((prev) => [sessionItem, ...prev]);
    setActiveSessionId(newSession.id);
    setMessages([]);
    setStreamingContent('');
  };

  const handleSessionSelect = async (sessionId: string) => {
    if (sessionId !== activeSessionId) {
      setActiveSessionId(sessionId);
      await loadMessages(sessionId);
      setStreamingContent('');
      setIsStreaming(false);
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
    };
    setSessions((prev) => prev.map((s) => (s.id === sessionId ? sessionItem : s)));
  };

  // Message sending handler
  const handleSendMessage = (text: string) => {
    if (!activeSessionId || isStreaming) return;

    // Ensure WebSocket is connected before sending
    if (!wsConnectionRef.current || wsConnectionRef.current.ws.readyState !== WebSocket.OPEN) {
      console.log('[CodeChatPage] WebSocket not connected, reconnecting...');
      connectWebSocket(activeSessionId);
      // Wait briefly for connection to establish before sending
      setTimeout(() => {
        if (wsConnectionRef.current && wsConnectionRef.current.ws.readyState === WebSocket.OPEN) {
          handleSendMessage(text);
        } else {
          setError('WebSocket connection failed. Please try again.');
        }
      }, 500);
      return;
    }

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
    try {
      wsConnectionRef.current.sendMessage(text);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send message');
      setIsStreaming(false);
      setStreamingContent('');
      streamingContentRef.current = '';
      performance.stopMonitoring();
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
              {isStreaming && streamingContent && (
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
        progress={progressEvents}
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
