import { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Paper,
  Typography,
  Badge,
  CircularProgress,
  Alert,
  TextField,
  InputAdornment,
} from '@mui/material';
import { Search } from '@mui/icons-material';
import { DragDropContext, Droppable, type DropResult } from '@hello-pangea/dnd';
import { KanbanTaskCard } from './KanbanTaskCard';
import { TaskDetailDialog } from './TaskDetailDialog';
import { restClient } from '../services/restClient';
import type { HumanTask, AgentTask, TaskStatus, FlattenedTask } from '../types/coordinator';

interface KanbanColumn {
  id: TaskStatus;
  title: string;
  color: string;
  bgColor: string;
}

const columns: KanbanColumn[] = [
  {
    id: 'pending',
    title: 'Pending',
    color: '#64748b',
    bgColor: '#f8fafc',
  },
  {
    id: 'in_progress',
    title: 'In Progress',
    color: '#2563eb',
    bgColor: '#eff6ff',
  },
  {
    id: 'blocked',
    title: 'Blocked',
    color: '#dc2626',
    bgColor: '#fef2f2',
  },
  {
    id: 'completed',
    title: 'Completed',
    color: '#16a34a',
    bgColor: '#f0fdf4',
  },
];

export function KanbanBoard() {
  const [tasks, setTasks] = useState<HumanTask[]>([]);
  const [agentTasks, setAgentTasks] = useState<AgentTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTask, setSelectedTask] = useState<FlattenedTask | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);

  // Load tasks on mount
  useEffect(() => {
    loadTasks();

    // Auto-refresh every 30 seconds
    const interval = setInterval(loadTasks, 30000);
    return () => clearInterval(interval);
  }, []);

  const loadTasks = async () => {
    console.log('[KanbanBoard] loadTasks called, selectedTask:', selectedTask?.id, 'dialogOpen:', dialogOpen);
    try {
      setError(null);
      const [humanTasks, agentResponse] = await Promise.all([
        restClient.listHumanTasks(),
        restClient.listAgentTasks()
      ]);
      console.log('[KanbanBoard] Tasks loaded, agents count:', agentResponse.length);
      setTasks(humanTasks);
      setAgentTasks(agentResponse);

      // If dialog is open, refresh the selected task with fresh data
      if (selectedTask && dialogOpen) {
        console.log('[KanbanBoard] Refreshing selected task:', selectedTask.id);
        refreshSelectedTask(selectedTask.id, humanTasks, agentResponse);
      } else {
        console.log('[KanbanBoard] Not refreshing - selectedTask:', !!selectedTask, 'dialogOpen:', dialogOpen);
      }
    } catch (err) {
      console.error('Failed to load tasks:', err);
      setError(err instanceof Error ? err.message : 'Failed to load tasks');
    } finally {
      setLoading(false);
    }
  };

  // Refresh the selected task after data is reloaded (only Human and Agent tasks)
  const refreshSelectedTask = (taskId: string, humanTasks: HumanTask[], agents: AgentTask[]) => {
    console.log('[KanbanBoard] refreshSelectedTask called for:', taskId);
    // Check if it's a human task
    const humanTask = humanTasks.find(t => t.id === taskId);
    if (humanTask) {
      console.log('[KanbanBoard] Found human task, updating selectedTask');
      setSelectedTask({
        id: humanTask.id,
        title: humanTask.title,
        description: humanTask.description,
        status: humanTask.status,
        priority: humanTask.priority,
        createdAt: humanTask.createdAt,
        updatedAt: humanTask.updatedAt,
        completedAt: humanTask.completedAt,
        taskType: 'human',
        tags: humanTask.tags,
        notes: humanTask.notes,
        createdBy: humanTask.createdBy,
      });
      return;
    }

    // Check if it's an agent task
    const agentTask = agents.find(t => t.id === taskId);
    if (agentTask) {
      console.log('[KanbanBoard] Found agent task, humanPromptNotes:', agentTask.humanPromptNotes);
      setSelectedTask({
        id: agentTask.id,
        title: agentTask.title || `${agentTask.agentName}: ${agentTask.role}`,
        description: agentTask.role,
        status: agentTask.status,
        priority: agentTask.priority,
        createdAt: agentTask.createdAt,
        updatedAt: agentTask.updatedAt,
        completedAt: agentTask.completedAt,
        taskType: 'agent',
        agentName: agentTask.agentName,
        role: agentTask.role,
        humanTaskId: agentTask.humanTaskId,
        tags: agentTask.tags || [],
        notes: agentTask.notes,
        contextSummary: agentTask.contextSummary,
        filesModified: agentTask.filesModified,
        qdrantCollections: agentTask.qdrantCollections,
        priorWorkSummary: agentTask.priorWorkSummary,
        todos: agentTask.todos,
        humanPromptNotes: agentTask.humanPromptNotes,
        humanPromptNotesAddedAt: agentTask.humanPromptNotesAddedAt,
        humanPromptNotesUpdatedAt: agentTask.humanPromptNotesUpdatedAt,
      });
      return;
    }
    console.log('[KanbanBoard] Task not found in humanTasks or agents');
  };

  // Flatten tasks (human tasks and agent tasks only - TODOs are shown inside agent task cards)
  const tasksByStatus = useMemo(() => {
    const flattenedTasks: FlattenedTask[] = [];

    // Add human tasks
    tasks.forEach(humanTask => {
      flattenedTasks.push({
        id: humanTask.id,
        title: humanTask.title,
        description: humanTask.description,
        status: humanTask.status,
        priority: humanTask.priority,
        createdAt: humanTask.createdAt,
        updatedAt: humanTask.updatedAt,
        completedAt: humanTask.completedAt,
        taskType: 'human',
        tags: humanTask.tags,
        notes: humanTask.notes,
        createdBy: humanTask.createdBy,
      });
    });

    // Add agent tasks (but NOT their todos - those are shown inside the agent task card)
    agentTasks.forEach(agentTask => {
      // Add the agent task itself
      flattenedTasks.push({
        id: agentTask.id,
        title: agentTask.title || `${agentTask.agentName}: ${agentTask.role}`,
        description: agentTask.role,
        status: agentTask.status,
        priority: agentTask.priority,
        createdAt: agentTask.createdAt,
        updatedAt: agentTask.updatedAt,
        completedAt: agentTask.completedAt,
        taskType: 'agent',
        agentName: agentTask.agentName,
        role: agentTask.role,
        humanTaskId: agentTask.humanTaskId,
        tags: agentTask.tags || [],
        notes: agentTask.notes,
        contextSummary: agentTask.contextSummary,
        filesModified: agentTask.filesModified,
        qdrantCollections: agentTask.qdrantCollections,
        priorWorkSummary: agentTask.priorWorkSummary,
        todos: agentTask.todos,
        humanPromptNotes: agentTask.humanPromptNotes,
        humanPromptNotesAddedAt: agentTask.humanPromptNotesAddedAt,
        humanPromptNotesUpdatedAt: agentTask.humanPromptNotesUpdatedAt,
      });

      // NOTE: TODOs are no longer shown as individual cards in kanban columns.
      // They are visible inside the agent task card when clicked.
    });

    // Filter by search query
    const filtered = searchQuery
      ? flattenedTasks.filter(
          (task) =>
            task.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
            task.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
            task.tags?.some((tag: string) => tag.toLowerCase().includes(searchQuery.toLowerCase()))
        )
      : flattenedTasks;

    // Group by status
    const grouped: Record<TaskStatus, FlattenedTask[]> = {
      pending: [],
      in_progress: [],
      blocked: [],
      completed: [],
    };

    filtered.forEach((task) => {
      grouped[task.status].push(task);
    });

    return grouped;
  }, [tasks, agentTasks, searchQuery]);

  // Handle drag and drop (only for Human and Agent tasks, NOT TODOs)
  const onDragEnd = async (result: DropResult) => {
    const { source, destination, draggableId } = result;

    // Dropped outside a valid droppable
    if (!destination) return;

    // Dropped in same position
    if (source.droppableId === destination.droppableId && source.index === destination.index) {
      return;
    }

    // Update task status
    const newStatus = destination.droppableId as TaskStatus;

    // Update on server (only Human and Agent tasks can be dragged)
    try {
      await restClient.updateTaskStatus(
        draggableId,
        newStatus,
        `Status changed from ${source.droppableId} to ${newStatus}`
      );

      // Optimistic update for human tasks
      setTasks((prevTasks) =>
        prevTasks.map((task) =>
          task.id === draggableId ? { ...task, status: newStatus } : task
        )
      );

      // Optimistic update for agent tasks
      setAgentTasks((prevTasks) =>
        prevTasks.map((task) =>
          task.id === draggableId ? { ...task, status: newStatus } : task
        )
      );
    } catch (err) {
      console.error('Failed to update task status:', err);

      // Revert optimistic updates
      setTasks((prevTasks) =>
        prevTasks.map((task) =>
          task.id === draggableId ? { ...task, status: source.droppableId as TaskStatus } : task
        )
      );
      setAgentTasks((prevTasks) =>
        prevTasks.map((task) =>
          task.id === draggableId ? { ...task, status: source.droppableId as TaskStatus } : task
        )
      );

      setError('Failed to update task status');

      // Reload to ensure UI is in sync
      await loadTasks();
    }
  };

  const handleTaskClick = (task: any) => {
    // Open dialog for both human and agent tasks
    setSelectedTask(task);
    setDialogOpen(true);
  };

  const handleDialogClose = () => {
    setDialogOpen(false);
    setSelectedTask(null);
  };

  if (loading) {
    return (
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          minHeight: '400px',
        }}
      >
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ width: '100%' }}>
      {/* Search and Filter Bar */}
      <Box sx={{ mb: 3 }}>
        <TextField
          fullWidth
          placeholder="Search tasks by title, description, or tags..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <Search />
                </InputAdornment>
              ),
              endAdornment: searchQuery && (
                <InputAdornment position="end">
                  <Typography variant="body2" color="text.secondary">
                    {tasks.filter(
                      (task) =>
                        task.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
                        task.description.toLowerCase().includes(searchQuery.toLowerCase())
                    ).length}{' '}
                    results
                  </Typography>
                </InputAdornment>
              ),
            }
          }}
          sx={{
            '& .MuiOutlinedInput-root': {
              backgroundColor: 'white',
            },
          }}
        />
      </Box>

      {/* Error Alert */}
      {error && (
        <Alert severity="error" onClose={() => setError(null)} sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      {/* Kanban Board */}
      <DragDropContext onDragEnd={onDragEnd}>
        <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(4, 1fr)' }, gap: 2 }}>
          {columns.map((column) => {
            const columnTasks = tasksByStatus[column.id];

            return (
              <Box key={column.id}>
                <Paper
                  elevation={0}
                  sx={{
                    backgroundColor: column.bgColor,
                    border: '1px solid',
                    borderColor: 'divider',
                    borderRadius: 2,
                    height: '100%',
                    minHeight: '600px',
                    display: 'flex',
                    flexDirection: 'column',
                  }}
                >
                  {/* Column Header */}
                  <Box
                    sx={{
                      p: 2,
                      borderBottom: '1px solid',
                      borderColor: 'divider',
                      backgroundColor: 'white',
                      borderTopLeftRadius: 8,
                      borderTopRightRadius: 8,
                    }}
                  >
                    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                      <Typography
                        variant="h6"
                        sx={{
                          fontSize: '0.875rem',
                          fontWeight: 600,
                          color: column.color,
                          textTransform: 'uppercase',
                          letterSpacing: '0.05em',
                        }}
                      >
                        {column.title}
                      </Typography>
                      <Badge
                        badgeContent={columnTasks.length}
                        color="primary"
                        sx={{
                          '& .MuiBadge-badge': {
                            backgroundColor: column.color,
                            color: 'white',
                          },
                        }}
                      />
                    </Box>
                  </Box>

                  {/* Droppable Area */}
                  <Droppable droppableId={column.id}>
                    {(provided, snapshot) => (
                      <Box
                        ref={provided.innerRef}
                        {...provided.droppableProps}
                        sx={{
                          p: 2,
                          flexGrow: 1,
                          minHeight: 100,
                          backgroundColor: snapshot.isDraggingOver
                            ? 'action.hover'
                            : 'transparent',
                          transition: 'background-color 0.2s ease',
                          overflowY: 'auto',
                          maxHeight: 'calc(100vh - 300px)',
                        }}
                      >
                        {columnTasks.length === 0 ? (
                          <Box
                            sx={{
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                              height: '100%',
                              minHeight: 200,
                            }}
                          >
                            <Typography
                              variant="body2"
                              color="text.secondary"
                              sx={{ textAlign: 'center' }}
                            >
                              No tasks
                            </Typography>
                          </Box>
                        ) : (
                          columnTasks.map((task, index) => (
                            <KanbanTaskCard
                              key={task.id}
                              task={task}
                              index={index}
                              onClick={handleTaskClick}
                            />
                          ))
                        )}
                        {provided.placeholder}
                      </Box>
                    )}
                  </Droppable>
                </Paper>
              </Box>
            );
          })}
        </Box>
      </DragDropContext>

      {/* Task Detail Dialog */}
      <TaskDetailDialog
        task={selectedTask}
        open={dialogOpen}
        onClose={handleDialogClose}
        onTaskUpdate={loadTasks}
      />
    </Box>
  );
}