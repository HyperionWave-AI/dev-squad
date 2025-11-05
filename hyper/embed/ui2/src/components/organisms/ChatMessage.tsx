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

import React from 'react';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism';
import * as Accordion from '@radix-ui/react-accordion';
import { ChevronDown, Wrench, CheckCircle, XCircle, Clock } from 'lucide-react';
import { cn } from '@/utils';
import { Badge } from '@/components/atoms/Badge';
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
  const content = isStreaming && isAssistant ? streamingContent : message.content;

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

  return (
    <div
      className={cn(
        'flex w-full mb-4',
        isUser ? 'justify-end' : 'justify-start'
      )}
    >
      <div
        className={cn(
          'max-w-[85%] rounded-lg px-4 py-3 shadow-sm',
          isUser
            ? 'bg-primary-500 text-white'
            : 'bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700'
        )}
      >
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

        {/* Tool Calls Accordion */}
        {hasToolCalls && (
          <Accordion.Root type="multiple" className="mt-3">
            {allToolCalls.map((toolCall, index) => {
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
                        <div className="font-semibold text-gray-700 dark:text-gray-300 mb-1">
                          {hasError ? 'Error:' : 'Result:'}
                        </div>
                        <pre
                          className={cn(
                            'p-2 rounded overflow-x-auto',
                            hasError
                              ? 'bg-red-900/20 text-red-300'
                              : 'bg-gray-900 text-gray-100'
                          )}
                        >
                          {hasError
                            ? toolResult.error
                            : JSON.stringify(toolResult.result, null, 2)}
                        </pre>
                      </div>
                    )}
                  </Accordion.Content>
                </Accordion.Item>
              );
            })}
          </Accordion.Root>
        )}
      </div>
    </div>
  );
};

ChatMessage.displayName = 'ChatMessage';
