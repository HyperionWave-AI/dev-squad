/**
 * SubchatCreationDialog Component
 *
 * Modal dialog for creating a new subchat with subagent assignment.
 * Allows user to select a subagent and optionally assign task/todo IDs.
 * Fully responsive design optimized for mobile, tablet, and desktop viewports.
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

  // Responsive typography and spacing
  const getResponsiveStyles = () => ({
    dialogTitle: {
      fontSize: {
        xs: '1.125rem', // 18px on mobile
        sm: '1.25rem',  // 20px on tablet
        md: '1.5rem',   // 24px on desktop
      },
    },
    subtitle: {
      fontSize: {
        xs: '0.875rem', // 14px on mobile
        sm: '1rem',     // 16px on tablet+
      },
    },
    body: {
      fontSize: {
        xs: '0.875rem', // 14px on mobile
        sm: '1rem',     // 16px on tablet+
      },
    },
    caption: {
      fontSize: {
        xs: '0.75rem',  // 12px on mobile
        sm: '0.875rem', // 14px on tablet+
      },
    },
    buttonMinHeight: {
      xs: 44, // Touch-friendly minimum
      sm: 40,
      md: 36,
    },
    padding: {
      xs: 2,  // 16px on mobile
      sm: 3,  // 24px on tablet
      md: 4,  // 32px on desktop
    },
    spacing: {
      xs: 2,  // 16px on mobile
      sm: 2.5, // 20px on tablet
      md: 3,  // 24px on desktop
    },
  });

  const styles = getResponsiveStyles();

  return (
    <Dialog 
      open={open} 
      onClose={handleClose} 
      maxWidth="md" 
      fullWidth
      fullScreen={isMobile}
      PaperProps={{
        sx: {
          borderRadius: isMobile ? 0 : { xs: 2, sm: 3 },
          maxHeight: isMobile ? '100vh' : '90vh',
          m: isMobile ? 0 : { xs: 1, sm: 2 },
          width: isMobile ? '100%' : 'auto',
        },
      }}
    >
      {/* Enhanced Dialog Title */}
      <DialogTitle
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          pb: 2,
          px: styles.padding,
          pt: styles.padding,
          borderBottom: '1px solid',
          borderColor: 'divider',
          flexDirection: { xs: 'column', sm: 'row' },
          gap: { xs: 2, sm: 0 },
          alignItems: { xs: 'flex-start', sm: 'center' },
        }}
      >
        <Box 
          display="flex" 
          alignItems="center" 
          gap={1.5}
          sx={{ 
            width: { xs: '100%', sm: 'auto' },
            justifyContent: { xs: 'space-between', sm: 'flex-start' },
          }}
        >
          <Box display="flex" alignItems="center" gap={1.5}>
            <Box
              sx={{
                p: { xs: 0.75, sm: 1 },
                borderRadius: 2,
                backgroundColor: 'primary.50',
                color: 'primary.main',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <PersonIcon sx={{ fontSize: { xs: '1.25rem', sm: '1.5rem' } }} />
            </Box>
            <Box>
              <Typography 
                variant="h6" 
                component="div" 
                sx={{ 
                  fontWeight: 600,
                  fontSize: styles.dialogTitle.fontSize,
                  lineHeight: 1.2,
                }}
              >
                Create New Subchat
              </Typography>
              <Typography 
                variant="body2" 
                color="text.secondary"
                sx={{ 
                  fontSize: styles.caption.fontSize,
                  display: { xs: 'none', sm: 'block' },
                }}
              >
                Delegate work to a specialist agent
              </Typography>
            </Box>
          </Box>
          {isMobile && (
            <IconButton
              onClick={handleClose}
              disabled={creating}
              sx={{
                color: 'text.secondary',
                minWidth: 44,
                minHeight: 44,
                '&:hover': {
                  backgroundColor: 'error.50',
                  color: 'error.main',
                },
              }}
            >
              <CloseIcon />
            </IconButton>
          )}
        </Box>
        {!isMobile && (
          <IconButton
            onClick={handleClose}
            disabled={creating}
            sx={{
              color: 'text.secondary',
              minWidth: 44,
              minHeight: 44,
              '&:hover': {
                backgroundColor: 'error.50',
                color: 'error.main',
              },
            }}
          >
            <CloseIcon />
          </IconButton>
        )}
        {isMobile && (
          <Typography 
            variant="body2" 
            color="text.secondary"
            sx={{ 
              fontSize: styles.caption.fontSize,
              width: '100%',
              mt: 1,
            }}
          >
            Delegate work to a specialist agent
          </Typography>
        )}
      </DialogTitle>

      <DialogContent sx={{ p: 0, overflow: 'hidden' }}>
        <Box sx={{ p: styles.padding }}>
          {loading && (
            <Box 
              display="flex" 
              flexDirection="column"
              alignItems="center" 
              justifyContent="center"
              py={{ xs: 4, sm: 6 }}
              gap={2}
            >
              <CircularProgress size={44} />
              <Typography 
                variant="body2" 
                color="text.secondary"
                sx={{ fontSize: styles.caption.fontSize }}
              >
                Loading available agents...
              </Typography>
            </Box>
          )}

          {!loading && error && (
            <Alert 
              severity="error" 
              sx={{ 
                mb: styles.spacing,
                borderRadius: { xs: 1, sm: 2 },
                fontSize: styles.caption.fontSize,
                '& .MuiAlert-message': {
                  fontSize: styles.body.fontSize,
                },
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
                mb: styles.spacing,
                borderRadius: { xs: 1, sm: 2 },
              }}
            >
              <Box>
                <Typography 
                  variant="subtitle2" 
                  sx={{ 
                    fontWeight: 600,
                    fontSize: styles.body.fontSize,
                  }}
                >
                  Subchat created successfully!
                </Typography>
                <Typography 
                  variant="body2"
                  sx={{ fontSize: styles.caption.fontSize }}
                >
                  Redirecting to the new subchat...
                </Typography>
              </Box>
            </Alert>
          )}

          {!loading && (
            <Stack spacing={styles.spacing}>
              {/* Subagent Selection */}
              <Box>
                <Typography 
                  variant="subtitle1" 
                  sx={{ 
                    fontWeight: 600, 
                    mb: 2,
                    fontSize: styles.subtitle.fontSize,
                  }}
                >
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
                      borderRadius: { xs: 1, sm: 2 },
                      minHeight: styles.buttonMinHeight,
                    },
                    '& .MuiInputLabel-root': {
                      fontSize: styles.body.fontSize,
                    },
                    '& .MuiSelect-select': {
                      fontSize: styles.body.fontSize,
                    },
                  }}
                  helperText="Each agent specializes in different areas of development"
                  FormHelperTextProps={{
                    sx: { fontSize: styles.caption.fontSize }
                  }}
                >
                  {Object.entries(groupedSubagents).map(([category, agents]) => [
                    <MenuItem key={category} disabled sx={{ py: 1 }}>
                      <Typography 
                        variant="subtitle2" 
                        color="primary.main"
                        sx={{ 
                          fontWeight: 600, 
                          textTransform: 'uppercase', 
                          fontSize: { xs: '0.6875rem', sm: '0.75rem' },
                        }}
                      >
                        {category}
                      </Typography>
                    </MenuItem>,
                    ...agents.map((agent) => (
                      <MenuItem 
                        key={agent.name} 
                        value={agent.name}
                        sx={{ 
                          py: { xs: 1.5, sm: 1.25 },
                          pl: { xs: 2.5, sm: 3 },
                          minHeight: styles.buttonMinHeight,
                          '&:hover': {
                            backgroundColor: 'primary.50',
                          },
                        }}
                      >
                        <Box sx={{ width: '100%' }}>
                          <Typography 
                            variant="body1" 
                            sx={{ 
                              fontWeight: 500, 
                              mb: 0.5,
                              fontSize: styles.body.fontSize,
                            }}
                          >
                            {agent.name}
                          </Typography>
                          <Typography 
                            variant="caption" 
                            color="text.secondary" 
                            display="block"
                            sx={{ 
                              fontSize: styles.caption.fontSize,
                              lineHeight: 1.3,
                            }}
                          >
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
                <Paper
                  variant="outlined"
                  sx={{
                    p: styles.padding,
                    backgroundColor: 'primary.50',
                    borderColor: 'primary.200',
                    borderRadius: { xs: 1, sm: 2 },
                  }}
                >
                  <Box display="flex" alignItems="flex-start" gap={2}>
                    <Box
                      sx={{
                        p: 1,
                        borderRadius: 1.5,
                        backgroundColor: 'primary.main',
                        color: 'primary.contrastText',
                        minWidth: 'auto',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      <PersonIcon sx={{ fontSize: { xs: '1rem', sm: '1.25rem' } }} />
                    </Box>
                    <Box sx={{ flex: 1, minWidth: 0 }}>
                      <Typography 
                        variant="subtitle2" 
                        sx={{ 
                          fontWeight: 600, 
                          color: 'primary.main',
                          fontSize: styles.body.fontSize,
                        }}
                      >
                        {selectedSubagentDetails.name}
                      </Typography>
                      <Typography 
                        variant="body2" 
                        color="text.secondary" 
                        sx={{ 
                          mt: 0.5,
                          fontSize: styles.caption.fontSize,
                          lineHeight: 1.4,
                        }}
                      >
                        {selectedSubagentDetails.description}
                      </Typography>
                      {selectedSubagentDetails.tools && selectedSubagentDetails.tools.length > 0 && (
                        <Box sx={{ mt: 1.5, display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                          {selectedSubagentDetails.tools.slice(0, isMobile ? 3 : 5).map((tool, index) => (
                            <Chip
                              key={index}
                              label={tool}
                              size="small"
                              sx={{
                                backgroundColor: 'primary.100',
                                color: 'primary.dark',
                                fontSize: { xs: '0.6875rem', sm: '0.75rem' },
                                height: { xs: 24, sm: 28 },
                                '& .MuiChip-label': {
                                  px: { xs: 1, sm: 1.5 },
                                },
                              }}
                            />
                          ))}
                          {selectedSubagentDetails.tools.length > (isMobile ? 3 : 5) && (
                            <Chip
                              label={`+${selectedSubagentDetails.tools.length - (isMobile ? 3 : 5)} more`}
                              size="small"
                              sx={{
                                backgroundColor: 'grey.200',
                                color: 'text.secondary',
                                fontSize: { xs: '0.6875rem', sm: '0.75rem' },
                                height: { xs: 24, sm: 28 },
                              }}
                            />
                          )}
                        </Box>
                      )}
                    </Box>
                  </Box>
                </Paper>
              )}

              {/* Optional Task/TODO Assignment */}
              <Box>
                <Typography 
                  variant="subtitle1" 
                  sx={{ 
                    fontWeight: 600, 
                    mb: 2,
                    fontSize: styles.subtitle.fontSize,
                  }}
                >
                  Optional Assignment
                </Typography>
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                  <TextField
                    fullWidth
                    label="Task ID"
                    value={taskId}
                    onChange={(e) => setTaskId(e.target.value)}
                    disabled={creating}
                    placeholder="e.g., task-123"
                    InputProps={{
                      startAdornment: (
                        <Box sx={{ mr: 1, display: 'flex', alignItems: 'center' }}>
                          <TaskIcon sx={{ color: 'text.secondary', fontSize: '1.25rem' }} />
                        </Box>
                      ),
                    }}
                    sx={{
                      '& .MuiOutlinedInput-root': {
                        borderRadius: { xs: 1, sm: 2 },
                        minHeight: styles.buttonMinHeight,
                      },
                      '& .MuiInputLabel-root': {
                        fontSize: styles.body.fontSize,
                      },
                      '& .MuiOutlinedInput-input': {
                        fontSize: styles.body.fontSize,
                      },
                    }}
                    helperText="Link to specific task"
                    FormHelperTextProps={{
                      sx: { fontSize: styles.caption.fontSize }
                    }}
                  />
                  <TextField
                    fullWidth
                    label="TODO ID"
                    value={todoId}
                    onChange={(e) => setTodoId(e.target.value)}
                    disabled={creating}
                    placeholder="e.g., todo-456"
                    InputProps={{
                      startAdornment: (
                        <Box sx={{ mr: 1, display: 'flex', alignItems: 'center' }}>
                          <TodoIcon sx={{ color: 'text.secondary', fontSize: '1.25rem' }} />
                        </Box>
                      ),
                    }}
                    sx={{
                      '& .MuiOutlinedInput-root': {
                        borderRadius: { xs: 1, sm: 2 },
                        minHeight: styles.buttonMinHeight,
                      },
                      '& .MuiInputLabel-root': {
                        fontSize: styles.body.fontSize,
                      },
                      '& .MuiOutlinedInput-input': {
                        fontSize: styles.body.fontSize,
                      },
                    }}
                    helperText="Link to specific TODO item"
                    FormHelperTextProps={{
                      sx: { fontSize: styles.caption.fontSize }
                    }}
                  />
                </Stack>
              </Box>

              {/* Info Box */}
              <Paper
                variant="outlined"
                sx={{
                  p: styles.padding,
                  backgroundColor: 'info.50',
                  borderColor: 'info.200',
                  borderRadius: { xs: 1, sm: 2 },
                }}
              >
                <Box display="flex" gap={2}>
                  <InfoIcon sx={{ color: 'info.main', fontSize: '1.25rem', mt: 0.25 }} />
                  <Box>
                    <Typography 
                      variant="body2" 
                      color="info.dark"
                      sx={{ 
                        fontSize: styles.caption.fontSize,
                        lineHeight: 1.4,
                      }}
                    >
                      The subchat will inherit the context from this conversation and allow the selected agent to work independently on the assigned task.
                    </Typography>
                  </Box>
                </Box>
              </Paper>
            </Stack>
          )}
        </Box>
      </DialogContent>

      <DialogActions
        sx={{
          p: styles.padding,
          pt: 2,
          borderTop: '1px solid',
          borderColor: 'divider',
          flexDirection: { xs: 'column-reverse', sm: 'row' },
          gap: { xs: 1.5, sm: 1 },
          '& > :not(:first-of-type)': {
            ml: { xs: 0, sm: 1 },
          },
        }}
      >
        <Button
          onClick={handleClose}
          disabled={creating}
          sx={{
            minHeight: styles.buttonMinHeight,
            px: { xs: 3, sm: 2.5 },
            borderRadius: { xs: 1, sm: 1.5 },
            textTransform: 'none',
            fontSize: styles.body.fontSize,
            width: { xs: '100%', sm: 'auto' },
            order: { xs: 2, sm: 1 },
          }}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          variant="contained"
          onClick={handleSubmit}
          disabled={!selectedSubagent || creating || loading}
          startIcon={creating ? <CircularProgress size={16} /> : <PersonIcon />}
          sx={{
            minHeight: styles.buttonMinHeight,
            px: { xs: 3, sm: 2.5 },
            borderRadius: { xs: 1, sm: 1.5 },
            textTransform: 'none',
            fontWeight: 600,
            fontSize: styles.body.fontSize,
            width: { xs: '100%', sm: 'auto' },
            order: { xs: 1, sm: 2 },
            // Touch feedback on mobile
            '&:active': {
              transform: isMobile ? 'scale(0.98)' : 'none',
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