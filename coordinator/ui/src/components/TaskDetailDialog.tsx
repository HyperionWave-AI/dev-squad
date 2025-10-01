import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  Chip,
  Divider,
  IconButton,
  Paper,
  LinearProgress,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
} from '@mui/material';
import {
  Close,
  AccessTime,
  Person,
  CheckCircle,
  Schedule,
  Block,
  ExpandMore,
  RadioButtonUnchecked,
  Circle,
  SmartToy,
} from '@mui/icons-material';
import ReactMarkdown from 'react-markdown';
import type { HumanTask, AgentTask, FlattenedTask, Priority, TaskStatus, TodoStatus } from '../types/coordinator';
import { mcpClient } from '../services/mcpClient';

interface TaskDetailDialogProps {
  task: FlattenedTask | null;
  open: boolean;
  onClose: () => void;
}

const getPriorityColor = (priority: Priority): 'default' | 'primary' | 'warning' | 'error' => {
  switch (priority) {
    case 'urgent':
      return 'error';
    case 'high':
      return 'warning';
    case 'medium':
      return 'primary';
    case 'low':
    default:
      return 'default';
  }
};

const getStatusIcon = (status: TaskStatus) => {
  switch (status) {
    case 'completed':
      return <CheckCircle fontSize="small" />;
    case 'in_progress':
      return <Schedule fontSize="small" />;
    case 'blocked':
      return <Block fontSize="small" />;
    case 'pending':
    default:
      return <AccessTime fontSize="small" />;
  }
};

const getStatusColor = (status: TaskStatus): string => {
  switch (status) {
    case 'completed':
      return '#16a34a';
    case 'in_progress':
      return '#2563eb';
    case 'blocked':
      return '#dc2626';
    case 'pending':
    default:
      return '#64748b';
  }
};

const getTodoStatusIcon = (status: TodoStatus) => {
  switch (status) {
    case 'completed':
      return <CheckCircle fontSize="small" sx={{ color: '#16a34a' }} />;
    case 'in_progress':
      return <Circle fontSize="small" sx={{ color: '#2563eb' }} />;
    case 'pending':
    default:
      return <RadioButtonUnchecked fontSize="small" sx={{ color: '#64748b' }} />;
  }
};

