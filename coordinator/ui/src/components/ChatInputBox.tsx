/**
 * ChatInputBox Component
 *
 * Multiline text input with send button for chat messages.
 * Features: Enter to send, Shift+Enter for new line, character count, auto-clear.
 */

import { useState, type KeyboardEvent } from 'react';
import { Box, TextField, IconButton, Typography } from '@mui/material';
import { Send } from '@mui/icons-material';

interface ChatInputBoxProps {
  onSendMessage: (text: string) => void;
  disabled?: boolean;
  placeholder?: string;
}

export function ChatInputBox({
  onSendMessage,
  disabled = false,
  placeholder = 'Type your message...',
}: ChatInputBoxProps) {
  const [message, setMessage] = useState('');

  const handleSend = () => {
    const trimmed = message.trim();
    if (trimmed && !disabled) {
      onSendMessage(trimmed);
      setMessage('');
    }
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    // Enter without Shift = send
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault();
      handleSend();
    }
    // Shift+Enter = new line (default behavior)
  };

  const charCount = message.length;
  const showCharCount = charCount > 500;
  const isOverLimit = charCount > 5000;

  return (
    <Box
      sx={{
        borderTop: '1px solid',
        borderColor: 'divider',
        backgroundColor: 'background.paper',
        p: 2,
      }}
    >
      <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-end' }}>
        {/* Text Input */}
        <TextField
          fullWidth
          multiline
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          placeholder={placeholder}
          minRows={1}
          maxRows={8}
          sx={{
            '& .MuiOutlinedInput-root': {
              backgroundColor: 'background.default',
              maxHeight: '200px',
              overflowY: 'auto',
            },
          }}
          helperText={
            showCharCount && (
              <Typography
                variant="caption"
                color={isOverLimit ? 'error' : 'text.secondary'}
                sx={{ display: 'block', textAlign: 'right' }}
              >
                {charCount.toLocaleString()} characters
                {isOverLimit && ' (limit: 5,000)'}
              </Typography>
            )
          }
        />

        {/* Send Button */}
        <IconButton
          color="primary"
          onClick={handleSend}
          disabled={disabled || !message.trim() || isOverLimit}
          sx={{
            width: 48,
            height: 48,
            backgroundColor: 'primary.main',
            color: 'white',
            '&:hover': {
              backgroundColor: 'primary.dark',
            },
            '&.Mui-disabled': {
              backgroundColor: 'action.disabledBackground',
              color: 'action.disabled',
            },
          }}
        >
          <Send />
        </IconButton>
      </Box>

      {/* Hint Text */}
      {!disabled && (
        <Typography
          variant="caption"
          color="text.disabled"
          sx={{ display: 'block', mt: 0.5, ml: 1 }}
        >
          Press Enter to send, Shift+Enter for new line
        </Typography>
      )}
    </Box>
  );
}
