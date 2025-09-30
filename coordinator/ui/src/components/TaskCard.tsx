import React from 'react';
import type { HumanTask, TaskStatus, Priority } from '../types/coordinator';

interface TaskCardProps {
  task: HumanTask;
  onClick?: () => void;
}

const statusColors: Record<TaskStatus, string> = {
  pending: 'bg-gray-100 text-gray-800 border-gray-300',
  in_progress: 'bg-blue-100 text-blue-800 border-blue-300',
  completed: 'bg-green-100 text-green-800 border-green-300',
  blocked: 'bg-red-100 text-red-800 border-red-300'
};

const priorityColors: Record<Priority, string> = {
  low: 'bg-gray-50 text-gray-600',
  medium: 'bg-yellow-50 text-yellow-700',
  high: 'bg-orange-50 text-orange-700',
  urgent: 'bg-red-50 text-red-700'
};

export const TaskCard: React.FC<TaskCardProps> = ({ task, onClick }) => {
  return (
    <div
      className={`p-4 border-2 rounded-lg shadow-sm hover:shadow-md transition-shadow cursor-pointer ${
        statusColors[task.status]
      }`}
      onClick={onClick}
    >
      <div className="flex justify-between items-start mb-2">
        <h3 className="font-bold text-lg">{task.title}</h3>
        <span className={`px-2 py-1 rounded text-xs font-semibold ${priorityColors[task.priority]}`}>
          {task.priority}
        </span>
      </div>

      <p className="text-sm mb-2">{task.description}</p>

      <div className="flex items-center justify-between text-xs">
        <div className="flex gap-2">
          <span className="font-semibold">Status:</span>
          <span>{task.status.replace('_', ' ')}</span>
        </div>
        <div>
          <span className="text-gray-600">
            {new Date(task.createdAt).toLocaleDateString()}
          </span>
        </div>
      </div>

      {task.tags.length > 0 && (
        <div className="flex gap-1 mt-2 flex-wrap">
          {task.tags.map((tag) => (
            <span
              key={tag}
              className="px-2 py-0.5 bg-white bg-opacity-50 rounded text-xs"
            >
              {tag}
            </span>
          ))}
        </div>
      )}
    </div>
  );
};