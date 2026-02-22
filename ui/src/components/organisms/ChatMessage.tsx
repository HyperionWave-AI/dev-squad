/**
 * ChatMessage Organism
 *
 * Displays a single chat message with:
 * - Role-based styling (user/assistant/system)
 * - Markdown rendering for message content
 * - Syntax highlighting for code blocks
 * - Collapsible tool calls display using Radix Accordion
 * - Tool results with status indicators
 */

import React, { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism';
import * as Accordion from '@radix-ui/react-accordion';
import { ChevronDown, Wrench, CheckCircle, XCircle, Clock } from 'lucide-react';
import { cn } from '@/utils';
import { Badge } from '@/components/atoms/Badge';
import { CopyButton } from '@/components/atoms/CopyButton';
import { useConversationMode } from '@/contexts/ConversationModeContext';
import { ToolResultDisplay } from '@/components/molecules/ToolResultDisplay';
import type { ChatMessage as ChatMessageType, ToolCall, ToolResult } from '@/services/chatService';

export interface ChatMessageProps {
  message: ChatMessageType;
  isStreaming?: boolean;
  streamingContent?: string;
  pendingToolCalls?: Set<string>;
  streamingToolCalls?: ToolCall[];
  streamingToolResults?: Map<string, ToolResult>;
}

export const ChatMessage: React.FC<ChatMessageProps> = ({
  message,
  isStreaming = false,
  streamingContent,
  pendingToolCalls = new Set(),
  streamingToolCalls = [],
  streamingToolResults = new Map(),
}) => {
  const isUser = message.role === 'user';
  const isAssistant = message.role === 'assistant';
  const isToolCall = message.role === 'tool_call';
  const isToolResult = message.role === 'tool_result';
  const content = isStreaming && isAssistant ? streamingContent : message.content;

  // Get conversation mode to determine if we should show tool calls
  const { mode } = useConversationMode();
  const showToolDetails = mode === 'debug';
  const TOOL_DISPLAY_LIMIT = 5;
  const [showAllTools, setShowAllTools] = useState(false);

  // Handle tool_call messages - only show in debug mode
  if (isToolCall && message.toolCall && !showToolDetails) {
    return null; // Hide in default mode
  }

  // Render tool_call message in debug mode
  if (isToolCall && message.toolCall && showToolDetails) {
    return (
      <div className="flex w-full mb-4 justify-start px-2">
        <div className="max-w-[75%] w-full">
          <Accordion.Root type="multiple" defaultValue={[]}>
            <Accordion.Item
              value={message.toolCall.id}
              className="border border-blue-200 dark:border-blue-700 rounded-lg overflow-hidden bg-blue-50 dark:bg-blue-900/20"
            >
              <Accordion.Header>
                <Accordion.Trigger className="flex items-center justify-between w-full px-3 py-2 text-sm font-medium bg-blue-100 dark:bg-blue-900/40 hover:bg-blue-200 dark:hover:bg-blue-900/60 transition-colors group">
                  <div className="flex items-center gap-2">
                    <Wrench className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                    <span className="font-mono text-blue-900 dark:text-blue-100">
                      {message.toolCall.name}
                    </span>
                    <Clock className="w-4 h-4 text-blue-500 animate-pulse" />
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-blue-600 dark:text-blue-400">
                      {new Date(message.timestamp).toLocaleTimeString()}
                    </span>
                    <ChevronDown className="w-4 h-4 text-blue-600 dark:text-blue-400 transition-transform group-data-[state=open]:rotate-180" />
                  </div>
                </Accordion.Trigger>
              </Accordion.Header>
              <Accordion.Content className="overflow-hidden data-[state=open]:animate-accordion-down data-[state=closed]:animate-accordion-up">
                <div className="p-3">
                  <div className="text-xs text-gray-700 dark:text-gray-300 font-semibold mb-1">
                    Arguments:
                  </div>
                  <pre className="bg-gray-900 text-gray-100 p-2 rounded text-xs overflow-x-auto">
                    {JSON.stringify(message.toolCall.args, null, 2)}
                  </pre>
                </div>
              </Accordion.Content>
            </Accordion.Item>
          </Accordion.Root>
        </div>
      </div>
    );
  }

  // Handle tool_result messages - only show in debug mode
  if (isToolResult && message.toolResult && !showToolDetails) {
    return null; // Hide in default mode
  }

  // Render tool_result message in debug mode
  if (isToolResult && message.toolResult && showToolDetails) {
    const hasError = !!message.toolResult.error;
    return (
      <div className="flex w-full mb-4 justify-start px-2">
        <div className="max-w-[75%] w-full">
          <Accordion.Root type="multiple" defaultValue={[]}>
            <Accordion.Item
              value={message.toolResult.id}
              className={cn(
                "border rounded-lg overflow-hidden",
                hasError
                  ? "border-red-200 dark:border-red-700 bg-red-50 dark:bg-red-900/20"
                  : "border-green-200 dark:border-green-700 bg-green-50 dark:bg-green-900/20"
              )}
            >
              <Accordion.Header>
                <Accordion.Trigger className={cn(
                  "flex items-center justify-between w-full px-3 py-2 text-sm font-medium transition-colors group",
                  hasError
                    ? "bg-red-100 dark:bg-red-900/40 hover:bg-red-200 dark:hover:bg-red-900/60"
                    : "bg-green-100 dark:bg-green-900/40 hover:bg-green-200 dark:hover:bg-green-900/60"
                )}>
                  <div className="flex items-center gap-2">
                    {hasError ? (
                      <XCircle className="w-4 h-4 text-red-600 dark:text-red-400" />
                    ) : (
                      <CheckCircle className="w-4 h-4 text-green-600 dark:text-green-400" />
                    )}
                    <span className={cn(
                      "font-mono",
                      hasError
                        ? "text-red-900 dark:text-red-100"
                        : "text-green-900 dark:text-green-100"
                    )}>
                      {message.toolResult.name}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={cn(
                      "text-xs",
                      hasError
                        ? "text-red-600 dark:text-red-400"
                        : "text-green-600 dark:text-green-400"
                    )}>
                      {message.toolResult.durationMs}ms
                    </span>
                    <ChevronDown className={cn(
                      "w-4 h-4 transition-transform group-data-[state=open]:rotate-180",
                      hasError
                        ? "text-red-600 dark:text-red-400"
                        : "text-green-600 dark:text-green-400"
                    )} />
                  </div>
                </Accordion.Trigger>
              </Accordion.Header>
              <Accordion.Content className="overflow-hidden data-[state=open]:animate-accordion-down data-[state=closed]:animate-accordion-up">
                <div className="p-3">
                  <ToolResultDisplay
                    toolResult={{
                      id: message.toolResult.id,
                      tool: message.toolResult.name,
                      result: message.toolResult.output,
                      error: message.toolResult.error || null,
                      durationMs: Number(message.toolResult.durationMs)
                    }}
                    hasError={hasError}
                  />
                </div>
              </Accordion.Content>
            </Accordion.Item>
          </Accordion.Root>
        </div>
      </div>
    );
  }

  // Combine persisted tool calls with streaming ones
  const allToolCalls = [
    ...(message.toolCalls || []),
    ...(isStreaming ? streamingToolCalls : []),
  ];

  // Combine persisted tool results with streaming ones
  const allToolResults = new Map([
    ...(message.toolResults || new Map()),
    ...(isStreaming ? streamingToolResults : new Map()),
  ]);

  const hasToolCalls = allToolCalls.length > 0;
  const hasMoreTools = allToolCalls.length > TOOL_DISPLAY_LIMIT;
  const toolsToDisplay = showAllTools || !hasMoreTools
    ? allToolCalls
    : allToolCalls.slice(0, TOOL_DISPLAY_LIMIT);
  const hiddenCount = allToolCalls.length - TOOL_DISPLAY_LIMIT;

  return (
    <div
      className={cn(
        'flex w-full mb-4 group', // Added 'group' for hover state
        isUser ? 'justify-end' : 'justify-start'
      )}
    >
      <div
        className={cn(
          'max-w-[85%] rounded-lg px-4 py-3 shadow-sm relative', // Added 'relative' for positioning
          'overflow-x-auto',
          isUser
            ? 'bg-primary-500 text-white'
            : 'bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700'
        )}
      >
        {/* Copy Button - positioned absolutely, only visible on hover */}
        {content && (
          <CopyButton 
            text={content} 
            className="absolute top-2 right-2" 
          />
        )}

        {/* Role Badge */}
        <div className="flex items-center gap-2 mb-2">
          <Badge variant={isUser ? 'default' : 'secondary'}>
            {message.role}
          </Badge>
          <span className="text-xs opacity-70">
            {new Date(message.timestamp).toLocaleTimeString()}
          </span>
        </div>

        {/* Message Content with Markdown */}
        {content && (
          <div
            className={cn(
              'prose prose-sm max-w-none',
              isUser ? 'prose-invert' : 'dark:prose-invert',
              'prose-pre:bg-gray-900 prose-pre:text-gray-100',
              'prose-code:text-primary-600 dark:prose-code:text-primary-400'
            )}
          >
            <ReactMarkdown
              components={{
                code({ node, inline, className, children, ...props }: any) {
                  const match = /language-(\w+)/.exec(className || '');
                  const language = match ? match[1] : '';

                  if (!inline && language) {
                    return (
                      <SyntaxHighlighter
                        style={oneDark}
                        language={language}
                        PreTag="div"
                        className="rounded-md !mt-2 !mb-2"
                        {...props}
                      >
                        {String(children).replace(/\n$/, '')}
                      </SyntaxHighlighter>
                    );
                  }

                  return (
                    <code className={className} {...props}>
                      {children}
                    </code>
                  );
                },
              }}
            >
              {content}
            </ReactMarkdown>
          </div>
        )}

        {/* Streaming Indicator */}
        {isStreaming && isAssistant && (
          <div className="flex items-center gap-2 mt-2 text-xs opacity-70">
            <div className="flex gap-1">
              <span className="w-2 h-2 bg-primary-500 rounded-full animate-pulse" />
              <span className="w-2 h-2 bg-primary-500 rounded-full animate-pulse delay-75" />
              <span className="w-2 h-2 bg-primary-500 rounded-full animate-pulse delay-150" />
            </div>
            <span>AI is typing...</span>
          </div>
        )}

        {/* Tool Calls Accordion - Only show in debug mode */}
        {showToolDetails && hasToolCalls && (
          <div className="mt-3">
            <Accordion.Root type="multiple">
              {toolsToDisplay.map((toolCall) => {
              const toolResult = allToolResults.get(toolCall.id);
              const isPending = pendingToolCalls.has(toolCall.id);
              const hasError = toolResult?.error;
              const isComplete = toolResult && !hasError;

              return (
                <Accordion.Item
                  key={toolCall.id}
                  value={toolCall.id}
                  className="border border-gray-200 dark:border-gray-700 rounded-lg mb-2 last:mb-0 overflow-hidden"
                >
                  <Accordion.Header>
                    <Accordion.Trigger className="flex items-center justify-between w-full px-3 py-2 text-sm font-medium hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors group">
                      <div className="flex items-center gap-2">
                        <Wrench className="w-4 h-4" />
                        <span className="font-mono">{toolCall.tool}</span>

                        {/* Status Indicator */}
                        {isPending && (
                          <Clock className="w-4 h-4 text-yellow-500 animate-pulse" />
                        )}
                        {isComplete && (
                          <CheckCircle className="w-4 h-4 text-green-500" />
                        )}
                        {hasError && (
                          <XCircle className="w-4 h-4 text-red-500" />
                        )}

                        {/* Duration */}
                        {toolResult?.durationMs && (
                          <span className="text-xs opacity-70">
                            {toolResult.durationMs}ms
                          </span>
                        )}
                      </div>

                      <ChevronDown className="w-4 h-4 transition-transform group-data-[state=open]:rotate-180" />
                    </Accordion.Trigger>
                  </Accordion.Header>

                  <Accordion.Content className="bg-gray-50 dark:bg-gray-900/50 px-3 py-2 text-xs font-mono">
                    {/* Tool Arguments */}
                    <div className="mb-2">
                      <div className="font-semibold text-gray-700 dark:text-gray-300 mb-1">
                        Arguments:
                      </div>
                      <pre className="bg-gray-900 text-gray-100 p-2 rounded overflow-x-auto">
                        {JSON.stringify(toolCall.args, null, 2)}
                      </pre>
                    </div>

                    {/* Tool Result */}
                    {toolResult && (
                      <div>
                        <ToolResultDisplay
                          toolResult={toolResult}
                          hasError={hasError}
                        />
                      </div>
                    )}
                  </Accordion.Content>
                </Accordion.Item>
              );
            })}
          </Accordion.Root>
          {hasMoreTools && (
            <button
              onClick={() => setShowAllTools(!showAllTools)}
              className="mt-2 px-3 py-1.5 text-xs text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700 rounded transition-colors w-full"
            >
              {showAllTools ? (
                <>Show less</>
              ) : (
                <>Show {hiddenCount} more tool call{hiddenCount !== 1 ? 's' : ''}</>
              )}
            </button>
          )}
        </div>
      )}
      </div>
    </div>
  );
};

ChatMessage.displayName = 'ChatMessage';
