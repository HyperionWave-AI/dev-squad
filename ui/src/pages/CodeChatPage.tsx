/**
 * CodeChatPage Component
 *
 * Main chat interface with session management and WebSocket streaming.
 * Features: multiple sessions, message history, real-time AI responses.
 */

import { useState, useEffect, useRef, useCallback } from 'react';
import { Box, Alert, Snackbar, IconButton, Drawer, Divider, Typography } from '@mui/material';
import { AccountTree as SubchatsIcon } from '@mui/icons-material';
import { ChatSessionList } from '../components/ChatSessionList';
import { ChatMessageView } from '../components/ChatMessageView';
import { ChatInputBox } from '../components/ChatInputBox';
import { AgentSelector } from '../components/AgentSelector';
import { SubchatList } from '../components/SubchatList';
import { ConversationModeToggle } from '../components/ConversationModeToggle';
import {
  createSession,
  getSessions,
  getMessages,
  deleteSession,
  updateSession,
  connectChatStream,
  type ChatSession,
  type ChatMessage,
  type ChatStreamConnection,
  type ToolCall,
  type ToolResult,
} from '../services/chatService';

export function CodeChatPage() {
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null);
  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamingContent, setStreamingContent] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [pendingToolCalls, setPendingToolCalls] = useState<Set<string>>(new Set());
  const [subchatsDrawerOpen, setSubchatsDrawerOpen] = useState(false);
  // Real-time streaming tool calls/results (updated immediately, not just on stream end)
  const [streamingToolCalls, setStreamingToolCalls] = useState<ToolCall[]>([]);
  const [streamingToolResults, setStreamingToolResults] = useState<Map<string, ToolResult>>(new Map());

  const wsConnectionRef = useRef<ChatStreamConnection | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const streamingContentRef = useRef<string>('');
  const currentMessageToolsRef = useRef<{
    toolCalls: ToolCall[];
    toolResults: Map<string, ToolResult>;
  }>({ toolCalls: [], toolResults: new Map() });

  // Load sessions on mount and set up auto-refresh polling
  useEffect(() => {
    loadSessions();

    // Auto-refresh sessions every 3 seconds to catch new subchats
    const intervalId = setInterval(() => {
      console.log('[CodeChatPage] 🔄 Auto-refresh: polling for new sessions');
      loadSessions();
    }, 3000);

    return () => {
      console.log('[CodeChatPage] 🛑 Cleaning up auto-refresh interval');
      clearInterval(intervalId);
    };
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

  const loadSessions = async () => {
    console.log('[CodeChatPage] ============================================');
    console.log('[CodeChatPage] loadSessions called at', new Date().toISOString());
    console.log('[CodeChatPage] Current sessions count:', sessions.length);
    try {
      const fetchedSessions = await getSessions();
      console.log('[CodeChatPage] ✅ Fetched sessions from API:', fetchedSessions.length, 'sessions');
      console.log('[CodeChatPage] Full sessions list:', fetchedSessions.map(s => ({
        id: s.id,
        title: s.title,
        parentChatId: s.parentChatId,
        createdAt: s.createdAt
      })));
      console.log('[CodeChatPage] Sessions with parentChatId:', fetchedSessions.filter(s => s.parentChatId).map(s => ({
        id: s.id,
        title: s.title,
        parentChatId: s.parentChatId
      })));

      // Deduplicate sessions by ID (prevents race condition artifacts)
      const sessionMap = new Map<string, typeof fetchedSessions[0]>();
      fetchedSessions.forEach(session => {
        sessionMap.set(session.id, session);
      });
      const deduplicatedSessions = Array.from(sessionMap.values());

      if (deduplicatedSessions.length !== fetchedSessions.length) {
        console.warn('[CodeChatPage] ⚠️ Removed duplicate sessions:',
          fetchedSessions.length - deduplicatedSessions.length,
          'duplicates found');
      }

      console.log('[CodeChatPage] Calling setSessions() with', deduplicatedSessions.length, 'sessions');
      setSessions(deduplicatedSessions);
      console.log('[CodeChatPage] ✅ setSessions() completed');

      // Select first session if none active (ONLY on initial load, not during auto-refresh)
      // Don't auto-select if user already has a session active or is navigating
      if (!activeSessionId && fetchedSessions.length > 0 && sessions.length === 0) {
        // Only auto-select on very first load when sessions array is empty
        setActiveSessionId(fetchedSessions[0].id);
        await loadMessages(fetchedSessions[0].id);
      }
    } catch (err) {
      console.error('[CodeChatPage] ❌ Error loading sessions:', err);
      setError(err instanceof Error ? err.message : 'Failed to load sessions');
    } finally {
      console.log('[CodeChatPage] ============================================');
    }
  };

  const loadMessages = async (sessionId: string) => {
    try {
      const fetchedMessages = await getMessages(sessionId);
      setMessages(fetchedMessages);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load messages');
    }
  };

  const connectWebSocket = useCallback((sessionId: string) => {
    // Disconnect existing connection
    if (wsConnectionRef.current) {
      wsConnectionRef.current.disconnect();
      wsConnectionRef.current = null;
    }

    // Reset streaming content ref and tool tracking
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
            const newMessage: ChatMessage = {
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
              toolCalls: tools.toolCalls.map(tc => tc.tool),
            });
            setMessages((prev) => {
              const updated = [...prev, newMessage];
              console.log('[CodeChatPage] ✅ Messages array updated. New length:', updated.length);
              return updated;
            });
          }

          // Refresh sessions list in case AI created subchats via tool calls
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
        } else {
          // Accumulate streaming content in both ref and state
          streamingContentRef.current += content;
          setStreamingContent((prev) => prev + content);
          setIsStreaming(true);
        }
      },
      onToolCall: (tool: string, args: Record<string, any>, id: string) => {
        console.log('[CodeChatPage] Tool call received:', tool, id);

        // If we have accumulated content before the tool call, save it as a separate message
        if (streamingContentRef.current.trim()) {
          const messageBeforeToolCall: ChatMessage = {
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
        // Update state immediately for real-time display
        setStreamingToolCalls((prev) => [...prev, toolCall]);
      },
      onToolResult: (id: string, tool: string, result: any, error: string | null, durationMs: number) => {
        console.log('[CodeChatPage] Tool result received:', tool, id, error ? 'ERROR' : 'SUCCESS');
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
        // Update state immediately for real-time display
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
      },
      onClose: () => {
        console.log('[CodeChatPage] WebSocket disconnected');
      },
    });

    wsConnectionRef.current = connection;
  }, []);

  const handleNewChat = async () => {
    try {
      const newSession = await createSession('New Chat');
      setSessions((prev) => [newSession, ...prev]);
      setActiveSessionId(newSession.id);
      setMessages([]);
      setStreamingContent('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create chat');
    }
  };

  const handleSessionSelect = async (sessionId: string) => {
    if (sessionId !== activeSessionId) {
      setActiveSessionId(sessionId);
      await loadMessages(sessionId);
      setStreamingContent('');
      setIsStreaming(false);
      // Reset agent selection for new session
      setSelectedAgentId(null);
    }
  };

  const handleAgentChange = (agentId: string | null) => {
    setSelectedAgentId(agentId);
    console.log('[CodeChatPage] Agent changed to:', agentId || 'Default AI');
  };

  const handleDeleteSession = async (sessionId: string) => {
    try {
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
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete session');
    }
  };

  const handleDeleteAllSessions = async () => {
    try {
      // Delete all sessions in parallel
      await Promise.all(sessions.map((session) => deleteSession(session.id)));
      setSessions([]);
      setActiveSessionId(null);
      setMessages([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete all sessions');
    }
  };

  const handleRenameSession = async (sessionId: string, newTitle: string) => {
    try {
      const updatedSession = await updateSession(sessionId, newTitle);
      setSessions((prev) =>
        prev.map((s) => (s.id === sessionId ? updatedSession : s))
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to rename session');
    }
  };

  const handleSendMessage = (text: string) => {
    // Allow sending messages even while streaming (user interruption support)
    if (!activeSessionId || !wsConnectionRef.current) return;

    // Optimistically add user message
    const userMessage: ChatMessage = {
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

    // Send message via WebSocket
    try {
      wsConnectionRef.current.sendMessage(text);
      console.log('[CodeChatPage] Message sent via WebSocket:', text);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send message');
      setIsStreaming(false);
      setStreamingContent('');
      streamingContentRef.current = '';
    }
  };

  const handleSubchatClick = async (subchatId: string) => {
    // Navigate to the subchat immediately (don't call loadSessions first - it causes race condition)
    setActiveSessionId(subchatId);
    await loadMessages(subchatId);
    setSubchatsDrawerOpen(false);
    // Auto-refresh will pick up any new sessions in the next 3-second cycle
  };
  return (
    <Box className="chat-dashboard-container" sx={{ display: 'flex', height: 'calc(100vh - 140px)' }}>
      {/* Left Sidebar - Session List (20%) */}
      <Box sx={{ width: '20%', minWidth: 250, maxWidth: 350 }}>
        <ChatSessionList
          sessions={sessions}
          activeSessionId={activeSessionId}
          onSessionSelect={handleSessionSelect}
          onNewChat={handleNewChat}
          onDeleteSession={handleDeleteSession}
          onDeleteAllSessions={handleDeleteAllSessions}
          onRenameSession={handleRenameSession}
        />
      </Box>

      {/* Main Chat Area (80%) */}
      <Box
        className="chat-main-content"
        sx={{
          flex: 1,
          display: 'flex',
          backgroundColor: 'background.default',
        }}
      >
        {/* Top Bar with Agent Selector, Conversation Mode Toggle, and Subchats Button */}
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 1,
            borderBottom: '1px solid',
            borderColor: 'divider',
            p: 1,
          }}
        >
          <Box sx={{ flex: 1 }}>
            <AgentSelector
              sessionId={activeSessionId}
              selectedAgentId={selectedAgentId}
              onAgentChange={handleAgentChange}
              disabled={isStreaming}
            />
          </Box>
          <ConversationModeToggle />
          <IconButton
            onClick={() => setSubchatsDrawerOpen(!subchatsDrawerOpen)}
            disabled={!activeSessionId}
            color={subchatsDrawerOpen ? 'primary' : 'default'}
            sx={{ mr: 1 }}
          >
            <SubchatsIcon />
          </IconButton>
        </Box>

        {/* Messages View */}
        <Box sx={{ flex: 1, overflow: 'hidden' }}>
          <ChatMessageView
            messages={messages}
            isStreaming={isStreaming}
            streamingContent={streamingContent}
            pendingToolCalls={pendingToolCalls}
            streamingToolCalls={streamingToolCalls}
            streamingToolResults={streamingToolResults}
          />
        </Box>

        {/* Input Box */}
        <ChatInputBox
          onSendMessage={handleSendMessage}
          disabled={!activeSessionId}
          placeholder={
            !activeSessionId
              ? 'Create a new chat to get started'
              : 'Type your message...'
          }
        />
      </Box>

      {/* Subchats Drawer */}
      <Drawer
        anchor="right"
        open={subchatsDrawerOpen}
        onClose={() => setSubchatsDrawerOpen(false)}
        PaperProps={{
          sx: { width: 400, p: 2 },
        }}
      >
        <Box sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
          <SubchatsIcon color="primary" />
          <Typography variant="h6">Parallel Workflows</Typography>
        </Box>
        <Divider sx={{ mb: 2 }} />
        {activeSessionId && (
          <SubchatList
            parentChatId={activeSessionId}
            onSubchatClick={handleSubchatClick}
            onSubchatCreated={loadSessions}
          />
        )}
      </Drawer>

      {/* Error Snackbar */}
      <Snackbar
        open={!!error}
        autoHideDuration={6000}
        onClose={() => setError(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert
          onClose={() => setError(null)}
          severity="error"
          variant="filled"
          sx={{ width: '100%' }}
        >
          {error}
        </Alert>
      </Snackbar>
    </Box>
  );
}
