/**
 * CodeChatPage
 *
 * Main chat interface with session management and WebSocket streaming.
 * Features:
 * - Two-column layout: SessionList + Chat area
 * - WebSocket real-time streaming
 * - Tool calls display with Radix Accordion
 * - Session management (create, rename, delete)
 * - Subchat support with read-only indicator
 */

import React, { useState, useEffect, useRef, useCallback } from 'react';
import { AlertCircle } from 'lucide-react';
import { SessionList } from '@/components/organisms/SessionList';
import { ChatMessage } from '@/components/organisms/ChatMessage';
import { ChatInput } from '@/components/organisms/ChatInput';
import { PerformanceMonitor } from '@/components/organisms/PerformanceMonitor';
import { useStreamingPerformance } from '@/hooks/useStreamingPerformance';
import {
  createSession,
  getSessions,
  getMessages,
  deleteSession,
  updateSession,
  connectChatStream,
  type ChatSession,
  type ChatMessage as ChatMessageType,
  type ChatStreamConnection,
  type ToolCall,
  type ToolResult,
} from '@/services/chatService';

export const CodeChatPage: React.FC = () => {
  // Session state
  const [sessions, setSessions] = useState<ChatSession[]>([]);
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

  // Keep ref in sync with state
  useEffect(() => {
    activeSessionIdRef.current = activeSessionId;
  }, [activeSessionId]);

  // Auto-scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
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
      setSessions(fetchedSessions);

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

  // Load messages for a session
  const loadMessages = async (sessionId: string) => {
    try {
      const fetchedMessages = await getMessages(sessionId);
      setMessages(fetchedMessages);
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
          // Stream complete - add AI message to chat
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
            setMessages((prev) => [...prev, newMessage]);
          }

          // Refresh sessions in case AI created subchats
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
    setSessions((prev) => [newSession, ...prev]);
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
    setSessions((prev) => prev.map((s) => (s.id === sessionId ? updatedSession : s)));
  };

  // Message sending handler
  const handleSendMessage = (text: string) => {
    if (!activeSessionId || isStreaming || !wsConnectionRef.current) return;

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

  // Check if active session is a subchat (read-only)
  const isActiveSessionSubchat = (): boolean => {
    if (!activeSessionId) return false;
    const activeSession = sessions.find((s) => s.id === activeSessionId);
    if (!activeSession) return false;
    return activeSession.title.startsWith('Subchat:') || !!activeSession.parentChatId;
  };

  return (
    <div className="flex h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      {/* Left Sidebar - Session List */}
      <div className="w-80 shrink-0 backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border-r border-white/30 dark:border-gray-700/30">
        <SessionList
          sessions={sessions}
          activeSessionId={activeSessionId}
          onSessionSelect={handleSessionSelect}
          onNewChat={handleNewChat}
          onDeleteSession={handleDeleteSession}
          onDeleteAllSessions={handleDeleteAllSessions}
          onRenameSession={handleRenameSession}
        />
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col">
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
          disabled={!activeSessionId || isStreaming || isActiveSessionSubchat()}
          placeholder={
            !activeSessionId
              ? 'Create a new chat to get started'
              : isActiveSessionSubchat()
              ? 'This subchat is read-only. Monitor the AI agent progress here.'
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
    </div>
  );
};

export default CodeChatPage;