export function TaskDetailDialog({ task, open, onClose }: TaskDetailDialogProps) {
  const [agentTasks, setAgentTasks] = useState<AgentTask[]>([]);
  const [loading, setLoading] = useState(false);
  const [parentTask, setParentTask] = useState<HumanTask | null>(null);

  const isAgentTask = task?.taskType === 'agent';

  useEffect(() => {
    if (open && task) {
      if (isAgentTask) {
        loadParentTask();
      } else {
        loadAgentTasks();
      }
    }
  }, [open, task, isAgentTask]);

  const loadAgentTasks = async () => {
    if (!task) return;

    try {
      setLoading(true);
      await mcpClient.connect();
      const allAgentTasks = await mcpClient.listAgentTasks();
      const relatedTasks = allAgentTasks.filter(at => at.humanTaskId === task.id);
      setAgentTasks(relatedTasks);
    } catch (error) {
      console.error('Failed to load agent tasks:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadParentTask = async () => {
    if (!task || !task.humanTaskId) return;

    try {
      setLoading(true);
      await mcpClient.connect();
      const parent = await mcpClient.getHumanTask(task.humanTaskId);
      setParentTask(parent);
    } catch (error) {
      console.error('Failed to load parent task:', error);
    } finally {
      setLoading(false);
    }
  };

  if (!task) return null;

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const calculateProgress = (agentTask: AgentTask) => {
    if (!agentTask.todos || agentTask.todos.length === 0) return 0;
    const completed = agentTask.todos.filter(t => t.status === 'completed').length;
    return (completed / agentTask.todos.length) * 100;
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="md"
      fullWidth
      PaperProps={{
        sx: {
          borderRadius: 2,
          maxHeight: '90vh',
        },
      }}
    >
      {/* Dialog Header */}
      <DialogTitle
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'flex-start',
          pb: 2,
        }}
      >
        <Box sx={{ flex: 1, pr: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
            {isAgentTask && <SmartToy color="primary" />}
            <Typography variant="h5" sx={{ fontWeight: 600 }}>
              {task.title || task.role || 'Task Details'}
            </Typography>
          </Box>
          {isAgentTask && task.agentName && (
            <Typography variant="subtitle1" color="primary" sx={{ mb: 1, fontWeight: 500 }}>
              Agent: {task.agentName}
            </Typography>
          )}
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
            {isAgentTask && (
              <Chip
                label="AGENT TASK"
                size="small"
                sx={{
                  backgroundColor: '#e0e7ff',
                  color: '#4f46e5',
                  fontWeight: 600
                }}
              />
            )}
            <Chip
              label={(task.priority || 'medium').toUpperCase()}
              size="small"
              color={getPriorityColor(task.priority || 'medium')}
              sx={{ fontWeight: 600 }}
            />
            <Chip
              icon={getStatusIcon(task.status)}
              label={task.status.replace('_', ' ').toUpperCase()}
              size="small"
              sx={{
                backgroundColor: `${getStatusColor(task.status)}15`,
                color: getStatusColor(task.status),
                fontWeight: 600,
                '& .MuiChip-icon': {
                  color: getStatusColor(task.status),
                },
              }}
            />
            {task.tags && task.tags.length > 0 && task.tags.map((tag: string, idx: number) => (
              <Chip key={idx} label={tag} size="small" variant="outlined" />
            ))}
          </Box>
        </Box>
        <IconButton onClick={onClose} size="small">
          <Close />
        </IconButton>
      </DialogTitle>

      <Divider />

      {/* Dialog Content */}
      <DialogContent sx={{ pt: 3 }}>
        {/* Task Metadata */}
        <Paper
          elevation={0}
          sx={{
            p: 2,
            mb: 3,
            backgroundColor: 'grey.50',
            border: '1px solid',
            borderColor: 'divider',
          }}
        >
          <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 2 }}>
            <Box>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                Created
              </Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <AccessTime fontSize="small" color="action" />
                <Typography variant="body2">{formatDate(task.createdAt)}</Typography>
              </Box>
            </Box>
            <Box>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                Updated
              </Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <AccessTime fontSize="small" color="action" />
                <Typography variant="body2">{task.updatedAt ? formatDate(task.updatedAt) : 'N/A'}</Typography>
              </Box>
            </Box>
            <Box>
              <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                Created By
              </Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                <Person fontSize="small" color="action" />
                <Typography variant="body2">{task.createdBy}</Typography>
              </Box>
            </Box>
            {task.completedAt && (
              <Box>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 0.5 }}>
                  Completed
                </Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                  <CheckCircle fontSize="small" color="success" />
                  <Typography variant="body2">{formatDate(task.completedAt)}</Typography>
                </Box>
              </Box>
            )}
          </Box>
        </Paper>

        {/* Task Description */}
        <Box sx={{ mb: 3 }}>
          <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
            Description
          </Typography>
          <Paper
            elevation={0}
            sx={{
              p: 2,
              backgroundColor: 'white',
              border: '1px solid',
              borderColor: 'divider',
              '& p': { mb: 1 },
              '& pre': {
                backgroundColor: 'grey.100',
                p: 1,
                borderRadius: 1,
                overflow: 'auto',
              },
              '& code': {
                backgroundColor: 'grey.100',
                px: 0.5,
                py: 0.25,
                borderRadius: 0.5,
                fontSize: '0.875em',
              },
            }}
          >
            <ReactMarkdown>{task.description || 'No description available'}</ReactMarkdown>
          </Paper>
        </Box>

        {/* Notes */}
        {task.notes && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              Notes
            </Typography>
            <Paper
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: '#fffbeb',
                border: '1px solid #fde68a',
              }}
            >
              <ReactMarkdown>{task.notes}</ReactMarkdown>
            </Paper>
          </Box>
        )}

        {/* Agent/Todo Task Metadata */}
        {(task.taskType === 'agent' || task.taskType === 'todo') && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              Agent Details
            </Typography>
            <Paper
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: 'white',
                border: '1px solid',
                borderColor: 'divider',
              }}
            >
              {task.agentName && (
                <Box sx={{ mb: 1 }}>
                  <Typography variant="caption" color="text.secondary">Agent Name</Typography>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>🤖 {task.agentName}</Typography>
                </Box>
              )}
              {task.role && (
                <Box sx={{ mb: 1 }}>
                  <Typography variant="caption" color="text.secondary">Role</Typography>
                  <Typography variant="body2">{task.role}</Typography>
                </Box>
              )}
              {task.parentTaskTitle && (
                <Box>
                  <Typography variant="caption" color="text.secondary">Parent Task</Typography>
                  <Typography variant="body2">{task.parentTaskTitle}</Typography>
                </Box>
              )}
            </Paper>
          </Box>
        )}

        {/* Parent Human Task (for agent tasks) */}
        {isAgentTask && parentTask && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              Parent Human Task
            </Typography>
            <Paper
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: '#f0f9ff',
                border: '1px solid #bae6fd',
              }}
            >
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                {parentTask.title}
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                {parentTask.description || parentTask.prompt}
              </Typography>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <Chip
                  label={parentTask.status.replace('_', ' ').toUpperCase()}
                  size="small"
                  sx={{
                    backgroundColor: `${getStatusColor(parentTask.status)}15`,
                    color: getStatusColor(parentTask.status),
                  }}
                />
                <Chip
                  label={`Created ${formatDate(parentTask.createdAt)}`}
                  size="small"
                  variant="outlined"
                />
              </Box>
            </Paper>
          </Box>
        )}

        {/* Agent Tasks (for human tasks) */}
        {!isAgentTask && (
          <Box>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
              <SmartToy color="primary" />
              <Typography variant="h6" sx={{ fontWeight: 600 }}>
                Agent Tasks
              </Typography>
              <Chip label={agentTasks.length} size="small" color="primary" />
            </Box>

            {loading && <LinearProgress sx={{ mb: 2 }} />}

            {agentTasks.length === 0 && !loading && (
              <Paper
                elevation={0}
                sx={{
                  p: 3,
                  textAlign: 'center',
                  backgroundColor: 'grey.50',
                  border: '1px solid',
                  borderColor: 'divider',
                }}
              >
                <Typography color="text.secondary">No agent tasks assigned yet</Typography>
              </Paper>
            )}

          {agentTasks.map((agentTask, idx) => {
            const progress = calculateProgress(agentTask);

            return (
              <Accordion
                key={agentTask.id}
                defaultExpanded={idx === 0}
                sx={{
                  mb: 1,
                  '&:before': { display: 'none' },
                  border: '1px solid',
                  borderColor: 'divider',
                }}
              >
                <AccordionSummary
                  expandIcon={<ExpandMore />}
                  sx={{
                    '& .MuiAccordionSummary-content': {
                      display: 'flex',
                      flexDirection: 'column',
                      gap: 1,
                    },
                  }}
                >
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <SmartToy fontSize="small" color="action" />
                      <Typography sx={{ fontWeight: 600 }}>{agentTask.agentName}</Typography>
                    </Box>
                    <Chip
                      icon={getStatusIcon(agentTask.status)}
                      label={agentTask.status.replace('_', ' ')}
                      size="small"
                      sx={{
                        backgroundColor: `${getStatusColor(agentTask.status)}15`,
                        color: getStatusColor(agentTask.status),
                        '& .MuiChip-icon': {
                          color: getStatusColor(agentTask.status),
                        },
                      }}
                    />
                  </Box>
                  <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
                    {agentTask.role}
                  </Typography>
                  {agentTask.todos && agentTask.todos.length > 0 && (
                    <Box sx={{ width: '100%', mt: 1 }}>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                        <Typography variant="caption" color="text.secondary">
                          Progress
                        </Typography>
                        <Typography variant="caption" color="text.secondary">
                          {agentTask.todos.filter(t => t.status === 'completed').length} / {agentTask.todos.length}
                        </Typography>
                      </Box>
                      <LinearProgress variant="determinate" value={progress} sx={{ borderRadius: 1 }} />
                    </Box>
                  )}
                </AccordionSummary>
                <AccordionDetails>
                  {/* Agent Task Details */}
                  <Box sx={{ mb: 2 }}>
                    <Typography variant="caption" color="text.secondary" display="block" gutterBottom>
                      Created: {formatDate(agentTask.createdAt)}
                    </Typography>
                    {agentTask.notes && (
                      <Paper
                        elevation={0}
                        sx={{
                          p: 1.5,
                          mt: 1,
                          backgroundColor: 'grey.50',
                          border: '1px solid',
                          borderColor: 'divider',
                        }}
                      >
                        <Typography variant="body2">{agentTask.notes}</Typography>
                      </Paper>
                    )}
                  </Box>

                  {/* Todo List */}
                  {agentTask.todos && agentTask.todos.length > 0 && (
                    <Box>
                      <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                        Tasks
                      </Typography>
                      <List dense disablePadding>
                        {agentTask.todos.map((todo) => (
                          <ListItem
                            key={todo.id}
                            sx={{
                              px: 0,
                              py: 0.5,
                              opacity: todo.status === 'completed' ? 0.6 : 1,
                            }}
                          >
                            <ListItemIcon sx={{ minWidth: 32 }}>
                              {getTodoStatusIcon(todo.status)}
                            </ListItemIcon>
                            <ListItemText
                              primary={todo.description}
                              secondary={todo.notes}
                              primaryTypographyProps={{
                                sx: {
                                  textDecoration: todo.status === 'completed' ? 'line-through' : 'none',
                                },
                              }}
                            />
                          </ListItem>
                        ))}
                      </List>
                    </Box>
                  )}
                </AccordionDetails>
              </Accordion>
            );
          })}
        </Box>
        )}
      </DialogContent>

      {/* Dialog Actions */}
      <DialogActions sx={{ px: 3, py: 2 }}>
        <Button onClick={onClose} variant="contained">
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
}