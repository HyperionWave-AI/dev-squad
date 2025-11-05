import { useState, useEffect, useMemo } from 'react';
import { DragDropContext, type DropResult } from '@hello-pangea/dnd';
import { Input } from '@/components/atoms/Input';
import { Button } from '@/components/atoms/Button';
import { KanbanColumn } from '@/components/organisms/KanbanColumn';
import { TaskDetailDialog } from '@/components/organisms/TaskDetailDialog';
import { MetricsDashboard } from '@/components/organisms/MetricsDashboard';
import ErrorBoundary from '@/components/organisms/ErrorBoundary';
import { restClient } from '@/services/restClient';
import type { HumanTask, AgentTask, TaskStatus, FlattenedTask } from '@/types/coordinator';
import { Search, Loader, AlertCircle, LayoutDashboard } from 'lucide-react';

interface KanbanColumnConfig {
  id: TaskStatus;
  title: string;
  color: string;
  bgColor: string;
}

const columns: KanbanColumnConfig[] = [
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

type TimeFilter = 'all' | 'today' | 'yesterday' | 'last100';

export function KanbanBoard() {
  const [tasks, setTasks] = useState<HumanTask[]>([]);
  const [agentTasks, setAgentTasks] = useState<AgentTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTask, setSelectedTask] = useState<FlattenedTask | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [timeFilter, setTimeFilter] = useState<TimeFilter>('all');

  // Load tasks on mount and auto-refresh
  useEffect(() => {
    loadTasks();
    const interval = setInterval(loadTasks, 30000);
    return () => clearInterval(interval);
  }, []);

  const loadTasks = async () => {
    try {
      setError(null);
      const [humanTasks, agentTasksResponse] = await Promise.all([
        restClient.listHumanTasks(),
        restClient.listAgentTasks(),
      ]);
      setTasks(humanTasks);
      setAgentTasks(agentTasksResponse);

      // Refresh selected task if dialog is open
      if (selectedTask && dialogOpen) {
        refreshSelectedTask(selectedTask.id, humanTasks, agentTasksResponse);
      }
    } catch (err) {
      console.error('Failed to load tasks:', err);
      setError(err instanceof Error ? err.message : 'Failed to load tasks');
    } finally {
      setLoading(false);
    }
  };

  const refreshSelectedTask = (taskId: string, humanTasks: HumanTask[], agents: AgentTask[]) => {
    const humanTask = humanTasks.find((t) => t.id === taskId);
    if (humanTask) {
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

    const agentTask = agents.find((t) => t.id === taskId);
    if (agentTask) {
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
    }
  };

  // Flatten tasks and group by status
  const tasksByStatus = useMemo(() => {
    const flattenedTasks: FlattenedTask[] = [];

    // Add human tasks
    tasks.forEach((humanTask) => {
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

    // Add agent tasks
    agentTasks.forEach((agentTask) => {
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
    });

    // Apply time filter
    let timeFiltered = flattenedTasks;
    if (timeFilter !== 'all') {
      const now = new Date();
      const oneDayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000);
      const twoDaysAgo = new Date(now.getTime() - 48 * 60 * 60 * 1000);

      timeFiltered = flattenedTasks.filter((task) => {
        const taskDate = new Date(task.createdAt);

        switch (timeFilter) {
          case 'today':
            return taskDate >= oneDayAgo;
          case 'yesterday':
            return taskDate >= twoDaysAgo && taskDate < oneDayAgo;
          case 'last100':
            return true;
          default:
            return true;
        }
      });

      if (timeFilter === 'last100') {
        timeFiltered.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
        timeFiltered = timeFiltered.slice(0, 100);
      }
    }

    // Filter by search query
    const filtered = searchQuery
      ? timeFiltered.filter(
          (task) =>
            task.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
            task.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
            task.tags?.some((tag) => tag.toLowerCase().includes(searchQuery.toLowerCase()))
        )
      : timeFiltered;

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
  }, [tasks, agentTasks, searchQuery, timeFilter]);

  // Calculate metrics from task data
  const taskMetrics = useMemo(() => {
    const allTasks = [...tasks, ...agentTasks];
    const totalTasks = allTasks.length;
    const completedTasks = allTasks.filter(task => task.status === 'completed').length;

    // Calculate average execution time for completed tasks
    let totalExecutionTimeMs = 0;
    let tasksWithExecutionTime = 0;

    allTasks.forEach(task => {
      if (task.status === 'completed' && task.completedAt && task.createdAt) {
        const completedDate = new Date(task.completedAt).getTime();
        const createdDate = new Date(task.createdAt).getTime();
        const executionTime = completedDate - createdDate;

        if (executionTime > 0) {
          totalExecutionTimeMs += executionTime;
          tasksWithExecutionTime++;
        }
      }
    });

    // Format average execution time
    let averageExecutionTime = 'N/A';
    if (tasksWithExecutionTime > 0) {
      const avgMs = totalExecutionTimeMs / tasksWithExecutionTime;
      const hours = Math.floor(avgMs / (1000 * 60 * 60));
      const minutes = Math.floor((avgMs % (1000 * 60 * 60)) / (1000 * 60));

      if (hours > 0) {
        averageExecutionTime = `${hours}h ${minutes}m`;
      } else {
        averageExecutionTime = `${minutes}m`;
      }
    }

    // Calculate success rate (completed / total * 100)
    const successRate = totalTasks > 0 ? (completedTasks / totalTasks) * 100 : 0;

    return {
      totalTasks,
      completedTasks,
      averageExecutionTime,
      successRate,
    };
  }, [tasks, agentTasks]);

  // Handle drag and drop
  const onDragEnd = async (result: DropResult) => {
    const { source, destination, draggableId } = result;

    if (!destination) return;
    if (source.droppableId === destination.droppableId && source.index === destination.index) {
      return;
    }

    const newStatus = destination.droppableId as TaskStatus;

    try {
      await restClient.updateTaskStatus(
        draggableId,
        newStatus,
        `Status changed from ${source.droppableId} to ${newStatus}`
      );

      // Optimistic update
      setTasks((prevTasks) =>
        prevTasks.map((task) => (task.id === draggableId ? { ...task, status: newStatus } : task))
      );
      setAgentTasks((prevTasks) =>
        prevTasks.map((task) => (task.id === draggableId ? { ...task, status: newStatus } : task))
      );
    } catch (err) {
      console.error('Failed to update task status:', err);
      loadTasks();
    }
  };

  const handleTaskClick = (task: FlattenedTask) => {
    setSelectedTask(task);
    setDialogOpen(true);
  };

  const handleDialogClose = () => {
    setDialogOpen(false);
    setSelectedTask(null);
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center min-h-screen">
        <Loader className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex justify-center items-center min-h-screen p-4">
        <div className="flex items-center gap-2 text-red-600 bg-red-50 dark:bg-red-900/20 px-4 py-3 rounded-lg border border-red-200 dark:border-red-800">
          <AlertCircle className="w-5 h-5" />
          <span>{error}</span>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <div className="container mx-auto p-6 space-y-6 max-w-7xl">
        {/* Header - Glassmorphic Container */}
        <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 shadow-lg">
          <div className="flex items-center gap-3">
            <div className="relative">
              <div className="absolute inset-0 bg-gradient-to-br from-purple-400 to-indigo-500 rounded-xl blur-lg opacity-30 animate-pulse"></div>
              <div className="relative p-3 bg-gradient-to-br from-purple-500 to-indigo-600 rounded-xl shadow-xl">
                <LayoutDashboard className="h-8 w-8 text-white" />
              </div>
            </div>
            <div>
              <h1 className="text-3xl font-bold bg-gradient-to-r from-purple-600 via-indigo-600 to-purple-600 bg-clip-text text-transparent">
                Task Board
              </h1>
              <p className="text-gray-600 dark:text-gray-400 mt-1">
                Manage and track your tasks across different stages
              </p>
            </div>
          </div>
        </div>

        {/* Filters and Search - Glassmorphic Container */}
        <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 shadow-lg">
          <div className="flex items-center gap-4">
            {/* Time Filter Buttons */}
            <div className="flex gap-2">
              {(['all', 'today', 'yesterday', 'last100'] as TimeFilter[]).map((filter) => (
                <Button
                  key={filter}
                  variant={timeFilter === filter ? 'primary' : 'outline'}
                  size="sm"
                  onClick={() => setTimeFilter(filter)}
                  className="capitalize"
                >
                  {filter === 'last100' ? 'Last 100' : filter}
                </Button>
              ))}
            </div>

            {/* Search Bar */}
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <Input
                type="text"
                placeholder="Search tasks..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
        </div>

        {/* Metrics Dashboard */}
        <MetricsDashboard metrics={taskMetrics} />

        {/* Kanban Board */}
        <DragDropContext onDragEnd={onDragEnd}>
          <div className="flex gap-4 p-4 min-h-[calc(100vh-480px)] overflow-x-auto">
            {columns.map((column) => (
              <KanbanColumn
                key={column.id}
                id={column.id}
                title={column.title}
                tasks={tasksByStatus[column.id]}
                color={column.color}
                bgColor={column.bgColor}
                onTaskClick={handleTaskClick}
              />
            ))}
          </div>
        </DragDropContext>

        {/* Task Detail Dialog */}
        {selectedTask && (
          <TaskDetailDialog
            task={selectedTask}
            open={dialogOpen}
            onClose={handleDialogClose}
            onTaskUpdate={loadTasks}
          />
        )}
      </div>
    </div>
  );
}

// Wrap with ErrorBoundary for production error handling
// Keep named export for testing purposes
export default () => (
  <ErrorBoundary>
    <KanbanBoard />
  </ErrorBoundary>
);
