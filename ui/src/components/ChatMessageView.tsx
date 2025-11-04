/**
 * ChatMessageView Component
 *
 * Displays chat messages in conversation view with markdown support.
 * Features: user/AI message styling, timestamps, auto-scroll, typing indicator.
 */

import { useEffect, useRef } from 'react';
import { Box, Typography, Paper } from '@mui/material';
import { Person, SmartToy } from '@mui/icons-material';
import ReactMarkdown from 'react-markdown';
import type { ChatMessage, ToolCall, ToolResult } from '../services/chatService';
import { ToolCallCard } from './ToolCallCard';
import { ToolResultCard } from './ToolResultCard';
import { useConversationMode } from '../contexts/ConversationModeContext';
import { TaskProgressIndicator } from './TaskProgressIndicator';

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
  const { mode } = useConversationMode();

  // Only show tool calls/results in debug mode
  const showToolDetails = mode === 'debug';

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
      // Filter out scaffold enforcement messages
      const scaffoldPatterns = [
        'FORCED WRITE SCAFFOLD',
        'WRITE-ONLY MODE ENFORCEMENT',
        'CURRENT EXECUTION SCORE',
        'IMPLEMENT NOW - DO NOT READ',
        'Your NEXT tool call MUST be',
        'You are BLOCKED from calling',
        'SCORING:',
        '🚨',
        '╔══════════════════════════════════════════════════════════════╗',
      ];

      const isScaffoldMessage = scaffoldPatterns.some(pattern =>
        message.content.includes(pattern)
      );

      // Don't render scaffold messages
      if (isScaffoldMessage) {
        return null;
      }

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
              overflowWrap: 'break-word',
              wordBreak: 'break-word',
            }}
          >
            <Typography variant="caption" color="text.secondary">
              {message.content}
            </Typography>
          </Paper>
        </Box>
      );
    }

    // Handle tool_call messages - only in debug mode
    if (isToolCall && message.toolCall && showToolDetails) {
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
          <Box sx={{ maxWidth: '75%', minWidth: 0 }}>
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

    // Handle tool_result messages - only in debug mode
    if (isToolResult && message.toolResult && showToolDetails) {
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
          <Box sx={{ maxWidth: '75%', minWidth: 0 }}>
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

    // Skip tool_call and tool_result messages in default mode (return null to not render them)
    if ((isToolCall || isToolResult) && !showToolDetails) {
      return null;
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
            minWidth: 0,
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
          <Box sx={{ flex: 1, minWidth: 0 }}>
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
                maxWidth: '100%',
                minWidth: 0,
                overflowWrap: 'break-word',
                wordBreak: 'break-word',
              }}
            >
              {isUser ? (
                <Typography
                  variant="body2"
                  sx={{ 
                    whiteSpace: 'pre-wrap', 
                    wordBreak: 'break-word',
                    overflowWrap: 'anywhere',
                    fontSize: '0.9rem',
                    maxWidth: '100%',
                  }}
                >
                  {message.content}
                </Typography>
              ) : (
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
                      overflowWrap: 'break-word',
                      wordBreak: 'break-all',
                    },
                    '& pre': {
                      backgroundColor: 'grey.800',
                      color: 'white',
                      p: 1.5,
                      borderRadius: 1,
                      overflowX: 'auto',
                      overflowY: 'hidden',
                      mb: 1,
                      maxWidth: '100%',
                    },
                    '& pre code': {
                      backgroundColor: 'transparent',
                      color: 'inherit',
                      overflowWrap: 'normal',
                      wordBreak: 'normal',
                    },
                    '& ul, & ol': { pl: 2.5, mb: 1 },
                    '& li': { mb: 0.5 },
                    '& a': {
                      overflowWrap: 'break-word',
                      wordBreak: 'break-all',
                    },
                    maxWidth: '100%',
                    overflowWrap: 'break-word',
                    wordBreak: 'break-word',
                  }}
                >
                  <ReactMarkdown>{message.content}</ReactMarkdown>
                </Box>
              )}
            </Paper>

            {/* Tool execution cards - debug mode shows technical details */}
            {!isUser && showToolDetails && message.toolCalls && message.toolCalls.length > 0 && (
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
                        timestamp={new Date(message.timestamp)}
                        isPending={isPending}
                      />
                      {toolResult && (
                        <Box sx={{ mt: 1 }}>
                          <ToolResultCard
                            tool={toolResult.tool}
                            result={toolResult.result}
                            error={toolResult.error}
                            durationMs={toolResult.durationMs}
                          />
                        </Box>
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
                textAlign: isUser ? 'right' : 'left',
                mt: 0.5,
                fontSize: '0.75rem',
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
        flex: 1,
        overflowY: 'auto',
        overflowX: 'hidden',
        p: 1,
        maxWidth: '100%',
        minWidth: 0,
      }}
    >
      {/* Task Progress Indicator - only in debug mode */}
      {showToolDetails && <TaskProgressIndicator mode="working" />}

      {/* Messages */}
      {messages.map(renderMessage)}

      {/* Streaming tool calls - debug mode only */}
      {showToolDetails && streamingToolCalls.map((toolCall) => (
        <Box
          key={toolCall.id}
          sx={{
            display: 'flex',
            justifyContent: 'flex-start',
            mb: 2,
            px: 2,
          }}
        >
          <Box sx={{ maxWidth: '75%', minWidth: 0 }}>
            <ToolCallCard
              tool={toolCall.tool}
              args={toolCall.args}
              id={toolCall.id}
              timestamp={new Date()}
              isPending={true}
            />
            {streamingToolResults.has(toolCall.id) && (
              <Box sx={{ mt: 1 }}>
                <ToolResultCard
                  tool={streamingToolResults.get(toolCall.id)!.tool}
                  result={streamingToolResults.get(toolCall.id)!.result}
                  error={streamingToolResults.get(toolCall.id)!.error}
                  durationMs={streamingToolResults.get(toolCall.id)!.durationMs}
                />
              </Box>
            )}
          </Box>
        </Box>
      ))}

      {/* Streaming message */}
      {isStreaming && (
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
              flexDirection: 'row',
              gap: 1.5,
              maxWidth: '75%',
              minWidth: 0,
              alignItems: 'flex-start',
            }}
          >
            {/* Avatar Icon */}
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
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Paper
                elevation={1}
                sx={{
                  px: 2,
                  py: 1.5,
                  backgroundColor: 'grey.100',
                  color: 'text.primary',
                  borderRadius: 2,
                  borderTopLeftRadius: 0.5,
                  borderTopRightRadius: 2,
                  maxWidth: '100%',
                  minWidth: 0,
                  overflowWrap: 'break-word',
                  wordBreak: 'break-word',
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
                      overflowWrap: 'break-word',
                      wordBreak: 'break-all',
                    },
                    '& pre': {
                      backgroundColor: 'grey.800',
                      color: 'white',
                      p: 1.5,
                      borderRadius: 1,
                      overflowX: 'auto',
                      overflowY: 'hidden',
                      mb: 1,
                      maxWidth: '100%',
                    },
                    '& pre code': {
                      backgroundColor: 'transparent',
                      color: 'inherit',
                      overflowWrap: 'normal',
                      wordBreak: 'normal',
                    },
                    '& ul, & ol': { pl: 2.5, mb: 1 },
                    '& li': { mb: 0.5 },
                    '& a': {
                      overflowWrap: 'break-word',
                      wordBreak: 'break-all',
                    },
                    maxWidth: '100%',
                    overflowWrap: 'break-word',
                    wordBreak: 'break-word',
                  }}
                >
                  <ReactMarkdown>{streamingContent}</ReactMarkdown>
                </Box>
                {/* Typing indicator */}
                <Box
                  sx={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: 0.5,
                    mt: streamingContent ? 1 : 0,
                  }}
                >
                  <Box
                    sx={{
                      width: 4,
                      height: 4,
                      borderRadius: '50%',
                      backgroundColor: 'grey.500',
                      animation: 'pulse 1.5s ease-in-out infinite',
                      animationDelay: '0s',
                      '@keyframes pulse': {
                        '0%, 70%, 100%': {
                          opacity: 0.4,
                          transform: 'scale(1)',
                        },
                        '35%': {
                          opacity: 1,
                          transform: 'scale(1.2)',
                        },
                      },
                    }}
                  />
                  <Box
                    sx={{
                      width: 4,
                      height: 4,
                      borderRadius: '50%',
                      backgroundColor: 'grey.500',
                      animation: 'pulse 1.5s ease-in-out infinite',
                      animationDelay: '0.2s',
                      '@keyframes pulse': {
                        '0%, 70%, 100%': {
                          opacity: 0.4,
                          transform: 'scale(1)',
                        },
                        '35%': {
                          opacity: 1,
                          transform: 'scale(1.2)',
                        },
                      },
                    }}
                  />
                  <Box
                    sx={{
                      width: 4,
                      height: 4,
                      borderRadius: '50%',
                      backgroundColor: 'grey.500',
                      animation: 'pulse 1.5s ease-in-out infinite',
                      animationDelay: '0.4s',
                      '@keyframes pulse': {
                        '0%, 70%, 100%': {
                          opacity: 0.4,
                          transform: 'scale(1)',
                        },
                        '35%': {
                          opacity: 1,
                          transform: 'scale(1.2)',
                        },
                      },
                    }}
                  />
                </Box>
              </Paper>
            </Box>
          </Box>
        </Box>
      )}

      {/* Scroll anchor */}
      <div ref={messagesEndRef} />
    </Box>
  );
}