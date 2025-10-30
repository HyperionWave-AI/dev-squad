/**
 * SubchatCreationDialog Component
 *
 * Modal dialog for creating a new subchat with subagent assignment.
 * Allows user to select a subagent and optionally assign task/todo IDs.
 * Optimized for side drawer layout with improved responsive design.
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
  useTheme,
  useMediaQuery,
  Paper,
  Fade,
  Slide,
} from '@mui/material';
import {
  Close as CloseIcon,
  Person as PersonIcon,
  Assignment as TaskIcon,
  CheckCircle as TodoIcon,
  Info as InfoIcon,
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

      // Validate that sessionId exists before showing success
      if (!subchat.sessionId) {
        throw new Error('Subchat created but session ID is missing. Please refresh and try again.');
      }

      setSuccess(true);
      setTimeout(() => {
        // Pass session ID (not subchat ID) for navigation
        onSubchatCreated(subchat.sessionId);
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

  // Optimized styles for side drawer layout
  const getDialogStyles = () => ({
    paper: {
      borderRadius: isMobile ? 0 : 2,
      maxHeight: isMobile ? '100vh' : '85vh',
      minHeight: isMobile ? '100vh' : '500px',
      margin: isMobile ? 0 : 2,
      width: isMobile ? '100%' : 'auto',
      maxWidth: isMobile ? '100%' : '600px',
      // Ensure proper positioning in side drawer
      position: 'relative',
      overflow: 'hidden',
    },
    title: {
      padding: theme.spacing(2, 3),
      borderBottom: `1px solid ${theme.palette.divider}`,
      minHeight: 64,
    },
    content: {
      padding: 0,
      overflow: 'auto',
      flex: 1,
    },
    actions: {
      padding: theme.spacing(2, 3),
      borderTop: `1px solid ${theme.palette.divider}`,
      gap: theme.spacing(1),
      minHeight: 72,
    },
  });

  const styles = getDialogStyles();

  return (
    <Dialog 
      open={open} 
      onClose={handleClose} 
      maxWidth="xs" 
      fullWidth
      fullScreen={isMobile}
      TransitionComponent={isMobile ? Slide : Fade}
      TransitionProps={isMobile ? { direction: 'up' } as any : undefined}
      // Enhanced dialog container positioning for side drawer
      sx={{
        '& .MuiDialog-container': {
          alignItems: isMobile ? 'stretch' : 'flex-start',
          justifyContent: isMobile ? 'stretch' : 'center',
          paddingTop: isMobile ? 0 : 2,
        },
      }}
      PaperProps={{
        sx: styles.paper,
      }}
      // Improved backdrop for side drawer
      BackdropProps={{
        sx: {
          backgroundColor: 'rgba(0, 0, 0, 0.3)',
          backdropFilter: 'blur(2px)',
        },
      }}
    >
      {/* Streamlined Dialog Title */}
      <DialogTitle sx={styles.title}>
        <Box display="flex" justifyContent="space-between" alignItems="center">
          <Box display="flex" alignItems="center" gap={1.5}>
            <Box
              sx={{
                p: 1,
                borderRadius: 1.5,
                backgroundColor: 'primary.50',
                color: 'primary.main',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <PersonIcon sx={{ fontSize: '1.25rem' }} />
            </Box>
            <Box>
              <Typography 
                variant="h6" 
                component="div" 
                sx={{ 
                  fontWeight: 600,
                  fontSize: isMobile ? '1.125rem' : '1.25rem',
                  lineHeight: 1.2,
                }}
              >
                Create Subchat
              </Typography>
              <Typography 
                variant="body2" 
                color="text.secondary"
                sx={{ 
                  fontSize: '0.875rem',
                  mt: 0.5,
                }}
              >
                Delegate to a specialist agent
              </Typography>
            </Box>
          </Box>
          <IconButton
            onClick={handleClose}
            disabled={creating}
            size="small"
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
        </Box>
      </DialogTitle>

      <DialogContent sx={styles.content}>
        <Box sx={{ p: isMobile ? 2 : 3 }}>
          {loading && (
            <Box 
              display="flex" 
              justifyContent="center"
              py={6}
              gap={2}
            >
              <CircularProgress size={40} thickness={4} />
              <Typography 
                variant="body2" 
                color="text.secondary"
                sx={{ fontSize: '0.875rem' }}
              >
                Loading agents...
              </Typography>
            </Box>
          )}

          {!loading && (
            <Stack spacing={isMobile ? 2.5 : 3}>
              {/* Error Alert */}
              {error && (
                <Alert
                  severity="error"
                  sx={{
                    borderRadius: 1.5,
                    '& .MuiAlert-message': {
                      fontSize: '0.875rem',
                    },
                  }}
                >
                  {error}
                </Alert>
              )}

              {/* Success Alert */}
              {success && (
                <Alert 
                  severity="success" 
                  sx={{ 
                    borderRadius: 1.5,
                    '& .MuiAlert-message': {
                      fontSize: '0.875rem',
                    },
                  }}
                >
                  Subchat created successfully! Redirecting...
                </Alert>
              )}

              {/* Subagent Selection */}
              <Box>
                <Typography 
                  variant="subtitle2" 
                  sx={{ 
                    mb: 1.5, 
                    fontWeight: 600,
                    fontSize: '0.875rem',
                    color: 'text.primary',
                  }}
                >
                  Select Subagent *
                </Typography>
                <TextField
                  select
                  fullWidth
                  value={selectedSubagent}
                  onChange={(e) => setSelectedSubagent(e.target.value)}
                  placeholder="Choose a specialist agent"
                  disabled={creating}
                  size="medium"
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      borderRadius: 1.5,
                      fontSize: '0.875rem',
                    },
                    '& .MuiInputLabel-root': {
                      fontSize: '0.875rem',
                    },
                  }}
                >
                  {Object.entries(groupedSubagents).map(([category, agents]) => [
                    <MenuItem key={category} disabled sx={{ 
                      fontWeight: 600, 
                      fontSize: '0.75rem',
                      textTransform: 'uppercase',
                      color: 'text.secondary',
                      py: 1,
                    }}>
                      {category}
                    </MenuItem>,
                    ...agents.map((agent) => (
                      <MenuItem 
                        key={agent.name} 
                        value={agent.name}
                        sx={{ 
                          pl: 3,
                          fontSize: '0.875rem',
                          py: 1.5,
                        }}
                      >
                        <Box>
                          <Typography variant="body2" sx={{ fontWeight: 500, fontSize: '0.875rem' }}>
                            {agent.name}
                          </Typography>
                          <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.75rem' }}>
                            {agent.description}
                          </Typography>
                        </Box>
                      </MenuItem>
                    ))
                  ]).flat()}
                </TextField>
              </Box>

              {/* Selected Subagent Details */}
              {selectedSubagentDetails && (
                <Paper 
                  sx={{ 
                    p: 2.5, 
                    backgroundColor: 'primary.50',
                    borderRadius: 1.5,
                    border: `1px solid ${theme.palette.primary.light}`,
                  }}
                >
                  <Box display="flex" alignItems="flex-start" gap={1.5}>
                    <Box
                      sx={{
                        p: 1,
                        borderRadius: 1,
                        backgroundColor: 'primary.main',
                        color: 'primary.contrastText',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        minWidth: 32,
                        height: 32,
                      }}
                    >
                      <PersonIcon sx={{ fontSize: '1rem' }} />
                    </Box>
                    <Box flex={1}>
                      <Typography variant="subtitle2" sx={{ fontWeight: 600, fontSize: '0.875rem', mb: 0.5 }}>
                        {selectedSubagentDetails.name}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={{ fontSize: '0.75rem', mb: 1 }}>
                        {selectedSubagentDetails.description}
                      </Typography>
                      <Chip 
                        label={selectedSubagentDetails.category} 
                        size="small" 
                        sx={{ 
                          fontSize: '0.6875rem',
                          height: 24,
                          backgroundColor: 'primary.main',
                          color: 'primary.contrastText',
                        }} 
                      />
                    </Box>
                  </Box>
                </Paper>
              )}

              {/* Optional Task/Todo IDs */}
              <Stack spacing={2}>
                <Typography 
                  variant="subtitle2" 
                  sx={{ 
                    fontWeight: 600,
                    fontSize: '0.875rem',
                    color: 'text.primary',
                  }}
                >
                  Optional Context
                </Typography>
                
                <TextField
                  label="Task ID"
                  value={taskId}
                  onChange={(e) => setTaskId(e.target.value)}
                  placeholder="Enter task ID to associate"
                  disabled={creating}
                  size="medium"
                  InputProps={{
                    startAdornment: (
                      <Box sx={{ mr: 1, color: 'text.secondary' }}>
                        <TaskIcon sx={{ fontSize: '1rem' }} />
                      </Box>
                    ),
                  }}
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      borderRadius: 1.5,
                      fontSize: '0.875rem',
                    },
                    '& .MuiInputLabel-root': {
                      fontSize: '0.875rem',
                    },
                  }}
                />
                
                <TextField
                  label="Todo ID"
                  value={todoId}
                  onChange={(e) => setTodoId(e.target.value)}
                  placeholder="Enter todo ID to associate"
                  disabled={creating}
                  size="medium"
                  InputProps={{
                    startAdornment: (
                      <Box sx={{ mr: 1, color: 'text.secondary' }}>
                        <TodoIcon sx={{ fontSize: '1rem' }} />
                      </Box>
                    ),
                  }}
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      borderRadius: 1.5,
                      fontSize: '0.875rem',
                    },
                    '& .MuiInputLabel-root': {
                      fontSize: '0.875rem',
                    },
                  }}
                />
              </Stack>

              {/* Info Box */}
              <Paper 
                sx={{ 
                  p: 2, 
                  backgroundColor: 'info.50',
                  borderRadius: 1.5,
                  border: `1px solid ${theme.palette.info.light}`,
                }}
              >
                <Box display="flex" gap={1.5}>
                  <InfoIcon sx={{ color: 'info.main', fontSize: '1rem', mt: 0.25 }} />
                  <Typography variant="body2" color="info.dark" sx={{ fontSize: '0.75rem', lineHeight: 1.4 }}>
                    The subchat will inherit the context from this conversation and allow the selected agent to work independently on the delegated task.
                  </Typography>
                </Box>
              </Paper>
            </Stack>
          )}
        </Box>
      </DialogContent>

      {/* Improved Dialog Actions */}
      <DialogActions sx={styles.actions}>
        <Button
          onClick={handleClose}
          disabled={creating}
          variant="outlined"
          size="medium"
          sx={{
            minWidth: 100,
            minHeight: 40,
            borderRadius: 1.5,
            textTransform: 'none',
            fontWeight: 500,
            fontSize: '0.875rem',
            transition: 'all 0.2s ease-in-out',
            '&:hover': {
              borderColor: 'primary.main',
            },
          }}
        >
          Cancel
        </Button>
        <Button
          onClick={handleSubmit}
          disabled={!selectedSubagent || creating || loading}
          variant="contained"
          size="medium"
          sx={{
            minWidth: 120,
            minHeight: 40,
            borderRadius: 1.5,
            textTransform: 'none',
            fontWeight: 600,
            fontSize: '0.875rem',
            position: 'relative',
          }}
        >
          {creating ? (
            <Box display="flex" alignItems="center" gap={1}>
              <CircularProgress 
                size={16} 
                thickness={4}
                sx={{ color: 'inherit' }}
              />
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

export default SubchatCreationDialog;