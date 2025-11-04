/**
 * Conversation Mode Toggle Component
 *
 * Toggle button that allows users to switch between:
 * - Default mode: Friendly natural language explanations
 * - Debug mode: Technical details with tool calls and JSON
 *
 * The preference is persisted across sessions
 */

import React from 'react';
import { ToggleButtonGroup, ToggleButton, Box, Typography, Tooltip, CircularProgress } from '@mui/material';
import { Message as MessageIcon, Code as CodeIcon } from '@mui/icons-material';
import { useConversationMode } from '../contexts/ConversationModeContext';
import type { ConversationMode } from '../services/userSettingsService';

export const ConversationModeToggle: React.FC = () => {
  const { mode, setMode, isLoading, error } = useConversationMode();

  const handleChange = async (_event: React.MouseEvent<HTMLElement>, newMode: ConversationMode | null) => {
    if (newMode && newMode !== mode) {
      await setMode(newMode);
    }
  };

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', px: 2 }}>
        <CircularProgress size={20} />
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
      <ToggleButtonGroup
        value={mode}
        exclusive
        onChange={handleChange}
        size="small"
        sx={{
          backgroundColor: 'background.paper',
          '& .MuiToggleButton-root': {
            py: 0.5,
            px: 1.5,
            border: '1px solid',
            borderColor: 'divider',
            textTransform: 'none',
            fontWeight: 500,
            fontSize: '0.875rem',
            '&.Mui-selected': {
              backgroundColor: 'primary.main',
              color: 'white',
              '&:hover': {
                backgroundColor: 'primary.dark',
              },
            },
          },
        }}
      >
        <ToggleButton value="default">
          <Tooltip title="Show natural language explanations - friendly for non-technical users" arrow>
            <Box display="flex" alignItems="center" gap={0.75}>
              <MessageIcon fontSize="small" />
              <Typography variant="caption" sx={{ fontWeight: 'inherit' }}>
                Default
              </Typography>
            </Box>
          </Tooltip>
        </ToggleButton>

        <ToggleButton value="debug">
          <Tooltip title="Show technical details, tool calls, and JSON payloads - for developers" arrow>
            <Box display="flex" alignItems="center" gap={0.75}>
              <CodeIcon fontSize="small" />
              <Typography variant="caption" sx={{ fontWeight: 'inherit' }}>
                Debug
              </Typography>
            </Box>
          </Tooltip>
        </ToggleButton>
      </ToggleButtonGroup>

      {error && (
        <Tooltip title={`Error: ${error}`} arrow>
          <Box
            sx={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              backgroundColor: 'error.main',
            }}
          />
        </Tooltip>
      )}
    </Box>
  );
};

export default ConversationModeToggle;
