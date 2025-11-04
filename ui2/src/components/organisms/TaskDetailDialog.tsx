import { useState, useEffect } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/molecules/Card';
import { Badge } from '@/components/atoms/Badge';
import { Button } from '@/components/atoms/Button';
import { cn } from '@/utils';
import ReactMarkdown from 'react-markdown';
import type { FlattenedTask, HumanTask, AgentTask, Priority, TaskStatus, TodoStatus } from '@/types/coordinator';
import { restClient } from '@/services/restClient';
import {
  X,
  Clock,
  User,
  CheckCircle,
  Loader,
  XCircle,
  AlertCircle,
  Bot,
  Code,
  FileCode,
  Database,
  Lightbulb,
  Circle,
  CircleDot,
} from 'lucide-react';

interface TaskDetailDialogProps {
  task: FlattenedTask | null;
  open: boolean;
  onClose: () => void;
  onTaskUpdate?: () => void;
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
      return <CheckCircle className="w-4 h-4" />;
    case 'in_progress':
      return <Loader className="w-4 h-4" />;
    case 'blocked':
      return <XCircle className="w-4 h-4" />;
    case 'pending':
    default:
      return <AlertCircle className="w-4 h-4" />;
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

const getTodoStatusIcon = (status: TodoStatus) => {
  switch (status) {
    case 'completed':
      return <CheckCircle className="w-4 h-4 text-green-600" />;
    case 'in_progress':
      return <CircleDot className="w-4 h-4 text-blue-600" />;
    case 'pending':
    default:
      return <Circle className="w-4 h-4 text-gray-400" />;
  }
};

const formatDate = (dateString: string): string => {
  const date = new Date(dateString);
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
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
      const tasks = await restClient.listAgentTasks();
      const relatedTasks = tasks.filter((at) => at.humanTaskId === task.id);
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
      const parent = await restClient.getHumanTask(task.humanTaskId);
      setParentTask(parent);
    } catch (error) {
      console.error('Failed to load parent task:', error);
    } finally {
      setLoading(false);
    }
  };

  if (!task) return null;

  const calculateProgress = (agentTask: AgentTask) => {
    if (!agentTask.todos || agentTask.todos.length === 0) return 0;
    const completed = agentTask.todos.filter((t) => t.status === 'completed').length;
    return (completed / agentTask.todos.length) * 100;
  };

  return (
    <Dialog.Root open={open} onOpenChange={onClose}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 z-50 animate-in fade-in" />
        <Dialog.Content
          className={cn(
            'fixed left-1/2 top-1/2 z-50 w-full max-w-3xl -translate-x-1/2 -translate-y-1/2',
            'bg-white dark:bg-gray-900 rounded-lg shadow-xl',
            'max-h-[90vh] overflow-y-auto',
            'animate-in fade-in zoom-in-95'
          )}
        >
          {/* Header */}
          <div className="sticky top-0 bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700 p-6 z-10">
            <div className="flex justify-between items-start gap-4">
              <div className="flex-1">
                <div className="flex items-center gap-2 mb-2">
                  {isAgentTask && <Bot className="w-5 h-5 text-blue-600" />}
                  <Dialog.Title className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                    {task.title || task.role || 'Task Details'}
                  </Dialog.Title>
                </div>
                {isAgentTask && task.agentName && (
                  <p className="text-sm font-medium text-blue-600 dark:text-blue-400 mb-2">
                    Agent: {task.agentName}
                  </p>
                )}
                <div className="flex flex-wrap gap-2">
                  {isAgentTask && (
                    <Badge variant="secondary" className="text-xs">
                      🤖 AGENT TASK
                    </Badge>
                  )}
                  {task.priority && (
                    <Badge variant={getPriorityVariant(task.priority)} className="text-xs uppercase">
                      {task.priority}
                    </Badge>
                  )}
                  <Badge variant="outline" className={cn('text-xs uppercase', getStatusColor(task.status))}>
                    {getStatusIcon(task.status)}
                    <span className="ml-1">{task.status.replace('_', ' ')}</span>
                  </Badge>
                  {task.tags?.map((tag, idx) => (
                    <Badge key={idx} variant="outline" className="text-xs">
                      {tag}
                    </Badge>
                  ))}
                </div>
              </div>
              <Dialog.Close asChild>
                <Button variant="ghost" size="sm" className="h-8 w-8 p-0">
                  <X className="w-4 h-4" />
                </Button>
              </Dialog.Close>
            </div>
          </div>

          {/* Content */}
          <div className="p-6 space-y-6">
            {/* Metadata */}
            <Card>
              <CardContent className="p-4">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <div className="text-gray-500 dark:text-gray-400 mb-1">Created</div>
                    <div className="flex items-center gap-1.5">
                      <Clock className="w-4 h-4" />
                      <span>{formatDate(task.createdAt)}</span>
                    </div>
                  </div>
                  <div>
                    <div className="text-gray-500 dark:text-gray-400 mb-1">Updated</div>
                    <div className="flex items-center gap-1.5">
                      <Clock className="w-4 h-4" />
                      <span>{task.updatedAt ? formatDate(task.updatedAt) : 'N/A'}</span>
                    </div>
                  </div>
                  {task.createdBy && (
                    <div>
                      <div className="text-gray-500 dark:text-gray-400 mb-1">Created By</div>
                      <div className="flex items-center gap-1.5">
                        <User className="w-4 h-4" />
                        <span>{task.createdBy}</span>
                      </div>
                    </div>
                  )}
                  {task.completedAt && (
                    <div>
                      <div className="text-gray-500 dark:text-gray-400 mb-1">Completed</div>
                      <div className="flex items-center gap-1.5 text-green-600">
                        <CheckCircle className="w-4 h-4" />
                        <span>{formatDate(task.completedAt)}</span>
                      </div>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* Description */}
            <div>
              <h3 className="text-lg font-semibold mb-3">Description</h3>
              <Card>
                <CardContent className="p-4 prose prose-sm dark:prose-invert max-w-none">
                  <ReactMarkdown>{task.description || 'No description available'}</ReactMarkdown>
                </CardContent>
              </Card>
            </div>

            {/* Notes */}
            {task.notes && (
              <div>
                <h3 className="text-lg font-semibold mb-3">Notes</h3>
                <Card className="bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800">
                  <CardContent className="p-4 prose prose-sm dark:prose-invert max-w-none">
                    <ReactMarkdown>{task.notes}</ReactMarkdown>
                  </CardContent>
                </Card>
              </div>
            )}

            {/* Context Summary */}
            {isAgentTask && task.contextSummary && (
              <div>
                <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
                  <Lightbulb className="w-5 h-5 text-green-600" />
                  Context Summary
                </h3>
                <Card className="bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800">
                  <CardContent className="p-4 prose prose-sm dark:prose-invert max-w-none">
                    <ReactMarkdown>{task.contextSummary}</ReactMarkdown>
                  </CardContent>
                </Card>
              </div>
            )}

            {/* Files Modified */}
            {isAgentTask && task.filesModified && task.filesModified.length > 0 && (
              <div>
                <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
                  <Code className="w-5 h-5 text-blue-600" />
                  Files to Modify
                </h3>
                <Card>
                  <CardContent className="p-4">
                    <ul className="space-y-2">
                      {task.filesModified.map((file, idx) => (
                        <li key={idx} className="flex items-center gap-2 text-sm font-mono">
                          <FileCode className="w-4 h-4 text-gray-400" />
                          <span className="text-gray-700 dark:text-gray-300">{file}</span>
                        </li>
                      ))}
                    </ul>
                  </CardContent>
                </Card>
              </div>
            )}

            {/* Qdrant Collections */}
            {isAgentTask && task.qdrantCollections && task.qdrantCollections.length > 0 && (
              <div>
                <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
                  <Database className="w-5 h-5 text-yellow-600" />
                  Suggested Qdrant Collections
                </h3>
                <Card className="bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800">
                  <CardContent className="p-4">
                    <div className="flex flex-wrap gap-2 mb-2">
                      {task.qdrantCollections.map((collection, idx) => (
                        <Badge key={idx} variant="outline" className="text-xs">
                          <Database className="w-3 h-3 mr-1" />
                          {collection}
                        </Badge>
                      ))}
                    </div>
                    <p className="text-xs text-gray-600 dark:text-gray-400">
                      💡 Query these collections only if you need specific technical patterns
                    </p>
                  </CardContent>
                </Card>
              </div>
            )}

            {/* Prior Work Summary */}
            {isAgentTask && task.priorWorkSummary && (
              <div>
                <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
                  <Bot className="w-5 h-5 text-purple-600" />
                  Prior Work Summary
                </h3>
                <Card className="bg-purple-50 dark:bg-purple-900/20 border-purple-200 dark:border-purple-800">
                  <CardContent className="p-4 prose prose-sm dark:prose-invert max-w-none">
                    <ReactMarkdown>{task.priorWorkSummary}</ReactMarkdown>
                  </CardContent>
                </Card>
              </div>
            )}

            {/* Parent Task */}
            {isAgentTask && parentTask && (
              <div>
                <h3 className="text-lg font-semibold mb-3">Parent Human Task</h3>
                <Card className="bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800">
                  <CardContent className="p-4">
                    <h4 className="font-semibold mb-2">{parentTask.title}</h4>
                    <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
                      {parentTask.description || parentTask.prompt}
                    </p>
                    <div className="flex gap-2">
                      <Badge variant="outline" className={cn('text-xs', getStatusColor(parentTask.status))}>
                        {parentTask.status.replace('_', ' ')}
                      </Badge>
                      <Badge variant="outline" className="text-xs">
                        Created {formatDate(parentTask.createdAt)}
                      </Badge>
                    </div>
                  </CardContent>
                </Card>
              </div>
            )}

            {/* TODOs (for agent tasks) */}
            {isAgentTask && task.todos && task.todos.length > 0 && (
              <div>
                <h3 className="text-lg font-semibold mb-3">
                  Tasks ({task.todos.filter((t) => t.status === 'completed').length}/{task.todos.length})
                </h3>
                <Card>
                  <CardContent className="p-4">
                    <ul className="space-y-4">
                      {task.todos.map((todo, idx) => (
                        <li
                          key={todo.id}
                          className={cn(
                            'flex gap-3 pb-4',
                            idx < task.todos!.length - 1 && 'border-b border-gray-200 dark:border-gray-700',
                            todo.status === 'completed' && 'opacity-60'
                          )}
                        >
                          <div className="pt-0.5">{getTodoStatusIcon(todo.status)}</div>
                          <div className="flex-1 space-y-2">
                            <p
                              className={cn(
                                'text-sm font-medium',
                                todo.status === 'completed' && 'line-through'
                              )}
                            >
                              {todo.description}
                            </p>
                            {todo.filePath && (
                              <div className="flex items-center gap-1.5 text-xs text-gray-600 dark:text-gray-400">
                                <Code className="w-3.5 h-3.5" />
                                <code className="bg-gray-100 dark:bg-gray-800 px-1.5 py-0.5 rounded">
                                  {todo.filePath}
                                </code>
                              </div>
                            )}
                            {todo.functionName && (
                              <div className="flex items-center gap-1.5 text-xs text-blue-600 dark:text-blue-400">
                                <FileCode className="w-3.5 h-3.5" />
                                <code className="font-semibold">{todo.functionName}()</code>
                              </div>
                            )}
                            {todo.contextHint && (
                              <div className="text-xs bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded p-2">
                                <div className="flex items-center gap-1 font-semibold text-green-700 dark:text-green-400 mb-1">
                                  <Lightbulb className="w-3.5 h-3.5" />
                                  Implementation Hint:
                                </div>
                                <p className="text-gray-600 dark:text-gray-400">{todo.contextHint}</p>
                              </div>
                            )}
                            {todo.notes && (
                              <p className="text-xs italic text-gray-600 dark:text-gray-400 pl-3 border-l-2 border-gray-300 dark:border-gray-600">
                                {todo.notes}
                              </p>
                            )}
                          </div>
                        </li>
                      ))}
                    </ul>
                  </CardContent>
                </Card>
              </div>
            )}

            {/* Agent Tasks (for human tasks) */}
            {!isAgentTask && (
              <div>
                <div className="flex items-center gap-2 mb-3">
                  <Bot className="w-5 h-5 text-blue-600" />
                  <h3 className="text-lg font-semibold">Agent Tasks</h3>
                  <Badge variant="secondary">{agentTasks.length}</Badge>
                </div>

                {loading && (
                  <div className="text-center py-8 text-gray-500">
                    <Loader className="w-6 h-6 animate-spin mx-auto" />
                  </div>
                )}

                {agentTasks.length === 0 && !loading && (
                  <Card className="bg-gray-50 dark:bg-gray-800">
                    <CardContent className="p-6 text-center text-gray-500">
                      No agent tasks assigned yet
                    </CardContent>
                  </Card>
                )}

                {agentTasks.map((agentTask) => {
                  const progress = calculateProgress(agentTask);
                  return (
                    <Card key={agentTask.id} className="mb-3">
                      <CardHeader className="pb-3">
                        <div className="flex justify-between items-start">
                          <div>
                            <CardTitle className="text-base flex items-center gap-2">
                              <Bot className="w-4 h-4" />
                              {agentTask.agentName}
                            </CardTitle>
                            <p className="text-sm text-gray-600 dark:text-gray-400 italic mt-1">
                              {agentTask.role}
                            </p>
                          </div>
                          <Badge variant="outline" className={cn('text-xs', getStatusColor(agentTask.status))}>
                            {agentTask.status.replace('_', ' ')}
                          </Badge>
                        </div>
                        {agentTask.todos && agentTask.todos.length > 0 && (
                          <div className="mt-3">
                            <div className="flex justify-between text-xs text-gray-600 dark:text-gray-400 mb-1">
                              <span>Progress</span>
                              <span>
                                {agentTask.todos.filter((t) => t.status === 'completed').length} /{' '}
                                {agentTask.todos.length}
                              </span>
                            </div>
                            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                              <div
                                className="bg-blue-600 h-2 rounded-full transition-all"
                                style={{ width: `${progress}%` }}
                              />
                            </div>
                          </div>
                        )}
                      </CardHeader>
                      {agentTask.notes && (
                        <CardContent className="pt-0">
                          <div className="text-sm bg-gray-50 dark:bg-gray-800 p-3 rounded border border-gray-200 dark:border-gray-700">
                            {agentTask.notes}
                          </div>
                        </CardContent>
                      )}
                    </Card>
                  );
                })}
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="sticky bottom-0 bg-white dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700 p-4">
            <div className="flex justify-end">
              <Dialog.Close asChild>
                <Button>Close</Button>
              </Dialog.Close>
            </div>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
