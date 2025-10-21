/**
 * ChatMessageView Component
 *
 * Displays chat messages in conversation view with markdown support.
 * Features: user/AI message styling, timestamps, auto-scroll, typing indicator.
 */

import { useEffect, useRef } from 'react';
import { Box, Typography, Paper, CircularProgress } from '@mui/material';
import { Person, SmartToy } from '@mui/icons-material';
import ReactMarkdown from 'react-markdown';
import type { ChatMessage, ToolCall, ToolResult } from '../services/chatService';
import { ToolCallCard } from './ToolCallCard';
import { ToolResultCard } from './ToolResultCard';

interface ChatMessageViewProps {
  messages: ChatMessage[];
  isStreaming: boolean;
  streamingContent?: string;
  pendingToolCalls?: Set<string>;
  streamingToolCalls?: ToolCall[];
  streamingToolResults?: Map<string, ToolResult>;
}

export function ChatMessageView({
  messages,
  isStreaming,
  streamingContent = '',
  pendingToolCalls = new Set(),
  streamingToolCalls = [],
  streamingToolResults = new Map(),
}: ChatMessageViewProps) {
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new messages arrive or tool calls update
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages.length, streamingContent, streamingToolCalls.length, streamingToolResults.size]);

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      hour12: true,
    });
  };

  const renderMessage = (message: ChatMessage) => {
    const isUser = message.role === 'user';
    const isSystem = message.role === 'system';
    const isToolCall = message.role === 'tool_call';
    const isToolResult = message.role === 'tool_result';

    // Handle system messages
    if (isSystem) {
      return (
        <Box
          key={message.id}
          sx={{
            display: 'flex',
            justifyContent: 'center',
            mb: 2,
          }}
        >
          <Paper
            elevation={0}
            sx={{
              px: 3,
              py: 1,
              backgroundColor: 'grey.100',
              borderRadius: 2,
              maxWidth: '80%',
            }}
          >
            <Typography variant="caption" color="text.secondary">
              {message.content}
            </Typography>
          </Paper>
        </Box>
      );
    }

    // Handle tool_call messages
    if (isToolCall && message.toolCall) {
      return (
        <Box
          key={message.id}
          sx={{
            display: 'flex',
            justifyContent: 'flex-start',
            mb: 2,
            px: 2,
          }}
        >
          <Box sx={{ maxWidth: '75%' }}>
            <ToolCallCard
              tool={message.toolCall.name}
              args={message.toolCall.args}
              id={message.toolCall.id}
              timestamp={new Date(message.timestamp)}
              isPending={false}
            />
          </Box>
        </Box>
      );
    }

    // Handle tool_result messages
    if (isToolResult && message.toolResult) {
      return (
        <Box
          key={message.id}
          sx={{
            display: 'flex',
            justifyContent: 'flex-start',
            mb: 2,
            px: 2,
          }}
        >
          <Box sx={{ maxWidth: '75%' }}>
            <ToolResultCard
              tool={message.toolResult.name}
              result={message.toolResult.output}
              error={message.toolResult.error}
              durationMs={message.toolResult.durationMs}
            />
          </Box>
        </Box>
      );
    }

    return (
      <Box
        key={message.id}
        sx={{
          display: 'flex',
          justifyContent: isUser ? 'flex-end' : 'flex-start',
          mb: 2,
          px: 2,
        }}
      >
        <Box
          sx={{
            display: 'flex',
            flexDirection: isUser ? 'row-reverse' : 'row',
            gap: 1.5,
            maxWidth: '75%',
            alignItems: 'flex-start',
          }}
        >
          {/* Avatar Icon */}
          <Box
            sx={{
              width: 32,
              height: 32,
              borderRadius: '50%',
              backgroundColor: isUser ? 'primary.main' : 'grey.300',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
              mt: 0.5,
            }}
          >
            {isUser ? (
              <Person sx={{ fontSize: 20, color: 'white' }} />
            ) : (
              <SmartToy sx={{ fontSize: 20, color: 'grey.700' }} />
            )}
          </Box>

          {/* Message Content */}
          <Box sx={{ flex: 1 }}>
            <Paper
              elevation={1}
              sx={{
                px: 2,
                py: 1.5,
                backgroundColor: isUser ? 'primary.main' : 'grey.100',
                color: isUser ? 'white' : 'text.primary',
                borderRadius: 2,
                borderTopLeftRadius: isUser ? 2 : 0.5,
                borderTopRightRadius: isUser ? 0.5 : 2,
              }}
            >
              {isUser ? (
                <Typography
                  variant="body1"
                  sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}
                >
                  {message.content}
                </Typography>
              ) : (
                <Box>
                  <Box
                    sx={{
                      '& p': { mb: 1, mt: 0 },
                      '& p:last-child': { mb: 0 },
                      '& code': {
                        backgroundColor: 'grey.200',
                        px: 0.5,
                        py: 0.25,
                        borderRadius: 0.5,
                        fontFamily: 'monospace',
                        fontSize: '0.875em',
                      },
                      '& pre': {
                        backgroundColor: 'grey.800',
                        color: 'white',
                        p: 1.5,
                        borderRadius: 1,
                        overflowX: 'auto',
                        mb: 1,
                      },
                      '& pre code': {
                        backgroundColor: 'transparent',
                        color: 'inherit',
                      },
                      '& ul, & ol': { pl: 2.5, mb: 1 },
                      '& li': { mb: 0.5 },
                    }}
                  >
                    <ReactMarkdown>{message.content}</ReactMarkdown>
                  </Box>
                </Box>
              )}
            </Paper>

            {/* Tool execution cards */}
            {!isUser && message.toolCalls && message.toolCalls.length > 0 && (
              <Box sx={{ mt: 1 }}>
                {message.toolCalls.map((toolCall) => {
                  const isPending = pendingToolCalls.has(toolCall.id);
                  const toolResult = message.toolResults?.get(toolCall.id);

                  return (
                    <Box key={toolCall.id}>
                      <ToolCallCard
                        tool={toolCall.tool}
                        args={toolCall.args}
                        id={toolCall.id}
                        timestamp={toolCall.timestamp}
                        isPending={isPending}
                      />
                      {toolResult && (
                        <ToolResultCard
                          tool={toolResult.tool}
                          result={toolResult.result}
                          error={toolResult.error}
                          durationMs={toolResult.durationMs}
                        />
                      )}
                    </Box>
                  );
                })}
              </Box>
            )}

            {/* Timestamp */}
            <Typography
              variant="caption"
              color="text.secondary"
              sx={{
                display: 'block',
                mt: 0.5,
                px: 1,
                textAlign: isUser ? 'right' : 'left',
              }}
            >
              {formatTimestamp(message.timestamp)}
            </Typography>
          </Box>
        </Box>
      </Box>
    );
  };

  return (
    <Box
      ref={containerRef}
      sx={{
        height: '100%',
        overflowY: 'auto',
        overflowX: 'hidden',
        backgroundColor: 'background.default',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      {/* Messages */}
      <Box sx={{ flexGrow: 1, py: 2 }}>
        {messages.length === 0 && !isStreaming ? (
          <Box
            sx={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              px: 3,
              textAlign: 'center',
            }}
          >
            <SmartToy sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
            <Typography variant="h6" color="text.secondary" gutterBottom>
              Start a conversation
            </Typography>
            <Typography variant="body2" color="text.disabled">
              Type your message below to begin chatting with the AI assistant
            </Typography>
          </Box>
        ) : (
          <>
            {messages.map((message) => renderMessage(message))}

            {/* Streaming Message */}
            {isStreaming && streamingContent && (
              <Box
                sx={{
                  display: 'flex',
                  justifyContent: 'flex-start',
                  mb: 2,
                  px: 2,
                }}
              >
                <Box
                  sx={{
                    display: 'flex',
                    gap: 1.5,
                    maxWidth: '75%',
                    alignItems: 'flex-start',
                  }}
                >
                  {/* AI Avatar */}
                  <Box
                    sx={{
                      width: 32,
                      height: 32,
                      borderRadius: '50%',
                      backgroundColor: 'grey.300',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      flexShrink: 0,
                      mt: 0.5,
                    }}
                  >
                    <SmartToy sx={{ fontSize: 20, color: 'grey.700' }} />
                  </Box>

                  {/* Streaming Content */}
                  <Paper
                    elevation={1}
                    sx={{
                      px: 2,
                      py: 1.5,
                      backgroundColor: 'grey.100',
                      borderRadius: 2,
                      borderTopLeftRadius: 0.5,
                      flex: 1,
                    }}
                  >
                    <Box
                      sx={{
                        '& p': { mb: 1, mt: 0 },
                        '& p:last-child': { mb: 0 },
                        '& code': {
                          backgroundColor: 'grey.200',
                          px: 0.5,
                          py: 0.25,
                          borderRadius: 0.5,
                          fontFamily: 'monospace',
                          fontSize: '0.875em',
                        },
                        '& pre': {
                          backgroundColor: 'grey.800',
                          color: 'white',
                          p: 1.5,
                          borderRadius: 1,
                          overflowX: 'auto',
                        },
                      }}
                    >
                      <ReactMarkdown>{streamingContent}</ReactMarkdown>
                    </Box>
                  </Paper>
                </Box>
              </Box>
            )}

            {/* Streaming Tool Calls (Real-time) */}
            {isStreaming && streamingToolCalls.length > 0 && (
              <Box sx={{ px: 2, mb: 2 }}>
                {streamingToolCalls.map((toolCall) => {
                  const isPending = pendingToolCalls.has(toolCall.id);
                  const toolResult = streamingToolResults.get(toolCall.id);

                  return (
                    <Box key={toolCall.id} sx={{ mb: 1 }}>
                      <ToolCallCard
                        tool={toolCall.tool}
                        args={toolCall.args}
                        id={toolCall.id}
                        timestamp={toolCall.timestamp}
                        isPending={isPending}
                      />
                      {toolResult && (
                        <ToolResultCard
                          tool={toolResult.tool}
                          result={toolResult.result}
                          error={toolResult.error}
                          durationMs={toolResult.durationMs}
                        />
                      )}
                    </Box>
                  );
                })}
              </Box>
            )}

            {/* Typing Indicator */}
            {isStreaming && !streamingContent && streamingToolCalls.length === 0 && (
              <Box
                sx={{
                  display: 'flex',
                  justifyContent: 'flex-start',
                  mb: 2,
                  px: 2,
                }}
              >
                <Box sx={{ display: 'flex', gap: 1.5, alignItems: 'center' }}>
                  <Box
                    sx={{
                      width: 32,
                      height: 32,
                      borderRadius: '50%',
                      backgroundColor: 'grey.300',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <SmartToy sx={{ fontSize: 20, color: 'grey.700' }} />
                  </Box>
                  <Paper
                    elevation={1}
                    sx={{
                      px: 2,
                      py: 1.5,
                      backgroundColor: 'grey.100',
                      borderRadius: 2,
                      display: 'flex',
                      alignItems: 'center',
                      gap: 1,
                    }}
                  >
                    <CircularProgress size={16} />
                    <Typography variant="body2" color="text.secondary">
                      Thinking...
                    </Typography>
                  </Paper>
                </Box>
              </Box>
            )}
          </>
        )}

        {/* Scroll anchor */}
        <div ref={messagesEndRef} />
      </Box>
    </Box>
  );
}
