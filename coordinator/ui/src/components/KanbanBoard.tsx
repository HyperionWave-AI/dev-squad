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
import { mcpClient } from '../services/mcpClient';
import type { HumanTask, TaskStatus } from '../types/coordinator';

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
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  // Load tasks on mount
  useEffect(() => {
    loadTasks();

    // Auto-refresh every 30 seconds
    const interval = setInterval(loadTasks, 30000);
    return () => clearInterval(interval);
  }, []);

  const loadTasks = async () => {
    try {
      setError(null);
      await mcpClient.connect();
      const humanTasks = await mcpClient.listHumanTasks();
      setTasks(humanTasks);
    } catch (err) {
      console.error('Failed to load tasks:', err);
      setError(err instanceof Error ? err.message : 'Failed to load tasks');
    } finally {
      setLoading(false);
    }
  };

  // Filter and organize tasks by status
  const tasksByStatus = useMemo(() => {
    const filtered = searchQuery
      ? tasks.filter(
          (task) =>
            task.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
            task.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
            task.tags.some((tag) => tag.toLowerCase().includes(searchQuery.toLowerCase()))
        )
      : tasks;

    const grouped: Record<TaskStatus, HumanTask[]> = {
      pending: [],
      in_progress: [],
      blocked: [],
      completed: [],
    };

    filtered.forEach((task) => {
      grouped[task.status].push(task);
    });

    return grouped;
  }, [tasks, searchQuery]);

  // Handle drag and drop
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
    const taskId = draggableId;

    // Optimistic update
    setTasks((prevTasks) =>
      prevTasks.map((task) =>
        task.id === taskId ? { ...task, status: newStatus } : task
      )
    );

    // Update on server
    try {
      await mcpClient.updateTaskStatus({
        taskId,
        status: newStatus,
        notes: `Status changed from ${source.droppableId} to ${newStatus}`,
      });
    } catch (err) {
      console.error('Failed to update task status:', err);
      // Revert optimistic update
      setTasks((prevTasks) =>
        prevTasks.map((task) =>
          task.id === taskId ? { ...task, status: source.droppableId as TaskStatus } : task
        )
      );
      setError('Failed to update task status');
    }
  };

  const handleTaskClick = (_task: HumanTask) => {
    // TODO: Open task detail modal
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
    </Box>
  );
}