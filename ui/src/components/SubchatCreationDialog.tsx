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
  Stack,
  Chip,
  IconButton,
  Divider,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import {
  Close as CloseIcon,
  Person as PersonIcon,
  Assignment as TaskIcon,
  CheckCircle as TodoIcon,
} from '@mui/icons-material';
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
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  
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
        // Pass session ID (not subchat ID) for navigation
        if (subchat.sessionId) {
          onSubchatCreated(subchat.sessionId);
        }
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

  // Get selected subagent details
  const selectedSubagentDetails = subagents.find(agent => agent.name === selectedSubagent);

  return (
    <Dialog 
      open={open} 
      onClose={handleClose} 
      maxWidth="md" 
      fullWidth
      fullScreen={isMobile}
      PaperProps={{
        sx: {
          borderRadius: isMobile ? 0 : 3,
          maxHeight: isMobile ? '100vh' : '90vh',
        },
      }}
    >
      {/* Enhanced Dialog Title */}
      <DialogTitle
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          pb: 2,
          borderBottom: '1px solid',
          borderColor: 'divider',
        }}
      >
        <Box display="flex" alignItems="center" gap={1.5}>
          <Box
            sx={{
              p: 1,
              borderRadius: 2,
              backgroundColor: 'primary.50',
              color: 'primary.main',
            }}
          >
            <PersonIcon />
          </Box>
          <Box>
            <Typography variant="h6" component="div" sx={{ fontWeight: 600 }}>
              Create New Subchat
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Delegate work to a specialist agent
            </Typography>
          </Box>
        </Box>
        <IconButton
          onClick={handleClose}
          disabled={creating}
          sx={{
            color: 'text.secondary',
            '&:hover': {
              backgroundColor: 'error.50',
              color: 'error.main',
            },
          }}
        >
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <DialogContent sx={{ p: 0 }}>
        <Box sx={{ p: 3 }}>
          {loading && (
            <Box 
              display="flex" 
              flexDirection="column"
              alignItems="center" 
              justifyContent="center"
              py={6}
              gap={2}
            >
              <CircularProgress size={48} />
              <Typography variant="body2" color="text.secondary">
                Loading available agents...
              </Typography>
            </Box>
          )}

          {!loading && error && (
            <Alert 
              severity="error" 
              sx={{ 
                mb: 3,
                borderRadius: 2,
              }}
              onClose={() => setError(null)}
            >
              {error}
            </Alert>
          )}

          {success && (
            <Alert 
              severity="success" 
              sx={{ 
                mb: 3,
                borderRadius: 2,
              }}
            >
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                  Subchat created successfully!
                </Typography>
                <Typography variant="body2">
                  Redirecting to the new subchat...
                </Typography>
              </Box>
            </Alert>
          )}

          {!loading && (
            <Stack spacing={3}>
              {/* Subagent Selection */}
              <Box>
                <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
                  Select Specialist Agent
                </Typography>
                <TextField
                  select
                  fullWidth
                  label="Choose Subagent"
                  value={selectedSubagent}
                  onChange={(e) => setSelectedSubagent(e.target.value)}
                  required
                  disabled={creating}
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      borderRadius: 2,
                    },
                  }}
                  helperText="Each agent specializes in different areas of development"
                >
                  {Object.entries(groupedSubagents).map(([category, agents]) => [
                    <MenuItem key={category} disabled sx={{ py: 1 }}>
                      <Typography 
                        variant="subtitle2" 
                        color="primary.main"
                        sx={{ fontWeight: 600, textTransform: 'uppercase', fontSize: '0.75rem' }}
                      >
                        {category}
                      </Typography>
                    </MenuItem>,
                    ...agents.map((agent) => (
                      <MenuItem 
                        key={agent.name} 
                        value={agent.name}
                        sx={{ 
                          py: 1.5,
                          pl: 3,
                          '&:hover': {
                            backgroundColor: 'primary.50',
                          },
                        }}
                      >
                        <Box sx={{ width: '100%' }}>
                          <Typography variant="body1" sx={{ fontWeight: 500, mb: 0.5 }}>
                            {agent.name}
                          </Typography>
                          <Typography variant="caption" color="text.secondary" display="block">
                            {agent.description}
                          </Typography>
                        </Box>
                      </MenuItem>
                    )),
                  ])}
                </TextField>

                {/* Selected Agent Preview */}
                {selectedSubagentDetails && (
                  <Box 
                    sx={{ 
                      mt: 2, 
                      p: 2, 
                      backgroundColor: 'primary.50', 
                      borderRadius: 2,
                      border: '1px solid',
                      borderColor: 'primary.200',
                    }}
                  >
                    <Box display="flex" alignItems="center" gap={1} mb={1}>
                      <PersonIcon fontSize="small" color="primary" />
                      <Typography variant="subtitle2" color="primary.main" sx={{ fontWeight: 600 }}>
                        Selected Agent
                      </Typography>
                    </Box>
                    <Typography variant="body2" sx={{ fontWeight: 500, mb: 0.5 }}>
                      {selectedSubagentDetails.name}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {selectedSubagentDetails.description}
                    </Typography>
                    <Box mt={1}>
                      <Chip 
                        label={selectedSubagentDetails.category} 
                        size="small" 
                        color="primary"
                        variant="outlined"
                      />
                    </Box>
                  </Box>
                )}
              </Box>

              <Divider />

              {/* Optional Assignments */}
              <Box>
                <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
                  Optional Assignments
                </Typography>
                <Stack spacing={2.5}>
                  <Box>
                    <Box display="flex" alignItems="center" gap={1} mb={1}>
                      <TaskIcon fontSize="small" color="primary" />
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        Task Assignment
                      </Typography>
                    </Box>
                    <TextField
                      fullWidth
                      label="Task ID"
                      value={taskId}
                      onChange={(e) => setTaskId(e.target.value)}
                      disabled={creating}
                      placeholder="e.g., 79c045f8-5d41-4050-b0ee-54e1cff9053b"
                      helperText="Link this subchat to a specific task for better organization"
                      sx={{
                        '& .MuiOutlinedInput-root': {
                          borderRadius: 2,
                        },
                      }}
                    />
                  </Box>

                  <Box>
                    <Box display="flex" alignItems="center" gap={1} mb={1}>
                      <TodoIcon fontSize="small" color="success" />
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        TODO Assignment
                      </Typography>
                    </Box>
                    <TextField
                      fullWidth
                      label="TODO ID"
                      value={todoId}
                      onChange={(e) => setTodoId(e.target.value)}
                      disabled={creating}
                      placeholder="e.g., 7ef26573-68b6-46be-bbc3-dca310477625"
                      helperText="Assign this subchat to work on a specific TODO item"
                      sx={{
                        '& .MuiOutlinedInput-root': {
                          borderRadius: 2,
                        },
                      }}
                    />
                  </Box>
                </Stack>
              </Box>
            </Stack>
          )}
        </Box>
      </DialogContent>

      <DialogActions
        sx={{
          p: 3,
          borderTop: '1px solid',
          borderColor: 'divider',
          gap: 1.5,
          flexDirection: isMobile ? 'column-reverse' : 'row',
        }}
      >
        <Button 
          onClick={handleClose} 
          disabled={creating}
          size="large"
          sx={{
            minWidth: isMobile ? '100%' : 120,
            py: 1.5,
            borderRadius: 2,
            textTransform: 'none',
            fontWeight: 500,
          }}
        >
          Cancel
        </Button>
        <Button
          onClick={handleSubmit}
          variant="contained"
          disabled={!selectedSubagent || creating || loading}
          startIcon={creating ? <CircularProgress size={20} color="inherit" /> : <PersonIcon />}
          size="large"
          sx={{
            minWidth: isMobile ? '100%' : 160,
            py: 1.5,
            borderRadius: 2,
            textTransform: 'none',
            fontWeight: 600,
            boxShadow: 2,
            '&:hover': {
              boxShadow: 4,
            },
          }}
        >
          {creating ? 'Creating...' : 'Create Subchat'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default SubchatCreationDialog;