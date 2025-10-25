import React from 'react';
import type { HumanTask, TaskStatus, Priority } from '../types/coordinator';

interface TaskCardProps {
  task: HumanTask;
  onClick?: () => void;
}

const statusStyles: Record<TaskStatus, React.CSSProperties> = {
  pending: {
    backgroundColor: 'var(--status-pending-bg, #f3f4f6)',
    color: 'var(--status-pending-text, #374151)',
    borderColor: 'var(--status-pending-border, #d1d5db)'
  },
  in_progress: {
    backgroundColor: 'var(--status-progress-bg, #dbeafe)',
    color: 'var(--status-progress-text, #1e40af)',
    borderColor: 'var(--status-progress-border, #93c5fd)'
  },
  completed: {
    backgroundColor: 'var(--status-completed-bg, #d1fae5)',
    color: 'var(--status-completed-text, #065f46)',
    borderColor: 'var(--status-completed-border, #6ee7b7)'
  },
  blocked: {
    backgroundColor: 'var(--status-blocked-bg, #fee2e2)',
    color: 'var(--status-blocked-text, #991b1b)',
    borderColor: 'var(--status-blocked-border, #fca5a5)'
  }
};

const priorityStyles: Record<Priority, React.CSSProperties> = {
  low: {
    backgroundColor: 'var(--priority-low-bg, #f9fafb)',
    color: 'var(--priority-low-text, #4b5563)'
  },
  medium: {
    backgroundColor: 'var(--priority-medium-bg, #fffbeb)',
    color: 'var(--priority-medium-text, #b45309)'
  },
  high: {
    backgroundColor: 'var(--priority-high-bg, #fff7ed)',
    color: 'var(--priority-high-text, #c2410c)'
  },
  urgent: {
    backgroundColor: 'var(--priority-urgent-bg, #fef2f2)',
    color: 'var(--priority-urgent-text, #991b1b)'
  }
};

export const TaskCard: React.FC<TaskCardProps> = ({ task, onClick }) => {
  return (
    <div
      className="p-4 border-2 rounded-lg shadow-sm hover:shadow-md transition-all cursor-pointer"
      style={{
        ...statusStyles[task.status],
        boxShadow: 'var(--shadow)',
        transition: 'all 0.2s ease'
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.boxShadow = 'var(--shadow-lg)';
        e.currentTarget.style.transform = 'translateY(-1px)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.boxShadow = 'var(--shadow)';
        e.currentTarget.style.transform = 'translateY(0)';
      }}
      onClick={onClick}
    >
      <div className="flex justify-between items-start mb-2">
        <h3 
          className="font-bold text-lg"
          style={{ color: 'var(--text-primary)' }}
        >
          {task.title}
        </h3>
        <span 
          className="px-2 py-1 rounded text-xs font-semibold"
          style={priorityStyles[task.priority]}
        >
          {task.priority}
        </span>
      </div>

      <p 
        className="text-sm mb-2"
        style={{ color: 'var(--text-secondary)' }}
      >
        {task.description}
      </p>

      <div className="flex items-center justify-between text-xs">
        <div className="flex gap-2">
          <span 
            className="font-semibold"
            style={{ color: 'var(--text-primary)' }}
          >
            Status:
          </span>
          <span style={{ color: 'var(--text-secondary)' }}>
            {task.status.replace('_', ' ')}
          </span>
        </div>
        <div>
          <span style={{ color: 'var(--text-secondary)' }}>
            {new Date(task.createdAt).toLocaleDateString()}
          </span>
        </div>
      </div>

      {task.tags.length > 0 && (
        <div className="flex gap-1 mt-2 flex-wrap">
          {task.tags.map((tag) => (
            <span
              key={tag}
              className="px-2 py-0.5 rounded text-xs"
              style={{
                backgroundColor: 'var(--tag-bg, rgba(255, 255, 255, 0.5))',
                color: 'var(--text-primary)',
                border: '1px solid var(--border-color)'
              }}
            >
              {tag}
            </span>
          ))}
        </div>
      )}
    </div>
  );
};