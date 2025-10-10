import React, { useState, useEffect } from 'react';
import type { HumanTask, AgentTask, TaskStatus } from '../types/coordinator';
import { restClient } from '../services/restClient';

interface TaskDetailProps {
  taskId: string;
  taskType: 'human' | 'agent';
  onClose: () => void;
}

export const TaskDetail: React.FC<TaskDetailProps> = ({ taskId, taskType, onClose }) => {
  const [task, setTask] = useState<HumanTask | AgentTask | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [updating, setUpdating] = useState(false);

  useEffect(() => {
    loadTask();
  }, [taskId, taskType]);

  const loadTask = async () => {
    try {
      setLoading(true);
      setError(null);

      if (taskType === 'human') {
        const humanTask = await restClient.getHumanTask(taskId);
        setTask(humanTask);
      } else {
        // For agent tasks, we need to list and find the specific one
        const agentTasks = await restClient.listAgentTasks();
        const agentTask = agentTasks.find(t => t.id === taskId);
        if (!agentTask) {
          throw new Error('Agent task not found');
        }
        setTask(agentTask);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load task');
    } finally {
      setLoading(false);
    }
  };

  const handleStatusUpdate = async (newStatus: TaskStatus) => {
    if (!task) return;

    try {
      setUpdating(true);
      await restClient.updateTaskStatus(task.id, newStatus);
      await loadTask();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to update status');
    } finally {
      setUpdating(false);
    }
  };

  if (loading) {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white p-6 rounded-lg">
          <div className="text-center">
            <div className="text-4xl mb-2">⏳</div>
            <p>Loading task details...</p>
          </div>
        </div>
      </div>
    );
  }

  if (error || !task) {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white p-6 rounded-lg max-w-md">
          <div className="text-center">
            <div className="text-4xl mb-2">❌</div>
            <p className="text-red-600 mb-4">{error || 'Task not found'}</p>
            <button
              onClick={onClose}
              className="px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-700"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    );
  }

  const isHumanTask = 'prompt' in task;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-xl max-w-3xl w-full max-h-[90vh] overflow-y-auto">
        <div className="sticky top-0 bg-white border-b px-6 py-4 flex justify-between items-center">
          <h2 className="text-2xl font-bold text-gray-800">
            {isHumanTask ? '👤 Human Task' : '🤖 Agent Task'}
          </h2>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 text-2xl"
          >
            ✕
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div>
            <label className="text-sm font-semibold text-gray-600">Title</label>
            <h3 className="text-xl font-bold text-gray-800">{task.title}</h3>
          </div>

          <div>
            <label className="text-sm font-semibold text-gray-600">Description</label>
            <p className="text-gray-700">{task.description}</p>
          </div>

          {isHumanTask && (
            <div>
              <label className="text-sm font-semibold text-gray-600">Original Prompt</label>
              <div className="mt-1 p-3 bg-gray-50 rounded border border-gray-200">
                <p className="text-gray-800 whitespace-pre-wrap">{(task as HumanTask).prompt}</p>
              </div>
            </div>
          )}

          {!isHumanTask && (
            <>
              <div>
                <label className="text-sm font-semibold text-gray-600">Agent Name</label>
                <p className="text-gray-800">{(task as AgentTask).agentName}</p>
              </div>

              <div>
                <label className="text-sm font-semibold text-gray-600">Assigned By</label>
                <p className="text-gray-800">{(task as AgentTask).assignedBy}</p>
              </div>

              {(task as AgentTask).dependencies && (task as AgentTask).dependencies!.length > 0 && (
                <div>
                  <label className="text-sm font-semibold text-gray-600">Dependencies</label>
                  <ul className="mt-1 space-y-1">
                    {(task as AgentTask).dependencies!.map((dep: string, i: number) => (
                      <li key={i} className="text-sm text-gray-600">• {dep}</li>
                    ))}
                  </ul>
                </div>
              )}

              {(task as AgentTask).blockers && (task as AgentTask).blockers!.length > 0 && (
                <div>
                  <label className="text-sm font-semibold text-red-600">Blockers</label>
                  <ul className="mt-1 space-y-1">
                    {(task as AgentTask).blockers!.map((blocker: string, i: number) => (
                      <li key={i} className="text-sm text-red-600">🚫 {blocker}</li>
                    ))}
                  </ul>
                </div>
              )}
            </>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-sm font-semibold text-gray-600">Status</label>
              <select
                value={task.status}
                onChange={(e) => handleStatusUpdate(e.target.value as TaskStatus)}
                disabled={updating}
                className="mt-1 w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="pending">Pending</option>
                <option value="in_progress">In Progress</option>
                <option value="completed">Completed</option>
                <option value="blocked">Blocked</option>
              </select>
            </div>

            <div>
              <label className="text-sm font-semibold text-gray-600">Priority</label>
              <p className="mt-1 px-3 py-2 bg-gray-50 rounded text-gray-800">
                {task.priority || 'N/A'}
              </p>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <label className="text-xs font-semibold text-gray-600">Created</label>
              <p className="text-gray-700">
                {new Date(task.createdAt).toLocaleString()}
              </p>
            </div>

            <div>
              <label className="text-xs font-semibold text-gray-600">Updated</label>
              <p className="text-gray-700">
                {new Date(task.updatedAt).toLocaleString()}
              </p>
            </div>
          </div>

          {task.completedAt && (
            <div>
              <label className="text-sm font-semibold text-green-600">Completed At</label>
              <p className="text-green-700">
                {new Date(task.completedAt).toLocaleString()}
              </p>
            </div>
          )}

          {task.tags && task.tags.length > 0 && (
            <div>
              <label className="text-sm font-semibold text-gray-600">Tags</label>
              <div className="flex gap-2 flex-wrap mt-1">
                {task.tags.map((tag: string) => (
                  <span
                    key={tag}
                    className="px-2 py-1 bg-blue-100 text-blue-800 rounded text-sm"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>

        <div className="sticky bottom-0 bg-gray-50 px-6 py-4 border-t flex justify-end">
          <button
            onClick={onClose}
            className="px-6 py-2 bg-gray-600 text-white rounded hover:bg-gray-700 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};