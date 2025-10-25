import { useState, useEffect } from 'react';
import {
  Box,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Chip,
  CircularProgress,
  Alert,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import { SmartToy, AutoAwesome } from '@mui/icons-material';
import { aiService, type Subagent } from '../services/aiService';

interface AgentSelectorProps {
  sessionId: string | null;
  selectedAgentId: string | null;
  onAgentChange: (agentId: string | null) => void;
  disabled?: boolean;
}

export function AgentSelector({
  sessionId,
  selectedAgentId,
  onAgentChange,
  disabled = false,
}: AgentSelectorProps) {
  const [subagents, setSubagents] = useState<Subagent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [updating, setUpdating] = useState(false);

  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));

  // Load subagents on mount
  useEffect(() => {
    loadSubagents();
  }, []);

  const loadSubagents = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await aiService.listSubagents();
      setSubagents(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load subagents');
    } finally {
      setLoading(false);
    }
  };

  const handleChange = async (agentId: string | null) => {
    if (!sessionId || updating) return;

    setUpdating(true);
    setError(null);

    try {
      // Update session subagent via API
      await aiService.setChatSessionSubagent(sessionId, agentId);
      onAgentChange(agentId);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update agent');
      // Don't update UI on error - keep previous selection
    } finally {
      setUpdating(false);
    }
  };

  const getSelectedAgentName = () => {
    if (!selectedAgentId) return 'Default AI';
    const agent = subagents.find((a) => a.id === selectedAgentId);
    return agent?.name || 'Unknown Agent';
  };

  if (loading) {
    return (
      <Box 
        sx={{ 
          display: 'flex', 
          alignItems: 'center', 
          gap: 1, 
          p: { xs: 1.5, sm: 2 },
          minHeight: { xs: 56, sm: 64 }
        }}
      >
        <CircularProgress size={isMobile ? 16 : 20} />
        <span style={{ fontSize: isMobile ? '0.875rem' : '1rem' }}>
          Loading agents...
        </span>
      </Box>
    );
  }

  if (error) {
    return (
      <Alert 
        severity="error" 
        sx={{ 
          m: { xs: 1, sm: 2 },
          fontSize: { xs: '0.875rem', sm: '1rem' }
        }}
      >
        {error}
      </Alert>
    );
  }

  return (
    <Box 
      sx={{ 
        p: { xs: 1.5, sm: 2 }, 
        borderBottom: 1, 
        borderColor: 'divider',
        backgroundColor: 'background.paper'
      }}
    >
      <FormControl 
        fullWidth 
        size={isMobile ? "medium" : "small"} 
        disabled={disabled || updating}
        sx={{
          '& .MuiSelect-select': {
            minHeight: { xs: 44, sm: 'auto' }, // Touch-friendly height on mobile
            fontSize: { xs: '1rem', sm: '0.875rem' },
            py: { xs: 1.5, sm: 1 }
          },
          '& .MuiInputLabel-root': {
            fontSize: { xs: '1rem', sm: '0.875rem' }
          }
        }}
      >
        <InputLabel id="agent-selector-label">Active Agent</InputLabel>
        <Select
          labelId="agent-selector-label"
          value={selectedAgentId || ''}
          label="Active Agent"
          onChange={(e) => handleChange(e.target.value || null)}
          startAdornment={
            updating ? (
              <CircularProgress size={isMobile ? 20 : 16} sx={{ mr: 1 }} />
            ) : selectedAgentId ? (
              <SmartToy sx={{ mr: 1, fontSize: isMobile ? 24 : 20 }} />
            ) : (
              <AutoAwesome sx={{ mr: 1, fontSize: isMobile ? 24 : 20 }} />
            )
          }
          renderValue={() => (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <span style={{ fontSize: isMobile ? '1rem' : '0.875rem' }}>
                {getSelectedAgentName()}
              </span>
              {selectedAgentId && (
                <Chip
                  label="Custom"
                  size={isMobile ? "medium" : "small"}
                  color="primary"
                  sx={{ 
                    height: isMobile ? 24 : 20, 
                    fontSize: isMobile ? '0.75rem' : '0.7rem',
                    '& .MuiChip-label': {
                      px: isMobile ? 1 : 0.75
                    }
                  }}
                />
              )}
            </Box>
          )}
          MenuProps={{
            PaperProps: {
              sx: {
                maxHeight: { xs: 300, sm: 400 }, // Limit height on mobile
                '& .MuiMenuItem-root': {
                  minHeight: { xs: 48, sm: 'auto' }, // Touch-friendly menu items
                  py: { xs: 1.5, sm: 1 },
                  fontSize: { xs: '1rem', sm: '0.875rem' }
                }
              }
            }
          }}
        >
          <MenuItem value="">
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <AutoAwesome fontSize={isMobile ? "medium" : "small"} />
              <span>Default AI</span>
            </Box>
          </MenuItem>
          {subagents.map((agent) => (
            <MenuItem key={agent.id} value={agent.id}>
              <Box sx={{ 
                display: 'flex', 
                alignItems: 'center', 
                gap: 1,
                width: '100%',
                overflow: 'hidden'
              }}>
                <SmartToy fontSize={isMobile ? "medium" : "small"} />
                <Box sx={{ 
                  display: 'flex', 
                  flexDirection: { xs: 'column', sm: 'row' },
                  alignItems: { xs: 'flex-start', sm: 'center' },
                  gap: { xs: 0.5, sm: 1 },
                  overflow: 'hidden',
                  width: '100%'
                }}>
                  <span style={{ fontWeight: 500 }}>{agent.name}</span>
                  {agent.description && (
                    <span 
                      style={{ 
                        fontSize: isMobile ? '0.875rem' : '0.75rem', 
                        color: 'gray',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                        maxWidth: isMobile ? '200px' : '150px'
                      }}
                    >
                      {isMobile ? 
                        agent.description.substring(0, 30) + (agent.description.length > 30 ? '...' : '') :
                        agent.description.substring(0, 40) + (agent.description.length > 40 ? '...' : '')
                      }
                    </span>
                  )}
                </Box>
              </Box>
            </MenuItem>
          ))}
        </Select>
      </FormControl>
    </Box>
  );
}