/**
 * REST API Client for Hyperion Coordinator
 *
 * Provides clean REST API interface to replace direct MCP tool calls.
 * All requests use fetch() with proper error handling.
 * Endpoints are proxied through nginx at /api/* routes.
 */

import type { HumanTask, AgentTask, TodoItem, KnowledgeEntry } from '../types/coordinator';

const BASE_URL = '/api';

// API response types (raw from backend)
interface APIHumanTask {
  id: string;
  prompt: string;
  status: 'pending' | 'in_progress' | 'completed' | 'blocked';
  createdAt: string;
  updatedAt: string;
  notes?: string;
}

interface APIAgentTask {
  id: string;
  humanTaskId: string;
  agentName: string;
  role: string;
  status: 'pending' | 'in_progress' | 'completed' | 'blocked';
  todos: APITodo[];
  createdAt: string;
  updatedAt: string;
  notes?: string;
  contextSummary?: string;
  filesModified?: string[];
  priorWorkSummary?: string;
  qdrantCollections?: string[] | null;
  humanPromptNotes?: string;
  humanPromptNotesAddedAt?: string | null;
  humanPromptNotesUpdatedAt?: string | null;
}

interface APITodo {
  id: string;
  description: string;
  status: 'pending' | 'in_progress' | 'completed';
  createdAt: string;
  completedAt?: string;
  notes?: string;
  filePath?: string;
  functionName?: string;
  contextHint?: string;
  humanPromptNotes?: string;
  humanPromptNotesAddedAt?: string | null;
  humanPromptNotesUpdatedAt?: string | null;
}

interface APIKnowledgeEntry {
  id: string;
  collection: string;
  text: string;
  metadata?: Record<string, any>;
  score?: number;
}

interface CreateAgentTaskParams {
  humanTaskId: string;
  agentName: string;
  role: string;
  todos: Array<{
    description: string;
    filePath?: string;
    functionName?: string;
    contextHint?: string;
  }>;
  contextSummary?: string;
  filesModified?: string[];
  priorWorkSummary?: string;
  qdrantCollections?: string[];
}

interface CollectionInfo {
  collection: string;
  count: number;
}

// Transform functions to convert API types to UI types
function transformHumanTask(api: APIHumanTask): HumanTask {
  return {
    ...api,
    title: api.prompt?.substring(0, 60) || 'Untitled Task',
    description: api.prompt || '',
    priority: 'medium' as const,
    createdBy: 'user',
    tags: [],
  };
}

function transformTodo(api: APITodo): TodoItem {
  return {
    ...api,
    humanPromptNotesAddedAt: api.humanPromptNotesAddedAt || undefined,
    humanPromptNotesUpdatedAt: api.humanPromptNotesUpdatedAt || undefined,
  };
}

function transformAgentTask(api: APIAgentTask): AgentTask {
  return {
    ...api,
    todos: api.todos.map(transformTodo),
    qdrantCollections: api.qdrantCollections || undefined,
    humanPromptNotesAddedAt: api.humanPromptNotesAddedAt || undefined,
    humanPromptNotesUpdatedAt: api.humanPromptNotesUpdatedAt || undefined,
  };
}

function transformKnowledgeEntry(api: APIKnowledgeEntry): KnowledgeEntry {
  return {
    ...api,
    metadata: api.metadata || {},
    createdAt: new Date().toISOString(), // API doesn't return this, use current time
  };
}

/**
 * REST API Client class
 */
class RestClient {
  /**
   * Generic fetch wrapper with error handling
   */
  private async fetchJSON<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const url = `${BASE_URL}${endpoint}`;

