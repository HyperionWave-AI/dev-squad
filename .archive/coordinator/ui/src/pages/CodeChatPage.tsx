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
  const [loading, setLoading] = useState(true);
  const [pendingToolCalls, setPendingToolCalls] = useState<Set<string>>(new Set());
  const [subchatsDrawerOpen, setSubchatsDrawerOpen] = useState(false);

  const wsConnectionRef = useRef<ChatStreamConnection | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const streamingContentRef = useRef<string>('');
  const currentMessageToolsRef = useRef<{
    toolCalls: ToolCall[];
    toolResults: Map<string, ToolResult>;
  }>({ toolCalls: [], toolResults: new Map() });

  // Load sessions on mount
  useEffect(() => {
    loadSessions();
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
    try {
      setLoading(true);
      const fetchedSessions = await getSessions();
      setSessions(fetchedSessions);

      // Select first session if none active
      if (!activeSessionId && fetchedSessions.length > 0) {
        setActiveSessionId(fetchedSessions[0].id);
        await loadMessages(fetchedSessions[0].id);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load sessions');
    } finally {
      setLoading(false);
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
            setMessages((prev) => [...prev, newMessage]);
            console.log('[CodeChatPage] AI response completed with', tools.toolCalls.length, 'tool calls');
          }
          // Clear streaming state
          streamingContentRef.current = '';
          currentMessageToolsRef.current = { toolCalls: [], toolResults: new Map() };
          setStreamingContent('');
          setIsStreaming(false);
          setPendingToolCalls(new Set());
        } else {
          // Accumulate streaming content in both ref and state
          streamingContentRef.current += content;
          setStreamingContent((prev) => prev + content);
          setIsStreaming(true);
        }
      },
      onToolCall: (tool: string, args: Record<string, any>, id: string) => {
        console.log('[CodeChatPage] Tool call received:', tool, id);
        const toolCall: ToolCall = {
          id,
          tool,
          args,
          timestamp: new Date(),
        };
        currentMessageToolsRef.current.toolCalls.push(toolCall);
        setPendingToolCalls((prev) => new Set(prev).add(id));
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
    if (!activeSessionId || isStreaming || !wsConnectionRef.current) return;

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

  const handleSubchatClick = (subchatId: string) => {
    // Navigate to the subchat by setting it as active session
    setActiveSessionId(subchatId);
    loadMessages(subchatId);
    setSubchatsDrawerOpen(false);
  };

  return (
    <Box sx={{ display: 'flex', height: 'calc(100vh - 140px)' }}>
      {/* Left Sidebar - Session List (20%) */}
      <Box sx={{ width: '20%', minWidth: 250, maxWidth: 350 }}>
        <ChatSessionList
          sessions={sessions}
          activeSessionId={activeSessionId}
          onSessionSelect={handleSessionSelect}
          onNewChat={handleNewChat}
          onDeleteSession={handleDeleteSession}
          onRenameSession={handleRenameSession}
          loading={loading}
        />
      </Box>

      {/* Main Chat Area (80%) */}
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          backgroundColor: 'background.default',
        }}
      >
        {/* Top Bar with Agent Selector and Subchats Button */}
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 1,
            borderBottom: '1px solid',
            borderColor: 'divider',
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
          />
        </Box>

        {/* Input Box */}
        <ChatInputBox
          onSendMessage={handleSendMessage}
          disabled={!activeSessionId || isStreaming}
          placeholder={
            activeSessionId
              ? 'Type your message...'
              : 'Create a new chat to get started'
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
