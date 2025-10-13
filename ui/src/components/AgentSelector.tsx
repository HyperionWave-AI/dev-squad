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
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, p: 2 }}>
        <CircularProgress size={20} />
        <span>Loading agents...</span>
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        {error}
      </Alert>
    );
  }

  return (
    <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
      <FormControl fullWidth size="small" disabled={disabled || updating}>
        <InputLabel id="agent-selector-label">Active Agent</InputLabel>
        <Select
          labelId="agent-selector-label"
          value={selectedAgentId || ''}
          label="Active Agent"
          onChange={(e) => handleChange(e.target.value || null)}
          startAdornment={
            updating ? (
              <CircularProgress size={16} sx={{ mr: 1 }} />
            ) : selectedAgentId ? (
              <SmartToy sx={{ mr: 1, fontSize: 20 }} />
            ) : (
              <AutoAwesome sx={{ mr: 1, fontSize: 20 }} />
            )
          }
          renderValue={() => (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              {getSelectedAgentName()}
              {selectedAgentId && (
                <Chip
                  label="Custom"
                  size="small"
                  color="primary"
                  sx={{ height: 20, fontSize: '0.7rem' }}
                />
              )}
            </Box>
          )}
        >
          <MenuItem value="">
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <AutoAwesome fontSize="small" />
              Default AI
            </Box>
          </MenuItem>
          {subagents.map((agent) => (
            <MenuItem key={agent.id} value={agent.id}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <SmartToy fontSize="small" />
                {agent.name}
                {agent.description && (
                  <span style={{ fontSize: '0.85rem', color: 'gray', marginLeft: 8 }}>
                    - {agent.description.substring(0, 40)}
                    {agent.description.length > 40 ? '...' : ''}
                  </span>
                )}
              </Box>
            </MenuItem>
          ))}
        </Select>
      </FormControl>
    </Box>
  );
}
