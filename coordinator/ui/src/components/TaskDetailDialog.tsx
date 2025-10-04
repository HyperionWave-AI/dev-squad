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
  Code,
  Functions,
  Storage,
  Lightbulb,
  NoteAdd,
} from '@mui/icons-material';
import ReactMarkdown from 'react-markdown';
import type { HumanTask, AgentTask, FlattenedTask, Priority, TaskStatus, TodoStatus } from '../types/coordinator';
import { mcpClient } from '../services/mcpClient';
import { PromptNotesEditor } from './PromptNotesEditor';
import { TodoPromptNotes } from './TodoPromptNotes';

interface TaskDetailDialogProps {
  task: FlattenedTask | null;
  open: boolean;
  onClose: () => void;
  onTaskUpdate?: () => void;
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

export function TaskDetailDialog({ task, open, onClose, onTaskUpdate }: TaskDetailDialogProps) {
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

  const loadCurrentAgentTask = async () => {
    console.log('[TaskDetailDialog] loadCurrentAgentTask called, taskType:', task?.taskType);
    if (!task || task.taskType !== 'agent') return;

    try {
      setLoading(true);
      await mcpClient.connect();
      const allAgentTasks = await mcpClient.listAgentTasks();
      const updatedTask = allAgentTasks.find(at => at.id === task.id);

      console.log('[TaskDetailDialog] Updated task found:', !!updatedTask, 'has notes:', updatedTask?.humanPromptNotes);
      if (updatedTask && onTaskUpdate) {
        console.log('[TaskDetailDialog] Calling onTaskUpdate');
        // Trigger parent to refresh with the updated task data
        onTaskUpdate();
      }
    } catch (error) {
      console.error('Failed to reload agent task:', error);
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

  const handleRefresh = () => {
    if (isAgentTask) {
      // Reload the agent task itself to get updated notes
      loadCurrentAgentTask();
      // Also load parent task for context
      loadParentTask();
    } else {
      loadAgentTasks();
      // Notify parent component to refresh task data
      if (onTaskUpdate) {
        onTaskUpdate();
      }
    }
  };

  const handleSaveTaskNotes = async (agentTaskId: string, notes: string) => {
    const agentTask = agentTasks.find(at => at.id === agentTaskId);
    if (!agentTask) return;

    try {
      if (agentTask.humanPromptNotes) {
        await mcpClient.updateTaskPromptNotes(agentTaskId, notes);
      } else {
        await mcpClient.addTaskPromptNotes(agentTaskId, notes);
      }
      handleRefresh();
    } catch (error) {
      console.error('Failed to save task notes:', error);
      throw error;
    }
  };

  const handleClearTaskNotes = async (agentTaskId: string) => {
    try {
      await mcpClient.clearTaskPromptNotes(agentTaskId);
      handleRefresh();
    } catch (error) {
      console.error('Failed to clear task notes:', error);
      throw error;
    }
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

        {/* Human Guidance Notes - for agent tasks viewed directly */}
        {isAgentTask && (
          <Box sx={{ mb: 3 }}>
            <PromptNotesEditor
              notes={task.humanPromptNotes}
              notesAddedAt={task.humanPromptNotesAddedAt}
              isEditable={task.status === 'pending'}
              onSave={async (notes) => {
                await mcpClient.connect();
                if (task.humanPromptNotes) {
                  await mcpClient.updateTaskPromptNotes(task.id, notes);
                } else {
                  await mcpClient.addTaskPromptNotes(task.id, notes);
                }
                handleRefresh();
              }}
              onClear={async () => {
                await mcpClient.connect();
                await mcpClient.clearTaskPromptNotes(task.id);
                handleRefresh();
              }}
              placeholder="Add human guidance notes to help the agent understand requirements, constraints, or context..."
            />
          </Box>
        )}

        {/* Context Summary - for agent tasks */}
        {isAgentTask && task.contextSummary && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
              <Lightbulb color="success" />
              Context Summary
            </Typography>
            <Paper
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: '#f0fdf4',
                border: '1px solid #86efac',
              }}
            >
              <ReactMarkdown>{task.contextSummary}</ReactMarkdown>
            </Paper>
          </Box>
        )}
        {!isAgentTask && agentTasks.map((agentTask) => agentTask.contextSummary && (
          <Box key={`context-${agentTask.id}`} sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
              <Lightbulb color="success" />
              Context Summary ({agentTask.agentName})
            </Typography>
            <Paper
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: '#f0fdf4',
                border: '1px solid #86efac',
              }}
            >
              <ReactMarkdown>{agentTask.contextSummary}</ReactMarkdown>
            </Paper>
          </Box>
        ))}

        {/* Files Modified - for agent tasks */}
        {isAgentTask && task.filesModified && task.filesModified.length > 0 && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
              <Code color="primary" />
              Files to Modify
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
              <List dense disablePadding>
                {task.filesModified.map((filePath, idx) => (
                  <ListItem key={idx} sx={{ px: 0, py: 0.5 }}>
                    <ListItemIcon sx={{ minWidth: 32 }}>
                      <Code fontSize="small" color="action" />
                    </ListItemIcon>
                    <ListItemText
                      primary={filePath}
                      primaryTypographyProps={{
                        sx: { fontFamily: 'monospace', fontSize: '0.875rem' }
                      }}
                    />
                  </ListItem>
                ))}
              </List>
            </Paper>
          </Box>
        )}
        {!isAgentTask && agentTasks.map((agentTask) => agentTask.filesModified && agentTask.filesModified.length > 0 && (
          <Box key={`files-${agentTask.id}`} sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
              <Code color="primary" />
              Files to Modify ({agentTask.agentName})
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
              <List dense disablePadding>
                {agentTask.filesModified.map((filePath, idx) => (
                  <ListItem key={idx} sx={{ px: 0, py: 0.5 }}>
                    <ListItemIcon sx={{ minWidth: 32 }}>
                      <Code fontSize="small" color="action" />
                    </ListItemIcon>
                    <ListItemText
                      primary={filePath}
                      primaryTypographyProps={{
                        sx: { fontFamily: 'monospace', fontSize: '0.875rem' }
                      }}
                    />
                  </ListItem>
                ))}
              </List>
            </Paper>
          </Box>
        ))}

        {/* Qdrant Collections - for agent tasks */}
        {isAgentTask && task.qdrantCollections && task.qdrantCollections.length > 0 && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
              <Storage color="warning" />
              Suggested Qdrant Collections
            </Typography>
            <Paper
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: '#fefce8',
                border: '1px solid #fde047',
              }}
            >
              <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 1 }}>
                {task.qdrantCollections.map((collection, idx) => (
                  <Chip
                    key={idx}
                    label={collection}
                    size="small"
                    icon={<Storage fontSize="small" />}
                    sx={{
                      backgroundColor: 'white',
                      border: '1px solid #fde047',
                    }}
                  />
                ))}
              </Box>
              <Typography variant="caption" color="text.secondary">
                💡 Query these collections only if you need specific technical patterns
              </Typography>
            </Paper>
          </Box>
        )}
        {!isAgentTask && agentTasks.map((agentTask) => agentTask.qdrantCollections && agentTask.qdrantCollections.length > 0 && (
          <Box key={`qdrant-${agentTask.id}`} sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
              <Storage color="warning" />
              Suggested Qdrant Collections ({agentTask.agentName})
            </Typography>
            <Paper
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: '#fefce8',
                border: '1px solid #fde047',
              }}
            >
              <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', mb: 1 }}>
                {agentTask.qdrantCollections.map((collection, idx) => (
                  <Chip
                    key={idx}
                    label={collection}
                    size="small"
                    icon={<Storage fontSize="small" />}
                    sx={{
                      backgroundColor: 'white',
                      border: '1px solid #fde047',
                    }}
                  />
                ))}
              </Box>
              <Typography variant="caption" color="text.secondary">
                💡 Query these collections only if you need specific technical patterns
              </Typography>
            </Paper>
          </Box>
        ))}

        {/* Prior Work Summary - for agent tasks */}
        {isAgentTask && task.priorWorkSummary && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
              <SmartToy color="secondary" />
              Prior Work Summary
            </Typography>
            <Paper
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: '#ede9fe',
                border: '1px solid #c4b5fd',
              }}
            >
              <ReactMarkdown>{task.priorWorkSummary}</ReactMarkdown>
            </Paper>
          </Box>
        )}
        {!isAgentTask && agentTasks.map((agentTask) => agentTask.priorWorkSummary && (
          <Box key={`prior-${agentTask.id}`} sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
              <SmartToy color="secondary" />
              Prior Work Summary ({agentTask.agentName})
            </Typography>
            <Paper
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: '#ede9fe',
                border: '1px solid #c4b5fd',
              }}
            >
              <ReactMarkdown>{agentTask.priorWorkSummary}</ReactMarkdown>
            </Paper>
          </Box>
        ))}

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

        {/* TODOs - for agent tasks viewed directly */}
        {isAgentTask && task.todos && task.todos.length > 0 && (
          <Box sx={{ mb: 3 }}>
            <Typography variant="h6" sx={{ fontWeight: 600, mb: 2 }}>
              Tasks ({task.todos.filter(t => t.status === 'completed').length}/{task.todos.length})
            </Typography>
            <List dense disablePadding>
              {task.todos.map((todo, idx) => (
                <Box key={todo.id}>
                  <ListItem
                    sx={{
                      px: 0,
                      py: 1,
                      opacity: todo.status === 'completed' ? 0.6 : 1,
                      alignItems: 'flex-start',
                    }}
                  >
                    <ListItemIcon sx={{ minWidth: 32, mt: 0.5 }}>
                      {getTodoStatusIcon(todo.status)}
                    </ListItemIcon>
                    <Box sx={{ flex: 1 }}>
                      {/* Description */}
                      <Typography
                        variant="body2"
                        sx={{
                          fontWeight: 600,
                          textDecoration: todo.status === 'completed' ? 'line-through' : 'none',
                          mb: 0.5,
                        }}
                      >
                        {todo.description}
                      </Typography>

                      {/* File Path */}
                      {todo.filePath && (
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                          <Code fontSize="small" sx={{ color: 'text.secondary' }} />
                          <Typography
                            variant="caption"
                            sx={{
                              fontFamily: 'monospace',
                              color: 'text.secondary',
                              backgroundColor: 'grey.100',
                              px: 0.5,
                              py: 0.25,
                              borderRadius: 0.5,
                            }}
                          >
                            {todo.filePath}
                          </Typography>
                        </Box>
                      )}

                      {/* Function Name */}
                      {todo.functionName && (
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                          <Functions fontSize="small" sx={{ color: 'primary.main' }} />
                          <Typography
                            variant="caption"
                            sx={{
                              fontFamily: 'monospace',
                              color: 'primary.main',
                              fontWeight: 600,
                            }}
                          >
                            {todo.functionName}()
                          </Typography>
                        </Box>
                      )}

                      {/* Context Hint */}
                      {todo.contextHint && (
                        <Paper
                          elevation={0}
                          sx={{
                            mt: 1,
                            p: 1,
                            backgroundColor: '#f0fdf4',
                            border: '1px solid #86efac',
                          }}
                        >
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                            <Lightbulb fontSize="small" sx={{ color: '#16a34a' }} />
                            <Typography variant="caption" sx={{ fontWeight: 600, color: '#16a34a' }}>
                              Implementation Hint:
                            </Typography>
                          </Box>
                          <Typography variant="caption" color="text.secondary">
                            {todo.contextHint}
                          </Typography>
                        </Paper>
                      )}

                      {/* Notes */}
                      {todo.notes && (
                        <Typography
                          variant="caption"
                          color="text.secondary"
                          sx={{
                            display: 'block',
                            mt: 0.5,
                            fontStyle: 'italic',
                            pl: 1,
                            borderLeft: '2px solid',
                            borderColor: 'divider',
                          }}
                        >
                          {todo.notes}
                        </Typography>
                      )}
                    </Box>
                  </ListItem>

                  {/* TODO Prompt Notes */}
                  <TodoPromptNotes
                    todo={todo}
                    agentTaskId={task.id}
                    isTaskPending={task.status === 'pending'}
                    onUpdate={handleRefresh}
                  />

                  {/* Divider between todos */}
                  {idx < (task.todos?.length || 0) - 1 && (
                    <Divider sx={{ my: 1 }} />
                  )}
                </Box>
              ))}
            </List>
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
                      {agentTask.contextSummary && (
                        <Chip
                          label="📋 Context-Rich"
                          size="small"
                          sx={{
                            height: 20,
                            backgroundColor: '#f0fdf4',
                            color: '#16a34a',
                            fontSize: '0.7rem',
                            fontWeight: 600,
                          }}
                        />
                      )}
                      {agentTask.humanPromptNotes && (
                        <Chip
                          icon={<NoteAdd />}
                          label="Has Notes"
                          size="small"
                          sx={{
                            height: 20,
                            backgroundColor: '#fef3c7',
                            color: '#92400e',
                            fontSize: '0.7rem',
                            fontWeight: 600,
                          }}
                        />
                      )}
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

                  {/* Human Guidance Notes */}
                  <Box sx={{ mb: 3 }}>
                    <PromptNotesEditor
                      notes={agentTask.humanPromptNotes}
                      notesAddedAt={agentTask.humanPromptNotesAddedAt}
                      isEditable={agentTask.status === 'pending'}
                      onSave={(notes) => handleSaveTaskNotes(agentTask.id, notes)}
                      onClear={() => handleClearTaskNotes(agentTask.id)}
                      placeholder="Add human guidance notes to help the agent understand requirements, constraints, or context..."
                    />
                  </Box>

                  {/* Todo List */}
                  {agentTask.todos && agentTask.todos.length > 0 && (
                    <Box>
                      <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                        Tasks
                      </Typography>
                      <List dense disablePadding>
                        {agentTask.todos.map((todo, idx) => (
                          <Box key={todo.id}>
                            <ListItem
                              sx={{
                                px: 0,
                                py: 1,
                                opacity: todo.status === 'completed' ? 0.6 : 1,
                                alignItems: 'flex-start',
                              }}
                            >
                              <ListItemIcon sx={{ minWidth: 32, mt: 0.5 }}>
                                {getTodoStatusIcon(todo.status)}
                              </ListItemIcon>
                              <Box sx={{ flex: 1 }}>
                                {/* Description */}
                                <Typography
                                  variant="body2"
                                  sx={{
                                    fontWeight: 600,
                                    textDecoration: todo.status === 'completed' ? 'line-through' : 'none',
                                    mb: 0.5,
                                  }}
                                >
                                  {todo.description}
                                </Typography>

                                {/* File Path */}
                                {todo.filePath && (
                                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                                    <Code fontSize="small" sx={{ color: 'text.secondary' }} />
                                    <Typography
                                      variant="caption"
                                      sx={{
                                        fontFamily: 'monospace',
                                        color: 'text.secondary',
                                        backgroundColor: 'grey.100',
                                        px: 0.5,
                                        py: 0.25,
                                        borderRadius: 0.5,
                                      }}
                                    >
                                      {todo.filePath}
                                    </Typography>
                                  </Box>
                                )}

                                {/* Function Name */}
                                {todo.functionName && (
                                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                                    <Functions fontSize="small" sx={{ color: 'primary.main' }} />
                                    <Typography
                                      variant="caption"
                                      sx={{
                                        fontFamily: 'monospace',
                                        color: 'primary.main',
                                        fontWeight: 600,
                                      }}
                                    >
                                      {todo.functionName}()
                                    </Typography>
                                  </Box>
                                )}

                                {/* Context Hint */}
                                {todo.contextHint && (
                                  <Paper
                                    elevation={0}
                                    sx={{
                                      mt: 1,
                                      p: 1,
                                      backgroundColor: '#f0fdf4',
                                      border: '1px solid #86efac',
                                    }}
                                  >
                                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 0.5 }}>
                                      <Lightbulb fontSize="small" sx={{ color: '#16a34a' }} />
                                      <Typography variant="caption" sx={{ fontWeight: 600, color: '#16a34a' }}>
                                        Implementation Hint:
                                      </Typography>
                                    </Box>
                                    <Typography variant="caption" color="text.secondary">
                                      {todo.contextHint}
                                    </Typography>
                                  </Paper>
                                )}

                                {/* Notes */}
                                {todo.notes && (
                                  <Typography
                                    variant="caption"
                                    color="text.secondary"
                                    sx={{
                                      display: 'block',
                                      mt: 0.5,
                                      fontStyle: 'italic',
                                      pl: 1,
                                      borderLeft: '2px solid',
                                      borderColor: 'divider',
                                    }}
                                  >
                                    {todo.notes}
                                  </Typography>
                                )}
                              </Box>
                            </ListItem>

                            {/* TODO Prompt Notes */}
                            <TodoPromptNotes
                              todo={todo}
                              agentTaskId={agentTask.id}
                              isTaskPending={agentTask.status === 'pending'}
                              onUpdate={handleRefresh}
                            />

                            {/* Divider between todos */}
                            {idx < agentTask.todos.length - 1 && (
                              <Divider sx={{ my: 1 }} />
                            )}
                          </Box>
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