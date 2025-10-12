/**
 * SubchatCreationDialog Component
 *
 * Modal dialog for creating a new subchat with subagent assignment.
 * Allows user to select a subagent and optionally assign task/todo IDs.
 */

import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  MenuItem,
  CircularProgress,
  Alert,
  Box,
  Typography,
} from '@mui/material';
import { subchatService, type Subagent } from '../services/subchatService';

interface SubchatCreationDialogProps {
  open: boolean;
  onClose: () => void;
  parentChatId: string;
  onSubchatCreated: (subchatId: string) => void;
}

export const SubchatCreationDialog: React.FC<SubchatCreationDialogProps> = ({
  open,
  onClose,
  parentChatId,
  onSubchatCreated,
}) => {
  const [subagents, setSubagents] = useState<Subagent[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  // Form state
  const [selectedSubagent, setSelectedSubagent] = useState('');
  const [taskId, setTaskId] = useState('');
  const [todoId, setTodoId] = useState('');

  // Load subagents when dialog opens
  useEffect(() => {
    if (open) {
      loadSubagents();
    }
  }, [open]);

  const loadSubagents = async () => {
    setLoading(true);
    setError(null);
    try {
      const agents = await subchatService.listSubagents();
      setSubagents(agents);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load subagents');
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!selectedSubagent) {
      setError('Please select a subagent');
      return;
    }

    setCreating(true);
    setError(null);
    setSuccess(false);

    try {
      const subchat = await subchatService.createSubchat({
        parentChatId,
        subagentName: selectedSubagent,
        taskId: taskId || undefined,
        todoId: todoId || undefined,
      });

      setSuccess(true);
      setTimeout(() => {
        onSubchatCreated(subchat.id);
        handleClose();
      }, 1000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create subchat');
    } finally {
      setCreating(false);
    }
  };

  const handleClose = () => {
    setSelectedSubagent('');
    setTaskId('');
    setTodoId('');
    setError(null);
    setSuccess(false);
    onClose();
  };

  // Group subagents by category
  const groupedSubagents = subagents.reduce((acc, agent) => {
    if (!acc[agent.category]) {
      acc[agent.category] = [];
    }
    acc[agent.category].push(agent);
    return acc;
  }, {} as Record<string, Subagent[]>);

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>Create Subchat</DialogTitle>

      <DialogContent>
        {loading && (
          <Box display="flex" justifyContent="center" py={3}>
            <CircularProgress />
          </Box>
        )}

        {!loading && error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {success && (
          <Alert severity="success" sx={{ mb: 2 }}>
            Subchat created successfully!
          </Alert>
        )}

        {!loading && (
          <form onSubmit={handleSubmit}>
            <TextField
              select
              fullWidth
              label="Select Subagent"
              value={selectedSubagent}
              onChange={(e) => setSelectedSubagent(e.target.value)}
              required
              disabled={creating}
              margin="normal"
              helperText="Choose a specialist agent for this subchat"
            >
              {Object.entries(groupedSubagents).map(([category, agents]) => [
                <MenuItem key={category} disabled>
                  <Typography variant="subtitle2" color="primary">
                    {category}
                  </Typography>
                </MenuItem>,
                ...agents.map((agent) => (
                  <MenuItem key={agent.name} value={agent.name}>
                    <Box>
                      <Typography variant="body2">{agent.name}</Typography>
                      <Typography variant="caption" color="text.secondary">
                        {agent.description}
                      </Typography>
                    </Box>
                  </MenuItem>
                )),
              ])}
            </TextField>

            <TextField
              fullWidth
              label="Task ID (Optional)"
              value={taskId}
              onChange={(e) => setTaskId(e.target.value)}
              disabled={creating}
              margin="normal"
              placeholder="e.g., 79c045f8-5d41-4050-b0ee-54e1cff9053b"
              helperText="Assign this subchat to a specific task"
            />

            <TextField
              fullWidth
              label="TODO ID (Optional)"
              value={todoId}
              onChange={(e) => setTodoId(e.target.value)}
              disabled={creating}
              margin="normal"
              placeholder="e.g., 7ef26573-68b6-46be-bbc3-dca310477625"
              helperText="Assign this subchat to a specific TODO item"
            />
          </form>
        )}
      </DialogContent>

      <DialogActions>
        <Button onClick={handleClose} disabled={creating}>
          Cancel
        </Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={!selectedSubagent || creating || loading}
          startIcon={creating ? <CircularProgress size={20} /> : null}
        >
          {creating ? 'Creating...' : 'Create Subchat'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default SubchatCreationDialog;
