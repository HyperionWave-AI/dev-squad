import { useState, useEffect } from 'react';
import type { AgentTask, TaskWithChildren } from '../types/coordinator';
import { mcpClient } from '../services/mcpClient';
import { TaskCard } from './TaskCard';
import { AgentTaskCard } from './AgentTaskCard';

export const TaskDashboard: React.FC = () => {
  const [humanTasks, setHumanTasks] = useState<TaskWithChildren[]>([]);
  const [agentTasks, setAgentTasks] = useState<AgentTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadTasks = async () => {
    try {
      setLoading(true);
      setError(null);

      const [humans, agents] = await Promise.all([
        mcpClient.listHumanTasks(),
        mcpClient.listAgentTasks()
      ]);

      // Group agent tasks by human task ID
      const taskMap: Record<string, AgentTask[]> = {};
      agents.forEach(agent => {
        if (!taskMap[agent.humanTaskId]) {
          taskMap[agent.humanTaskId] = [];
        }
        taskMap[agent.humanTaskId].push(agent);
      });

      // Attach agent tasks to human tasks
      const enriched = humans.map(h => ({
        ...h,
        agentTasks: taskMap[h.id] || []
      }));

      setHumanTasks(enriched);
      setAgentTasks(agents);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load tasks');
      console.error('Error loading tasks:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    mcpClient.connect().then(() => {
      loadTasks();
      // Poll every 3 seconds for updates
      const interval = setInterval(loadTasks, 3000);
      return () => clearInterval(interval);
    });
  }, []);

  if (loading && humanTasks.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="text-4xl mb-2">⏳</div>
          <p className="text-gray-600">Loading tasks...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4">
        <div className="flex items-center gap-2">
          <span className="text-2xl">❌</span>
          <div>
            <h3 className="font-bold text-red-800">Error Loading Tasks</h3>
            <p className="text-red-600 text-sm">{error}</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold text-gray-800">Task Dashboard</h2>
          <p className="text-gray-600 text-sm">
            {humanTasks.length} human task{humanTasks.length !== 1 ? 's' : ''} • {' '}
            {agentTasks.length} agent task{agentTasks.length !== 1 ? 's' : ''}
          </p>
        </div>
        <button
          onClick={loadTasks}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          🔄 Refresh
        </button>
      </div>

      {humanTasks.length === 0 ? (
        <div className="text-center py-12 bg-gray-50 rounded-lg">
          <div className="text-4xl mb-2">📋</div>
          <p className="text-gray-600">No tasks yet. Create your first task!</p>
        </div>
      ) : (
        <div className="space-y-4">
          {humanTasks.map((task) => (
            <div key={task.id} className="space-y-2">
              <TaskCard task={task} />

              {task.agentTasks && task.agentTasks.length > 0 && (
                <div className="space-y-2">
                  {task.agentTasks.map((agentTask) => (
                    <AgentTaskCard key={agentTask.id} task={agentTask} />
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};