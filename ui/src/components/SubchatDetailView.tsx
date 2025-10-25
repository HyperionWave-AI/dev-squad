/**
 * SubchatDetailView Component
 *
 * Real-time view of subchat messages, tool calls, and results.
 * Connects to subchat's WebSocket stream to show live agent activity.
 */

import { useState, useEffect, useRef, useCallback } from 'react';
import {
  Box,
  Paper,
  Typography,
  CircularProgress,
  Alert,
  IconButton,
  Chip,
} from '@mui/material';
import { Close as CloseIcon } from '@mui/icons-material';
import { subchatService, type Subchat } from '../services/subchatService';
import {
  connectChatStream,
  getMessages,
  type ChatMessage,
  type ChatStreamConnection,
  type ToolCall,
  type ToolResult,
} from '../services/chatService';
import { ChatMessageView } from './ChatMessageView';

interface SubchatDetailViewProps {
  subchatId: string;
  onClose?: () => void;
}

export function SubchatDetailView({ subchatId, onClose }: SubchatDetailViewProps) {
  const [subchat, setSubchat] = useState<Subchat | null>(null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'error' | 'disconnected'>('disconnected');

  // Streaming state
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamingContent, setStreamingContent] = useState('');
  const [pendingToolCalls, setPendingToolCalls] = useState<Set<string>>(new Set());
  const [streamingToolCalls, setStreamingToolCalls] = useState<ToolCall[]>([]);
  const [streamingToolResults, setStreamingToolResults] = useState<Map<string, ToolResult>>(new Map());

  const wsConnectionRef = useRef<ChatStreamConnection | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const streamingContentRef = useRef<string>('');
  const currentMessageToolsRef = useRef<{
    toolCalls: ToolCall[];
    toolResults: Map<string, ToolResult>;
  }>({ toolCalls: [], toolResults: new Map() });

  // Load existing messages for a session
  const loadMessages = async (sessionId: string) => {
    try {
      const fetchedMessages = await getMessages(sessionId);
      setMessages(fetchedMessages);
      console.log('[SubchatDetailView] Loaded', fetchedMessages.length, 'existing messages');
    } catch (err) {
      console.error('[SubchatDetailView] Failed to load messages:', err);
      setError(err instanceof Error ? err.message : 'Failed to load messages');
    }
  };

  // Load subchat details
  useEffect(() => {
    const loadSubchat = async () => {
      try {
        setLoading(true);
        setError(null);
        const data = await subchatService.getSubchat(subchatId);
        setSubchat(data);

        // Check if subchat has a session ID
        if (!data.sessionId) {
          setError('Subchat does not have an active session yet. Messages will appear once the agent starts working.');
          setLoading(false);
          return;
        }

        // Load existing messages before connecting to WebSocket
        await loadMessages(data.sessionId);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load subchat details');
        setLoading(false);
      }
    };

    loadSubchat();
  }, [subchatId]);

  // Connect WebSocket when subchat with sessionId is loaded
  useEffect(() => {
    if (subchat?.sessionId) {
      connectWebSocket(subchat.sessionId);
      setLoading(false);
    }

    // Cleanup on unmount or when subchat changes
    return () => {
      if (wsConnectionRef.current) {
        wsConnectionRef.current.disconnect();
        wsConnectionRef.current = null;
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, [subchat?.sessionId]);

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
    setConnectionStatus('connecting');

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
          }

          // Clear streaming state
          streamingContentRef.current = '';
          currentMessageToolsRef.current = { toolCalls: [], toolResults: new Map() };
          setStreamingContent('');
          setIsStreaming(false);
          setPendingToolCalls(new Set());
          setStreamingToolCalls([]);
          setStreamingToolResults(new Map());
        } else {
          // Accumulate streaming content
          streamingContentRef.current += content;
          setStreamingContent((prev) => prev + content);
          setIsStreaming(true);
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
      onToolResult: (id: string, tool: string, result: any, error: string | null, durationMs: number) => {
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
        setConnectionStatus('error');
        setIsStreaming(false);
        streamingContentRef.current = '';
        setStreamingContent('');

        // Attempt reconnect after 3 seconds
        reconnectTimeoutRef.current = setTimeout(() => {
          console.log('[SubchatDetailView] Attempting to reconnect...');
          connectWebSocket(sessionId);
        }, 3000);
      },
      onOpen: () => {
        console.log('[SubchatDetailView] WebSocket connected');
        setConnectionStatus('connected');
        setError(null);
      },
      onClose: () => {
        console.log('[SubchatDetailView] WebSocket disconnected');
        setConnectionStatus('disconnected');
      },
    });

    wsConnectionRef.current = connection;
  }, []);

  // Connection status indicator
  const getStatusColor = (): 'success' | 'warning' | 'error' | 'default' => {
    switch (connectionStatus) {
      case 'connected':
        return 'success';
      case 'connecting':
        return 'warning';
      case 'error':
        return 'error';
      default:
        return 'default';
    }
  };

  const getStatusText = (): string => {
    switch (connectionStatus) {
      case 'connected':
        return 'Live';
      case 'connecting':
        return 'Connecting...';
      case 'error':
        return 'Connection Error';
      default:
        return 'Disconnected';
    }
  };

  return (
    <Paper
      elevation={2}
      sx={{
        border: '2px solid',
        borderColor: 'primary.main',
        borderRadius: 2,
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          p: 2,
          backgroundColor: 'primary.main',
          color: 'white',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Typography variant="h6">
            {subchat?.subagentName || 'Loading...'}
          </Typography>
          <Chip
            label={getStatusText()}
            size="small"
            color={getStatusColor()}
            sx={{
              backgroundColor: 'white',
              fontWeight: 'bold',
            }}
          />
        </Box>
        {onClose && (
          <IconButton
            onClick={onClose}
            size="small"
            sx={{ color: 'white' }}
          >
            <CloseIcon />
          </IconButton>
        )}
      </Box>

      {/* Content */}
      <Box sx={{ height: 400, position: 'relative' }}>
        {loading && (
          <Box
            sx={{
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
              height: '100%',
            }}
          >
            <CircularProgress />
          </Box>
        )}

        {error && !loading && (
          <Box sx={{ p: 2 }}>
            <Alert severity={connectionStatus === 'error' ? 'warning' : 'info'}>
              {error}
            </Alert>
          </Box>
        )}

        {!loading && !error && subchat && (
          <Box sx={{ height: '100%', overflow: 'hidden' }}>
            <ChatMessageView
              messages={messages}
              isStreaming={isStreaming}
              streamingContent={streamingContent}
              pendingToolCalls={pendingToolCalls}
              streamingToolCalls={streamingToolCalls}
              streamingToolResults={streamingToolResults}
            />
          </Box>
        )}

        {!loading && !error && messages.length === 0 && !isStreaming && (
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'center',
              alignItems: 'center',
              height: '100%',
              p: 3,
              textAlign: 'center',
            }}
          >
            <Typography variant="body1" color="text.secondary" gutterBottom>
              No messages yet
            </Typography>
            <Typography variant="body2" color="text.disabled">
              Messages will appear here when the agent starts working
            </Typography>
          </Box>
        )}
      </Box>
    </Paper>
  );
}

export default SubchatDetailView;
