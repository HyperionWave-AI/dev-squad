// ============================================================================
// ⚠️  DEPRECATED: DO NOT USE THIS FILE - USE REST API INSTEAD
// ============================================================================
//
// ARCHITECTURE VIOLATION: Direct MCP calls from UI are forbidden.
//
// The correct architecture is:
//   UI → REST API → Storage Layer (NO MCP proxying)
//
// Use these REST clients instead:
//   - restClient.ts → For tasks, todos, knowledge operations
//   - restCodeClient.ts → For code index operations
//   - knowledgeApi.ts → For knowledge base operations
//
// All REST endpoints are implemented in:
//   - coordinator/internal/api/rest_handler.go (unified coordinator)
//
// This file exists only for legacy compatibility and will be removed.
// ============================================================================

// MCP Client for connecting to coordinator MCP server via HTTP bridge
import type { HumanTask, AgentTask, KnowledgeEntry } from '../types/coordinator';

class MCPCoordinatorClient {
  private bridgeUrl: string;
  private connected = false;
  private requestId = 0;

  constructor() {
    // Use relative URL so nginx proxy handles routing
    // This eliminates CORS issues by making requests same-origin
    this.bridgeUrl = '';
  }

  async connect() {
    if (this.connected) return;

    // Check if bridge is healthy via nginx proxy
    try {
      const response = await fetch('/bridge-health');
      if (!response.ok) {
        throw new Error('HTTP bridge not healthy');
      }
      this.connected = true;
      console.log('Connected to MCP HTTP bridge via nginx proxy');
    } catch (error) {
      console.error('Failed to connect to MCP HTTP bridge:', error);
      throw error;
    }
  }

  async disconnect() {
    this.connected = false;
  }

