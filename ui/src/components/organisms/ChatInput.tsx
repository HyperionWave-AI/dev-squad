/**
 * ChatInput Organism
 *
 * Chat input component with:
 * - Auto-resizing textarea
 * - Send button with loading state
 * - Enter key to send (Shift+Enter for new line)
 * - Disabled state support
 * - Character count display (optional)
 */

import React, { useState, useRef, useEffect } from 'react';
import { Send } from 'lucide-react';
import { cn } from '@/utils';
import { Button } from '@/components/atoms/Button';
import { Textarea } from '@/components/atoms/Textarea';

export interface ChatInputProps {
  onSendMessage: (message: string) => void;
  disabled?: boolean;
  placeholder?: string;
  maxLength?: number;
  showCharCount?: boolean;
  className?: string;
}

export const ChatInput: React.FC<ChatInputProps> = ({
  onSendMessage,
  disabled = false,
  placeholder = 'Type your message...',
  maxLength = 5000,
  showCharCount = false,
  className,
}) => {
  const [message, setMessage] = useState('');
  const [isSending, setIsSending] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    // Reset height to auto to get the correct scrollHeight
    textarea.style.height = 'auto';

    // Set height to scrollHeight (clamped between min and max)
    const minHeight = 60; // ~2 lines
    const maxHeight = 200; // ~8 lines
    const newHeight = Math.min(Math.max(textarea.scrollHeight, minHeight), maxHeight);
    textarea.style.height = `${newHeight}px`;
  }, [message]);

  const handleSend = async () => {
    const trimmedMessage = message.trim();
    if (!trimmedMessage || disabled || isSending) return;

    setIsSending(true);
    try {
      onSendMessage(trimmedMessage);
      setMessage('');

      // Reset textarea height after clearing
      if (textareaRef.current) {
        textareaRef.current.style.height = 'auto';
      }
    } catch (error) {
      console.error('[ChatInput] Error sending message:', error);
    } finally {
      setIsSending(false);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Send on Enter (without Shift)
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const charCount = message.length;
  const isOverLimit = charCount > maxLength;
  const canSend = message.trim().length > 0 && !isOverLimit && !disabled && !isSending;

  return (
    <div
      className={cn(
        'border-t border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4',
        className
      )}
    >
      <div className="flex gap-3 items-end">
        {/* Textarea */}
        <div className="flex-1 relative">
          <Textarea
            ref={textareaRef}
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={disabled || isSending}
            className={cn(
              'min-h-[60px] max-h-[200px] resize-none',
              'focus-visible:ring-offset-0',
              isOverLimit && 'border-red-500 focus-visible:ring-red-500'
            )}
            aria-label="Chat message input"
            aria-describedby={showCharCount ? 'char-count' : undefined}
          />

          {/* Character Count */}
          {showCharCount && (
            <div
              id="char-count"
              className={cn(
                'absolute bottom-2 right-2 text-xs',
                isOverLimit ? 'text-red-500' : 'text-gray-400 dark:text-gray-500'
              )}
              aria-live="polite"
            >
              {charCount}/{maxLength}
            </div>
          )}
        </div>

        {/* Send Button */}
        <Button
          onClick={handleSend}
          disabled={!canSend}
          variant="primary"
          size="icon"
          className="shrink-0"
          aria-label="Send message"
        >
          {isSending ? (
            <div className="w-5 h-5 border-2 border-white border-t-transparent rounded-full animate-spin" />
          ) : (
            <Send className="w-5 h-5" />
          )}
        </Button>
      </div>

      {/* Helper Text */}
      <div className="mt-2 text-xs text-gray-500 dark:text-gray-400 flex items-center justify-between">
        <span>Press Enter to send, Shift+Enter for new line</span>
        {disabled && (
          <span className="text-yellow-600 dark:text-yellow-500 font-medium">
            {placeholder.includes('read-only') ? 'Read-only mode' : 'Disabled'}
          </span>
        )}
      </div>
    </div>
  );
};

ChatInput.displayName = 'ChatInput';
