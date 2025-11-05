import { Draggable } from '@hello-pangea/dnd';
import { Card, CardContent } from '@/components/molecules/Card';
import { Badge } from '@/components/atoms/Badge';
import { cn } from '@/utils';
import type { FlattenedTask, Priority, TaskStatus } from '@/types/coordinator';
import { Clock, User, CheckCircle, Loader, XCircle, AlertCircle } from 'lucide-react';

interface TaskCardProps {
  task: FlattenedTask;
  index: number;
  onClick?: (task: FlattenedTask) => void;
}

const getPriorityVariant = (priority?: Priority): 'default' | 'warning' | 'destructive' => {
  switch (priority) {
    case 'urgent':
      return 'destructive';
    case 'high':
      return 'warning';
    case 'medium':
    case 'low':
    default:
      return 'default';
  }
};

const getStatusIcon = (status: TaskStatus) => {
  switch (status) {
    case 'completed':
      return <CheckCircle className="w-3.5 h-3.5" />;
    case 'in_progress':
      return <Loader className="w-3.5 h-3.5" />;
    case 'blocked':
      return <XCircle className="w-3.5 h-3.5" />;
    case 'pending':
    default:
      return <AlertCircle className="w-3.5 h-3.5" />;
  }
};

const getStatusColor = (status: TaskStatus): string => {
  switch (status) {
    case 'completed':
      return 'text-green-600 dark:text-green-400';
    case 'in_progress':
      return 'text-blue-600 dark:text-blue-400';
    case 'blocked':
      return 'text-red-600 dark:text-red-400';
    case 'pending':
    default:
      return 'text-gray-600 dark:text-gray-400';
  }
};

const getTaskTypeColor = (taskType: 'human' | 'agent' | 'todo'): string => {
  switch (taskType) {
    case 'human':
      return 'border-blue-500';
    case 'agent':
      return 'border-purple-500';
    case 'todo':
      return 'border-green-500';
    default:
      return 'border-gray-500';
  }
};

const getTaskTypeLabel = (taskType: 'human' | 'agent' | 'todo'): string => {
  switch (taskType) {
    case 'human':
      return '👤 Human';
    case 'agent':
      return '🤖 Agent';
    case 'todo':
      return '📋 Todo';
    default:
      return '';
  }
};

const formatDate = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const diffTime = Math.abs(now.getTime() - date.getTime());
  const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24));

  if (diffDays === 0) return 'Today';
  if (diffDays === 1) return 'Yesterday';
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
};

export function TaskCard({ task, index, onClick }: TaskCardProps) {
  return (
    <Draggable draggableId={task.id} index={index}>
      {(provided, snapshot) => (
        <div
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          onClick={() => onClick?.(task)}
          className={cn(
            'mb-3 cursor-pointer transition-all duration-200',
            snapshot.isDragging && 'rotate-2 opacity-90'
          )}
        >
          <Card
            className={cn(
              'border-l-4',
              getTaskTypeColor(task.taskType),
              'hover:shadow-md hover:-translate-y-0.5',
              snapshot.isDragging && 'shadow-lg'
            )}
          >
            <CardContent className="p-4">
              {/* Header with Task Type and Priority */}
              <div className="flex justify-between items-start mb-2 gap-2">
                <div className="flex gap-1.5 flex-wrap">
                  <Badge variant="secondary" className="text-xs">
                    {getTaskTypeLabel(task.taskType)}
                  </Badge>
                  {task.priority && (
                    <Badge variant={getPriorityVariant(task.priority)} className="text-xs uppercase">
                      {task.priority}
                    </Badge>
                  )}
                </div>
              </div>

              {/* Task Title */}
              <h3 className="text-sm font-semibold mb-2 text-gray-900 dark:text-gray-100 line-clamp-2">
                {task.title}
              </h3>

              {/* Task Description */}
              {task.description && (
                <p className="text-xs text-gray-600 dark:text-gray-400 mb-3 line-clamp-2">
                  {task.description}
                </p>
              )}

              {/* Context Summary */}
              {task.contextSummary && (
                <div className="mb-2 p-2 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded text-xs">
                  <div className="font-semibold text-blue-900 dark:text-blue-200 mb-1">📋 Context</div>
                  <div className="text-blue-800 dark:text-blue-300 line-clamp-2">{task.contextSummary}</div>
                </div>
              )}

              {/* Files Modified */}
              {task.filesModified && task.filesModified.length > 0 && (
                <div className="mb-2 p-2 bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded text-xs">
                  <div className="font-semibold text-purple-900 dark:text-purple-200 mb-1">
                    📁 Files ({task.filesModified.length})
                  </div>
                  <ul className="text-purple-800 dark:text-purple-300 pl-4 space-y-0.5">
                    {task.filesModified.slice(0, 3).map((file, idx) => (
                      <li key={idx} className="truncate">
                        {file}
                      </li>
                    ))}
                    {task.filesModified.length > 3 && (
                      <li className="italic">+ {task.filesModified.length - 3} more</li>
                    )}
                  </ul>
                </div>
              )}

              {/* Qdrant Collections */}
              {task.qdrantCollections && task.qdrantCollections.length > 0 && (
                <div className="mb-2 p-2 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded text-xs">
                  <div className="font-semibold text-green-900 dark:text-green-200 mb-1">🔍 Knowledge</div>
                  <div className="flex flex-wrap gap-1">
                    {task.qdrantCollections.map((collection, idx) => (
                      <Badge key={idx} variant="success" className="text-[10px] px-1.5 py-0">
                        {collection}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {/* Prior Work Summary */}
              {task.priorWorkSummary && (
                <div className="mb-2 p-2 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded text-xs">
                  <div className="font-semibold text-yellow-900 dark:text-yellow-200 mb-1">🔗 Prior Work</div>
                  <div className="text-yellow-800 dark:text-yellow-300 line-clamp-2">{task.priorWorkSummary}</div>
                </div>
              )}

              {/* TODOs Count (for agent tasks) */}
              {task.todos && task.todos.length > 0 && (
                <div className="mb-3">
                  <Badge variant="outline" className="text-xs">
                    ✓ {task.todos.filter((t) => t.status === 'completed').length}/{task.todos.length} TODOs
                  </Badge>
                </div>
              )}

              {/* Tags */}
              {task.tags && task.tags.length > 0 && (
                <div className="flex flex-wrap gap-1 mb-3">
                  {task.tags.map((tag, idx) => (
                    <Badge key={idx} variant="outline" className="text-[10px] px-1.5 py-0">
                      {tag}
                    </Badge>
                  ))}
                </div>
              )}

              {/* Footer with Status and Date */}
              <div className="flex justify-between items-center pt-2 border-t border-gray-200 dark:border-gray-700">
                <div className={cn('flex items-center gap-1.5 text-xs', getStatusColor(task.status))}>
                  {getStatusIcon(task.status)}
                  <span className="font-medium">{task.status.replace('_', ' ')}</span>
                </div>

                <div className="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
                  <Clock className="w-3 h-3" />
                  <span>{formatDate(task.createdAt)}</span>
                </div>
              </div>

              {/* Created By */}
              {task.createdBy && (
                <div className="flex items-center gap-1.5 mt-2 pt-2 border-t border-gray-200 dark:border-gray-700 text-xs text-gray-500 dark:text-gray-400">
                  <User className="w-3 h-3" />
                  <span>{task.createdBy}</span>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </Draggable>
  );
}
