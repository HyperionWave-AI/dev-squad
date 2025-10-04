import React, { useState } from 'react';
import type { AgentTask, TaskStatus, TodoStatus } from '../types/coordinator';

interface AgentTaskCardProps {
  task: AgentTask;
  onClick?: () => void;
}

const statusColors: Record<TaskStatus, string> = {
  pending: 'bg-gray-50 border-gray-200',
  in_progress: 'bg-blue-50 border-blue-200',
  completed: 'bg-green-50 border-green-200',
  blocked: 'bg-red-50 border-red-200'
};

const todoStatusIcons: Record<TodoStatus, string> = {
  pending: '⚪',
  in_progress: '🔵',
  completed: '✅'
};

const todoStatusColors: Record<TodoStatus, string> = {
  pending: 'text-gray-500',
  in_progress: 'text-blue-600',
  completed: 'text-green-600'
};

export const AgentTaskCard: React.FC<AgentTaskCardProps> = ({ task, onClick }) => {
  const [expanded, setExpanded] = useState(false);

  // Calculate TODO progress
  const totalTodos = task.todos?.length || 0;
  const completedTodos = task.todos?.filter(todo => todo.status === 'completed').length || 0;
  const progressPercentage = totalTodos > 0 ? Math.round((completedTodos / totalTodos) * 100) : 0;

  const handleExpand = (e: React.MouseEvent) => {
    e.stopPropagation();
    setExpanded(!expanded);
  };

  return (
    <div
      className={`p-3 border rounded-md shadow-sm hover:shadow transition-shadow ml-6 ${
        statusColors[task.status]
      }`}
    >
      <div className="flex items-start justify-between mb-2" onClick={onClick}>
        <div className="flex-1 cursor-pointer">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-gray-700">
              {task.agentName}
            </span>
          </div>
          <h4 className="text-sm font-medium mt-1">{task.role}</h4>
        </div>
      </div>

      {/* Context Information */}
      {task.contextSummary && (
        <div className="mb-2 p-2 bg-blue-50 border border-blue-200 rounded">
          <p className="text-xs font-semibold text-blue-900 mb-1">📋 Context</p>
          <p className="text-xs text-blue-800">{task.contextSummary}</p>
        </div>
      )}

      {task.filesModified && task.filesModified.length > 0 && (
        <div className="mb-2 p-2 bg-purple-50 border border-purple-200 rounded">
          <p className="text-xs font-semibold text-purple-900 mb-1">📁 Files to Modify</p>
          <ul className="text-xs text-purple-800 list-disc list-inside">
            {task.filesModified.map((file, idx) => (
              <li key={idx} className="truncate">{file}</li>
            ))}
          </ul>
        </div>
      )}

      {task.qdrantCollections && task.qdrantCollections.length > 0 && (
        <div className="mb-2 p-2 bg-green-50 border border-green-200 rounded">
          <p className="text-xs font-semibold text-green-900 mb-1">🔍 Knowledge Collections</p>
          <div className="flex gap-1 flex-wrap">
            {task.qdrantCollections.map((collection, idx) => (
              <span key={idx} className="text-xs px-2 py-0.5 bg-green-200 text-green-900 rounded">
                {collection}
              </span>
            ))}
          </div>
        </div>
      )}

      {task.priorWorkSummary && (
        <div className="mb-2 p-2 bg-amber-50 border border-amber-200 rounded">
          <p className="text-xs font-semibold text-amber-900 mb-1">🔗 Prior Work</p>
          <p className="text-xs text-amber-800">{task.priorWorkSummary}</p>
        </div>
      )}

      {task.notes && (
        <p className="text-xs text-gray-600 mb-2 italic">{task.notes}</p>
      )}

      {/* Progress Bar */}
      <div className="mb-2">
        <div className="flex items-center justify-between text-xs mb-1">
          <span className="font-medium text-gray-700">
            Progress: {completedTodos}/{totalTodos} TODOs
          </span>
          <span className="text-gray-600">{progressPercentage}%</span>
        </div>
        <div className="w-full bg-gray-200 rounded-full h-2">
          <div
            className={`h-2 rounded-full transition-all ${
              progressPercentage === 100 ? 'bg-green-500' :
              progressPercentage > 0 ? 'bg-blue-500' :
              'bg-gray-300'
            }`}
            style={{ width: `${progressPercentage}%` }}
          />
        </div>
      </div>

      {/* TODO List Toggle */}
      {totalTodos > 0 && (
        <button
          onClick={handleExpand}
          className="text-xs text-blue-600 hover:text-blue-800 font-medium mb-2 flex items-center gap-1"
        >
          {expanded ? '▼' : '▶'} {expanded ? 'Hide' : 'Show'} TODO List
        </button>
      )}

      {/* TODO List (Expandable) */}
      {expanded && totalTodos > 0 && (
        <div className="mt-2 space-y-1 border-t pt-2">
          {task.todos.map(todo => (
            <div
              key={todo.id}
              className="flex items-start gap-2 text-xs p-2 bg-white rounded border border-gray-200"
            >
              <span className="text-base">{todoStatusIcons[todo.status]}</span>
              <div className="flex-1">
                <p className={`${todoStatusColors[todo.status]} ${
                  todo.status === 'completed' ? 'line-through' : ''
                }`}>
                  {todo.description}
                </p>

                {/* TODO Context Information */}
                {todo.contextHint && (
                  <div className="mt-1 p-1.5 bg-blue-50 border border-blue-200 rounded">
                    <p className="text-xs font-semibold text-blue-900">💡 Context Hint</p>
                    <p className="text-xs text-blue-800">{todo.contextHint}</p>
                  </div>
                )}

                {todo.filePath && (
                  <div className="mt-1 flex items-center gap-1">
                    <span className="text-purple-700 font-semibold">📄 File:</span>
                    <code className="text-xs bg-purple-100 px-1 py-0.5 rounded text-purple-900">
                      {todo.filePath}
                    </code>
                  </div>
                )}

                {todo.functionName && (
                  <div className="mt-1 flex items-center gap-1">
                    <span className="text-green-700 font-semibold">⚡ Function:</span>
                    <code className="text-xs bg-green-100 px-1 py-0.5 rounded text-green-900">
                      {todo.functionName}
                    </code>
                  </div>
                )}

                {todo.notes && (
                  <p className="text-gray-500 italic mt-1">{todo.notes}</p>
                )}
                {todo.completedAt && (
                  <p className="text-gray-400 mt-1">
                    ✓ Completed: {new Date(todo.completedAt).toLocaleString()}
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Status Badge */}
      <div className="flex items-center justify-between text-xs mt-2">
        <span className={`px-2 py-0.5 rounded font-medium ${
          task.status === 'pending' ? 'bg-gray-200 text-gray-700' :
          task.status === 'in_progress' ? 'bg-blue-200 text-blue-800' :
          task.status === 'completed' ? 'bg-green-200 text-green-800' :
          'bg-red-200 text-red-800'
        }`}>
          {task.status.replace('_', ' ')}
        </span>

        <span className="text-gray-500">
          Updated: {new Date(task.updatedAt).toLocaleString()}
        </span>
      </div>
    </div>
  );
};