    try {
      const response = await fetch(url, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...options?.headers,
        },
      });

      if (!response.ok) {
        const errorText = await response.text();
        let errorMessage: string;
        try {
          const errorData = JSON.parse(errorText);
          errorMessage = errorData.error || errorData.message || `HTTP ${response.status}`;
        } catch {
          errorMessage = errorText || `HTTP ${response.status}`;
        }
        throw new Error(`API Error: ${errorMessage}`);
      }

      return await response.json();
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error(`Request failed: ${String(error)}`);
    }
  }

  // ============================================================
  // HUMAN TASKS
  // ============================================================

  /**
   * List all human tasks
   */
  async listHumanTasks(): Promise<HumanTask[]> {
    const data = await this.fetchJSON<{ tasks: APIHumanTask[] }>('/tasks');
    return (data.tasks || []).map(transformHumanTask);
  }

  /**
   * Get a single human task by ID
   */
  async getHumanTask(id: string): Promise<HumanTask> {
    const task = await this.fetchJSON<APIHumanTask>(`/tasks/${id}`);
    return transformHumanTask(task);
  }

  /**
   * Create a new human task
   */
  async createHumanTask(prompt: string): Promise<{ taskId: string; status: string }> {
    return await this.fetchJSON<{ taskId: string; status: string }>('/tasks', {
      method: 'POST',
      body: JSON.stringify({ prompt }),
    });
  }

  /**
   * Update task status
   */
  async updateTaskStatus(
    taskId: string,
    status: 'pending' | 'in_progress' | 'completed' | 'blocked',
    notes?: string
  ): Promise<{ success: boolean }> {
    return await this.fetchJSON<{ success: boolean }>(`/tasks/${taskId}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status, notes }),
    });
  }

  // ============================================================
  // AGENT TASKS
  // ============================================================

  /**
   * List agent tasks with optional filter by agent name
   */
  async listAgentTasks(agentName?: string): Promise<AgentTask[]> {
    const queryParam = agentName ? `?agentName=${encodeURIComponent(agentName)}` : '';
    const data = await this.fetchJSON<{ tasks: APIAgentTask[]; total?: number }>(
      `/agent-tasks${queryParam}`
    );
    return (data.tasks || []).map(transformAgentTask);
  }

  /**
   * Create a new agent task
   */
  async createAgentTask(params: CreateAgentTaskParams): Promise<{ taskId: string }> {
    return await this.fetchJSON<{ taskId: string }>('/agent-tasks', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  }

  /**
   * Update TODO status
   */
  async updateTodoStatus(
    agentTaskId: string,
    todoId: string,
    status: 'pending' | 'in_progress' | 'completed',
    notes?: string
  ): Promise<{ success: boolean }> {
    return await this.fetchJSON<{ success: boolean }>(
      `/agent-tasks/${agentTaskId}/todos/${todoId}/status`,
      {
        method: 'PUT',
        body: JSON.stringify({ status, notes }),
      }
    );
  }

  // ============================================================
  // PROMPT NOTES (Task Level)
  // ============================================================

  /**
   * Add task-level prompt notes
   */
  async addTaskPromptNotes(agentTaskId: string, promptNotes: string): Promise<{ success: boolean }> {
    return await this.fetchJSON<{ success: boolean }>(
      `/agent-tasks/${agentTaskId}/prompt-notes`,
      {
        method: 'POST',
        body: JSON.stringify({ promptNotes }),
      }
    );
  }

  /**
   * Update task-level prompt notes
   */
  async updateTaskPromptNotes(agentTaskId: string, promptNotes: string): Promise<{ success: boolean }> {
    return await this.fetchJSON<{ success: boolean }>(
      `/agent-tasks/${agentTaskId}/prompt-notes`,
      {
        method: 'PUT',
        body: JSON.stringify({ promptNotes }),
      }
    );
  }

  /**
   * Clear task-level prompt notes
   */
  async clearTaskPromptNotes(agentTaskId: string): Promise<{ success: boolean }> {
    return await this.fetchJSON<{ success: boolean }>(
      `/agent-tasks/${agentTaskId}/prompt-notes`,
      {
        method: 'DELETE',
      }
    );
  }

  // ============================================================
  // PROMPT NOTES (TODO Level)
  // ============================================================

  /**
   * Add TODO-level prompt notes
   */
  async addTodoPromptNotes(
    agentTaskId: string,
    todoId: string,
    promptNotes: string
  ): Promise<{ success: boolean }> {
    return await this.fetchJSON<{ success: boolean }>(
      `/agent-tasks/${agentTaskId}/todos/${todoId}/prompt-notes`,
      {
        method: 'POST',
        body: JSON.stringify({ promptNotes }),
      }
    );
  }

  /**
   * Update TODO-level prompt notes
   */
  async updateTodoPromptNotes(
    agentTaskId: string,
    todoId: string,
    promptNotes: string
  ): Promise<{ success: boolean }> {
    return await this.fetchJSON<{ success: boolean }>(
      `/agent-tasks/${agentTaskId}/todos/${todoId}/prompt-notes`,
      {
        method: 'PUT',
        body: JSON.stringify({ promptNotes }),
      }
    );
  }

  /**
   * Clear TODO-level prompt notes
   */
  async clearTodoPromptNotes(agentTaskId: string, todoId: string): Promise<{ success: boolean }> {
    return await this.fetchJSON<{ success: boolean }>(
      `/agent-tasks/${agentTaskId}/todos/${todoId}/prompt-notes`,
      {
        method: 'DELETE',
      }
    );
  }

  // ============================================================
  // KNOWLEDGE
  // ============================================================

  /**
   * Query knowledge base
   */
  async queryKnowledge(
    collection: string,
    query: string,
    limit?: number
  ): Promise<KnowledgeEntry[]> {
    const data = await this.fetchJSON<{ entries: APIKnowledgeEntry[] }>('/knowledge/query', {
      method: 'POST',
      body: JSON.stringify({
        collection,
        query,
        limit: limit || 5,
      }),
    });
    return (data.entries || []).map(transformKnowledgeEntry);
  }

  /**
   * Upsert knowledge entry
   */
  async upsertKnowledge(
    collectionName: string,
    information: string,
    metadata?: Record<string, any>
  ): Promise<{ success: boolean; id?: string }> {
    return await this.fetchJSON<{ success: boolean; id?: string }>('/knowledge', {
      method: 'POST',
      body: JSON.stringify({
        collectionName,
        information,
        metadata,
      }),
    });
  }

  /**
   * Get popular knowledge collections
   */
  async getPopularCollections(limit?: number): Promise<CollectionInfo[]> {
    const queryParam = limit ? `?limit=${limit}` : '';
    const data = await this.fetchJSON<{ collections: CollectionInfo[] }>(
      `/knowledge/popular-collections${queryParam}`
    );
    return data.collections || [];
  }
}

// Export singleton instance
export const restClient = new RestClient();
export default restClient;
