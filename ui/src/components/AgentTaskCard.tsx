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
      className={`
        p-3 sm:p-4 md:p-5 
        border rounded-lg 
        shadow-sm hover:shadow-md 
        transition-all duration-200 
        ml-2 sm:ml-4 md:ml-6 
        mb-3 sm:mb-4
        ${statusColors[task.status]}
        hover:scale-[1.01]
        cursor-pointer
        max-w-full
        overflow-hidden
      `}
    >
      <div className="flex items-start justify-between mb-3 sm:mb-4" onClick={onClick}>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-sm sm:text-base font-semibold text-gray-700 truncate">
              {task.agentName}
            </span>
          </div>
          <h4 className="text-sm sm:text-base font-medium text-gray-800 leading-tight">
            {task.role}
          </h4>
        </div>
      </div>

      {/* Context Information - Responsive Layout */}
      {task.contextSummary && (
        <div className="mb-3 p-2 sm:p-3 bg-blue-50 border border-blue-200 rounded-md">
          <p className="text-xs sm:text-sm font-semibold text-blue-900 mb-1">📋 Context</p>
          <p className="text-xs sm:text-sm text-blue-800 leading-relaxed break-words">
            {task.contextSummary}
          </p>
        </div>
      )}

      {/* Files to Modify - Mobile-Optimized */}
      {task.filesModified && task.filesModified.length > 0 && (
        <div className="mb-3 p-2 sm:p-3 bg-purple-50 border border-purple-200 rounded-md">
          <p className="text-xs sm:text-sm font-semibold text-purple-900 mb-2">📁 Files to Modify</p>
          <div className="space-y-1">
            {task.filesModified.map((file, idx) => (
              <div key={idx} className="text-xs sm:text-sm text-purple-800 break-all">
                <code className="bg-purple-100 px-1 py-0.5 rounded text-xs">
                  {file.split('/').pop() || file}
                </code>
                {/* Show full path on hover/focus for larger screens */}
                <div className="hidden sm:block text-xs text-purple-600 mt-0.5 opacity-75">
                  {file}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Knowledge Collections - Responsive Grid */}
      {task.qdrantCollections && task.qdrantCollections.length > 0 && (
        <div className="mb-3 p-2 sm:p-3 bg-green-50 border border-green-200 rounded-md">
          <p className="text-xs sm:text-sm font-semibold text-green-900 mb-2">🔍 Knowledge Collections</p>
          <div className="flex gap-1 sm:gap-2 flex-wrap">
            {task.qdrantCollections.map((collection, idx) => (
              <span 
                key={idx} 
                className="text-xs px-2 py-1 bg-green-200 text-green-900 rounded-full whitespace-nowrap"
              >
                {collection}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Prior Work Summary */}
      {task.priorWorkSummary && (
        <div className="mb-3 p-2 sm:p-3 bg-amber-50 border border-amber-200 rounded-md">
          <p className="text-xs sm:text-sm font-semibold text-amber-900 mb-1">🔗 Prior Work</p>
          <p className="text-xs sm:text-sm text-amber-800 leading-relaxed break-words">
            {task.priorWorkSummary}
          </p>
        </div>
      )}

      {/* Notes */}
      {task.notes && (
        <p className="text-xs sm:text-sm text-gray-600 mb-3 italic leading-relaxed break-words">
          {task.notes}
        </p>
      )}

      {/* Progress Bar - Enhanced for Mobile */}
      <div className="mb-3 sm:mb-4">
        <div className="flex items-center justify-between text-xs sm:text-sm mb-2">
          <span className="font-medium text-gray-700">
            Progress: {completedTodos}/{totalTodos} TODOs
          </span>
          <span className="text-gray-600 font-mono">
            {progressPercentage}%
          </span>
        </div>
        <div className="w-full bg-gray-200 rounded-full h-2 sm:h-3">
          <div
            className={`h-2 sm:h-3 rounded-full transition-all duration-300 ${
              progressPercentage === 100 ? 'bg-green-500' :
              progressPercentage > 0 ? 'bg-blue-500' :
              'bg-gray-300'
            }`}
            style={{ width: `${progressPercentage}%` }}
          />
        </div>
      </div>

      {/* TODO List Toggle - Touch-Friendly */}
      {totalTodos > 0 && (
        <button
          onClick={handleExpand}
          className="
            text-xs sm:text-sm 
            text-blue-600 hover:text-blue-800 
            font-medium mb-3 
            flex items-center gap-2
            py-2 px-1
            -mx-1
            rounded
            hover:bg-blue-50
            transition-colors
            touch-manipulation
            min-h-[44px] sm:min-h-[auto]
          "
        >
          <span className="text-base">{expanded ? '▼' : '▶'}</span>
          <span>{expanded ? 'Hide' : 'Show'} TODO List ({totalTodos})</span>
        </button>
      )}

      {/* TODO List (Expandable) - Mobile-Optimized */}
      {expanded && totalTodos > 0 && (
        <div className="mt-3 space-y-2 sm:space-y-3 border-t pt-3">
          {task.todos.map(todo => (
            <div
              key={todo.id}
              className="
                flex items-start gap-3 
                text-xs sm:text-sm 
                p-3 sm:p-4 
                bg-white rounded-lg 
                border border-gray-200 
                shadow-sm
              "
            >
              <span className="text-lg sm:text-xl flex-shrink-0 mt-0.5">
                {todoStatusIcons[todo.status]}
              </span>
              <div className="flex-1 min-w-0">
                <p className={`${todoStatusColors[todo.status]} ${
                  todo.status === 'completed' ? 'line-through' : ''
                } leading-relaxed break-words`}>
                  {todo.description}
                </p>

                {/* TODO Context Information */}
                {todo.contextHint && (
                  <div className="mt-2 p-2 sm:p-3 bg-blue-50 border border-blue-200 rounded-md">
                    <p className="text-xs font-semibold text-blue-900 mb-1">💡 Context Hint</p>
                    <p className="text-xs text-blue-800 leading-relaxed break-words">
                      {todo.contextHint}
                    </p>
                  </div>
                )}

                {/* File Path - Mobile-Friendly */}
                {todo.filePath && (
                  <div className="mt-2 flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-2">
                    <span className="text-purple-700 font-semibold text-xs flex-shrink-0">
                      📄 File:
                    </span>
                    <code className="text-xs bg-purple-100 px-2 py-1 rounded text-purple-900 break-all">
                      {todo.filePath}
                    </code>
                  </div>
                )}

                {/* Function Name */}
                {todo.functionName && (
                  <div className="mt-2 flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-2">
                    <span className="text-green-700 font-semibold text-xs flex-shrink-0">
                      ⚡ Function:
                    </span>
                    <code className="text-xs bg-green-100 px-2 py-1 rounded text-green-900 break-all">
                      {todo.functionName}
                    </code>
                  </div>
                )}

                {/* Notes and Completion Time */}
                {todo.notes && (
                  <p className="text-gray-500 italic mt-2 text-xs leading-relaxed break-words">
                    {todo.notes}
                  </p>
                )}
                {todo.completedAt && (
                  <p className="text-gray-400 mt-2 text-xs">
                    ✓ Completed: {new Date(todo.completedAt).toLocaleString()}
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Status Badge and Timestamp - Responsive Layout */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 text-xs mt-3 pt-3 border-t border-gray-200">
        <span className={`
          px-3 py-1.5 sm:px-2 sm:py-1 
          rounded-full font-medium 
          text-center sm:text-left
          ${
            task.status === 'pending' ? 'bg-gray-200 text-gray-700' :
            task.status === 'in_progress' ? 'bg-blue-200 text-blue-800' :
            task.status === 'completed' ? 'bg-green-200 text-green-800' :
            'bg-red-200 text-red-800'
          }
        `}>
          {task.status.replace('_', ' ').toUpperCase()}
        </span>

        <span className="text-gray-500 text-center sm:text-right">
          Updated: {new Date(task.updatedAt).toLocaleDateString()} {new Date(task.updatedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </span>
      </div>
    </div>
  );
};