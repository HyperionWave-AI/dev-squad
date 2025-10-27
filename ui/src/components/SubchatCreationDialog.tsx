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
              </Box>

              {/* Selected Agent Details */}
              {selectedSubagentDetails && (
                <Box
                  sx={{
                    p: 2.5,
                    borderRadius: 2,
                    backgroundColor: 'primary.50',
                    border: '1px solid',
                    borderColor: 'primary.200',
                  }}
                >
                  <Box display="flex" alignItems="center" gap={1.5} mb={1.5}>
                    <Box
                      sx={{
                        p: 0.75,
                        borderRadius: 1.5,
                        backgroundColor: 'primary.main',
                        color: 'white',
                      }}
                    >
                      <PersonIcon fontSize="small" />
                    </Box>
                    <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                      {selectedSubagentDetails.name}
                    </Typography>
                    <Chip 
                      label={selectedSubagentDetails.category} 
                      size="small"
                      sx={{
                        backgroundColor: 'primary.100',
                        color: 'primary.dark',
                        fontWeight: 500,
                      }}
                    />
                  </Box>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    {selectedSubagentDetails.description}
                  </Typography>
                </Box>
              )}

                          />
                        ))}
                      </Stack>
                    </Box>
                  )}
                </Box>
              )}

              <Divider sx={{ my: 2 }} />

              {/* Optional Task/Todo Assignment */}
              <Box>
                <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2 }}>
                  Optional Context (Advanced)
                </Typography>
                <Stack spacing={2}>
                  <TextField
                    fullWidth
                    label="Task ID"
                    value={taskId}
                    onChange={(e) => setTaskId(e.target.value)}
                    disabled={creating}
                    sx={{
                      '& .MuiOutlinedInput-root': {
                        borderRadius: 2,
                      },
                    }}
                    helperText="Link this subchat to a specific task (optional)"
                    InputProps={{
                      startAdornment: (
                        <Box sx={{ mr: 1, color: 'text.secondary' }}>
                          <TaskIcon fontSize="small" />
                        </Box>
                      ),
                    }}
                  />
                  <TextField
                    fullWidth
                    label="Todo ID"
                    value={todoId}
                    onChange={(e) => setTodoId(e.target.value)}
                    disabled={creating}
                    sx={{
                      '& .MuiOutlinedInput-root': {
                        borderRadius: 2,
                      },
                    }}
                    helperText="Link this subchat to a specific todo item (optional)"
                    InputProps={{
                      startAdornment: (
                        <Box sx={{ mr: 1, color: 'text.secondary' }}>
                          <TodoIcon fontSize="small" />
                        </Box>
                      ),
                    }}
                  />
                </Stack>
              </Box>

              {/* Placeholder Buttons Section */}
              <Box sx={{ 
                p: 2.5, 
                borderRadius: 2,
                backgroundColor: 'grey.50',
                border: '1px solid',
                borderColor: 'grey.200'
              }}>
                <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 2, color: 'text.secondary' }}>
                  Quick Actions
                </Typography>
                <Stack direction="row" spacing={2} justifyContent="flex-start" flexWrap="wrap" useFlexGap>
                  <Button
                    variant="outlined"
                    size="small"
                    sx={{ 
                      borderRadius: 2, 
                      textTransform: 'none',
                      minWidth: 100,
                      '&:hover': {
                        backgroundColor: 'primary.50',
                      }
                    }}
                  >
                    Action 1
                  </Button>
                  <Button
                    variant="outlined"
                    size="small"
                    sx={{ 
                      borderRadius: 2, 
                      textTransform: 'none',
                      minWidth: 100,
                      '&:hover': {
                        backgroundColor: 'primary.50',
                      }
                    }}
                  >
                    Action 2
                  </Button>
                  <Button
                    variant="outlined"
                    size="small"
                    sx={{ 
                      borderRadius: 2, 
                      textTransform: 'none',
                      minWidth: 100,
                      '&:hover': {
                        backgroundColor: 'primary.50',
                      }
                    }}
                  >
                    Action 3
                  </Button>
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
          gap: 2,
        }}
      >
        <Button
          onClick={handleClose}
          disabled={creating}
          sx={{
            borderRadius: 2,
            textTransform: 'none',
            px: 3,
          }}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          variant="contained"
          onClick={handleSubmit}
          disabled={!selectedSubagent || creating}
          sx={{
            borderRadius: 2,
            textTransform: 'none',
            px: 4,
            fontWeight: 600,
          }}
        >
          {creating ? (
            <Box display="flex" alignItems="center" gap={1}>
              <CircularProgress size={16} color="inherit" />
              Creating...
            </Box>
          ) : (
            'Create Subchat'
          )}
        </Button>
      </DialogActions>
    </Dialog>
  );
};