  private async callTool(name: string, args: Record<string, any>): Promise<any> {
    const requestId = `req-${++this.requestId}`;

    const response = await fetch(`${this.bridgeUrl}/mcp/tools/call`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Request-ID': requestId,
      },
      body: JSON.stringify({
        name,
        arguments: args,
      }),
    });

    if (!response.ok) {
      let errorMsg = 'Tool call failed';
      try {
        const error = await response.json();
        errorMsg = error.error || errorMsg;
      } catch (e) {
        errorMsg = `HTTP ${response.status}: ${response.statusText}`;
      }
      throw new Error(errorMsg);
    }

    const text = await response.text();
    if (!text) {
      throw new Error('Empty response from server');
    }

    try {
      const result = JSON.parse(text);
      return result;
    } catch (e) {
      console.error('Failed to parse response:', text);
      throw new Error(`Invalid JSON response: ${text.substring(0, 100)}`);
    }
  }

  private async readResource(uri: string): Promise<any> {
    const requestId = `req-${++this.requestId}`;

    const response = await fetch(`${this.bridgeUrl}/mcp/resources/read?uri=${encodeURIComponent(uri)}`, {
      method: 'GET',
      headers: {
        'X-Request-ID': requestId,
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Resource read failed');
    }

    const result = await response.json();
    return result;
  }

  // Removed unused listResources method

  // Task reading methods using tools for dynamic data
  async listHumanTasks(): Promise<HumanTask[]> {
    try {
      const result = await this.callTool('coordinator_list_human_tasks', {});

      // Extract tasks from result metadata if available, otherwise parse from text
      let tasks = [];
      if (result.tasks) {
        tasks = result.tasks;
      } else if (result.content && result.content[0] && result.content[0].text) {
        // Try to extract JSON from text response
        const text = result.content[0].text;
        const jsonMatch = text.match(/Tasks:\n(\[[\s\S]*\])/);
        if (jsonMatch) {
          tasks = JSON.parse(jsonMatch[1]);
        }
      }

      return tasks.map((task: any) => ({
        id: task.id,
        title: task.prompt?.substring(0, 60) || 'Untitled Task',
        description: task.prompt || '',
        prompt: task.prompt,
        status: task.status,
        priority: 'medium', // Default priority
        createdAt: task.createdAt,
        updatedAt: task.updatedAt,
        createdBy: 'user',
        tags: [],
        notes: task.notes
      }));
    } catch (error) {
      console.error('Failed to list human tasks:', error);
      throw error;
    }
  }

  async getHumanTask(taskId: string): Promise<HumanTask> {
    try {
      const taskData = await this.readResource(`hyperion://task/human/${taskId}`);
      if (taskData.contents && taskData.contents[0]) {
        const taskJson = JSON.parse(taskData.contents[0].text);
        return {
          id: taskJson.id,
          title: taskJson.prompt?.substring(0, 60) || 'Untitled Task',
          description: taskJson.prompt || '',
          prompt: taskJson.prompt,
          status: taskJson.status,
          priority: 'medium',
          createdAt: taskJson.createdAt,
          updatedAt: taskJson.updatedAt,
          createdBy: 'user',
          tags: [],
          notes: taskJson.notes
        };
      }
      throw new Error(`Task ${taskId} not found`);
    } catch (error) {
      console.error('Failed to get human task:', error);
      throw error;
    }
  }

  async listAgentTasks(agentName?: string): Promise<AgentTask[]> {
    try {
      const args: any = {};
      if (agentName) {
        args.agentName = agentName;
      }

      const result = await this.callTool('coordinator_list_agent_tasks', args);

      // Extract tasks from result metadata if available, otherwise parse from text
      let tasks = [];
      if (result.tasks) {
        tasks = result.tasks;
      } else if (result.content && result.content[0] && result.content[0].text) {
        // Try to extract JSON from text response
        const text = result.content[0].text;
        const jsonMatch = text.match(/Tasks:\n(\[[\s\S]*\])/);
        if (jsonMatch) {
          tasks = JSON.parse(jsonMatch[1]);
        }
      }

      return tasks.map((task: any) => ({
        id: task.id,
        humanTaskId: task.humanTaskId,
        agentName: task.agentName,
        role: task.role,
        title: task.role || 'Untitled Agent Task',
        description: task.role || '',
        status: task.status,
        priority: 'medium',
        createdAt: task.createdAt,
        updatedAt: task.updatedAt,
        assignedBy: 'coordinator',
        dependencies: [],
        blockers: [],
        tags: [],
        // Map todos with their humanPromptNotes fields
        todos: (task.todos || []).map((todo: any) => ({
          id: todo.id,
          description: todo.description,
          status: todo.status,
          createdAt: todo.createdAt,
          completedAt: todo.completedAt,
          notes: todo.notes,
          filePath: todo.filePath,
          functionName: todo.functionName,
          contextHint: todo.contextHint,
          humanPromptNotes: todo.humanPromptNotes,
          humanPromptNotesAddedAt: todo.humanPromptNotesAddedAt,
          humanPromptNotesUpdatedAt: todo.humanPromptNotesUpdatedAt,
        })),
        notes: task.notes,
        contextSummary: task.contextSummary,
        filesModified: task.filesModified,
        qdrantCollections: task.qdrantCollections,
        priorWorkSummary: task.priorWorkSummary,
        humanPromptNotes: task.humanPromptNotes,
        humanPromptNotesAddedAt: task.humanPromptNotesAddedAt,
        humanPromptNotesUpdatedAt: task.humanPromptNotesUpdatedAt
      }));
    } catch (error) {
      console.error('Failed to list agent tasks:', error);
      throw error;
    }
  }

  async getAgentTask(taskId: string): Promise<AgentTask> {
    try {
      const tasks = await this.listAgentTasks();
      const task = tasks.find(t => t.id === taskId);
      if (!task) throw new Error(`Agent task ${taskId} not found`);
      return task;
    } catch (error) {
      console.error('Failed to get agent task:', error);
      throw error;
    }
  }

  // Tool methods
  async createHumanTask(params: {
    prompt: string;
  }): Promise<{ success: boolean; taskId: string }> {
    try {
      const result = await this.callTool('coordinator_create_human_task', {
        prompt: params.prompt
      });

      // Extract task ID from result content
      const taskId = this.extractTaskIdFromContent(result);

      return {
        success: true,
        taskId
      };
    } catch (error) {
      console.error('Failed to create human task:', error);
      throw error;
    }
  }

  async createAgentTask(params: {
    humanTaskId: string;
    agentName: string;
    role: string;
    todos: string[];
  }): Promise<{ success: boolean; taskId: string }> {
    try {
      const result = await this.callTool('coordinator_create_agent_task', {
        humanTaskId: params.humanTaskId,
        agentName: params.agentName,
        role: params.role,
        todos: params.todos
      });

      const taskId = this.extractTaskIdFromContent(result);

      return {
        success: true,
        taskId
      };
    } catch (error) {
      console.error('Failed to create agent task:', error);
      throw error;
    }
  }

  async updateTaskStatus(params: {
    taskId: string;
    status: string;
    notes?: string;
  }): Promise<{ success: boolean }> {
    try {
      await this.callTool('coordinator_update_task_status', {
        taskId: params.taskId,
        status: params.status,
        notes: params.notes || ''
      });

      return { success: true };
    } catch (error) {
      console.error('Failed to update task status:', error);
      throw error;
    }
  }

  async updateTodoStatus(params: {
    agentTaskId: string;
    todoId: string;
    status: string;
    notes?: string;
  }): Promise<{ success: boolean }> {
    try {
      await this.callTool('coordinator_update_todo_status', {
        agentTaskId: params.agentTaskId,
        todoId: params.todoId,
        status: params.status,
        notes: params.notes || ''
      });

      return { success: true };
    } catch (error) {
      console.error('Failed to update todo status:', error);
      throw error;
    }
  }

  async queryKnowledge(params: {
    collection: string;
    query: string;
    limit?: number;
  }): Promise<KnowledgeEntry[]> {
    try {
      const result = await this.callTool('qdrant_find', {
        collectionName: params.collection,
        query: params.query,
        limit: params.limit || 5
      });

      // MCP server now returns JSON array directly in text content
      if (result.content && result.content[0] && result.content[0].text) {
        const jsonText = result.content[0].text;

        try {
          const entries = JSON.parse(jsonText) as KnowledgeEntry[];
          console.log('[mcpClient] Knowledge query result:', entries.length, 'entries');
          console.log('[mcpClient] First entry sample:', entries[0]);
          return entries;
        } catch (parseError) {
          console.error('[mcpClient] Failed to parse knowledge query JSON:', parseError);
          console.error('[mcpClient] Raw response:', jsonText);
          return [];
        }
      }

      console.log('[mcpClient] No knowledge entries found in response');
      return [];
    } catch (error) {
      console.error('Failed to query knowledge:', error);
      throw error;
    }
  }

  async upsertKnowledge(params: {
    collection: string;
    text: string;
    metadata?: Record<string, any>;
  }): Promise<{ success: boolean; id: string }> {
    try {
      const result = await this.callTool('qdrant_store', {
        collectionName: params.collection,
        information: params.text,
        metadata: params.metadata || {}
      });

      const id = this.extractIdFromContent(result);

      return {
        success: true,
        id
      };
    } catch (error) {
      console.error('Failed to upsert knowledge:', error);
      throw error;
    }
  }

  // Prompt notes methods
  async addTaskPromptNotes(agentTaskId: string, promptNotes: string): Promise<void> {
    console.log('[mcpClient] addTaskPromptNotes called:', { agentTaskId, promptNotes: promptNotes.substring(0, 50) });
    try {
      const result = await this.callTool('coordinator_add_task_prompt_notes', {
        agentTaskId,
        promptNotes
      });
      console.log('[mcpClient] addTaskPromptNotes result:', result);
    } catch (error) {
      console.error('Failed to add task prompt notes:', error);
      throw error;
    }
  }

  async updateTaskPromptNotes(agentTaskId: string, promptNotes: string): Promise<void> {
    console.log('[mcpClient] updateTaskPromptNotes called:', { agentTaskId, promptNotes: promptNotes.substring(0, 50) });
    try {
      const result = await this.callTool('coordinator_update_task_prompt_notes', {
        agentTaskId,
        promptNotes
      });
      console.log('[mcpClient] updateTaskPromptNotes result:', result);
    } catch (error) {
      console.error('Failed to update task prompt notes:', error);
      throw error;
    }
  }

  async clearTaskPromptNotes(agentTaskId: string): Promise<void> {
    try {
      await this.callTool('coordinator_clear_task_prompt_notes', {
        agentTaskId
      });
    } catch (error) {
      console.error('Failed to clear task prompt notes:', error);
      throw error;
    }
  }

  async addTodoPromptNotes(agentTaskId: string, todoId: string, promptNotes: string): Promise<void> {
    try {
      await this.callTool('coordinator_add_todo_prompt_notes', {
        agentTaskId,
        todoId,
        promptNotes
      });
    } catch (error) {
      console.error('Failed to add todo prompt notes:', error);
      throw error;
    }
  }

  async updateTodoPromptNotes(agentTaskId: string, todoId: string, promptNotes: string): Promise<void> {
    try {
      await this.callTool('coordinator_update_todo_prompt_notes', {
        agentTaskId,
        todoId,
        promptNotes
      });
    } catch (error) {
      console.error('Failed to update todo prompt notes:', error);
      throw error;
    }
  }

  async clearTodoPromptNotes(agentTaskId: string, todoId: string): Promise<void> {
    try {
      await this.callTool('coordinator_clear_todo_prompt_notes', {
        agentTaskId,
        todoId
      });
    } catch (error) {
      console.error('Failed to clear todo prompt notes:', error);
      throw error;
    }
  }

  async getPopularCollections(limit?: number): Promise<Array<{ collection: string; count: number }>> {
    try {
      const result = await this.callTool('coordinator_get_popular_collections', {
        limit: limit || 5
      });

      if (result.content && result.content[0] && result.content[0].text) {
        const jsonText = result.content[0].text;
        try {
          const collections = JSON.parse(jsonText) as Array<{ collection: string; count: number }>;
          console.log('[mcpClient] Popular collections result:', collections);
          return collections;
        } catch (parseError) {
          console.error('[mcpClient] Failed to parse popular collections JSON:', parseError);
          return [];
        }
      }

      return [];
    } catch (error) {
      console.error('Failed to get popular collections:', error);
      throw error;
    }
  }

  // Helper to extract task ID from MCP result content
  private extractTaskIdFromContent(result: any): string {
    if (result.content && result.content[0] && result.content[0].text) {
      const text = result.content[0].text;
      const match = text.match(/Task ID:\s*([a-f0-9-]+)/i);
      if (match) {
        return match[1];
      }
    }
    return 'unknown';
  }

  // Helper to extract ID from MCP result content
  private extractIdFromContent(result: any): string {
    if (result.content && result.content[0] && result.content[0].text) {
      const text = result.content[0].text;
      const match = text.match(/ID:\s*([a-f0-9-]+)/i);
      if (match) {
        return match[1];
      }
    }
    return 'unknown';
  }
}

export const mcpClient = new MCPCoordinatorClient();