// MCP Client for connecting to coordinator MCP server via HTTP bridge
import type { HumanTask, AgentTask, KnowledgeEntry } from '../types/coordinator';

class MCPCoordinatorClient {
  private bridgeUrl: string;
  private connected = false;
  private requestId = 0;

  constructor() {
    this.bridgeUrl = import.meta.env.VITE_MCP_BRIDGE_URL || 'http://localhost:8095';
  }

  async connect() {
    if (this.connected) return;

    // Check if bridge is healthy
    try {
      const response = await fetch(`${this.bridgeUrl}/health`);
      if (!response.ok) {
        throw new Error('HTTP bridge not healthy');
      }
      this.connected = true;
      console.log('Connected to MCP HTTP bridge at:', this.bridgeUrl);
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

    const response = await fetch(`${this.bridgeUrl}/api/mcp/tools/call`, {
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
      const error = await response.json();
      throw new Error(error.error || 'Tool call failed');
    }

    const result = await response.json();
    return result;
  }

  private async readResource(uri: string): Promise<any> {
    const requestId = `req-${++this.requestId}`;

    const response = await fetch(`${this.bridgeUrl}/api/mcp/resources/read?uri=${encodeURIComponent(uri)}`, {
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
        todos: task.todos || [],
        notes: task.notes,
        contextSummary: task.contextSummary,
        filesModified: task.filesModified,
        qdrantCollections: task.qdrantCollections,
        priorWorkSummary: task.priorWorkSummary
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
      const result = await this.callTool('coordinator_query_knowledge', {
        collection: params.collection,
        query: params.query,
        limit: params.limit || 5
      });

      // Parse results from content
      const entries: KnowledgeEntry[] = [];
      // Note: The actual parsing would depend on the MCP response format
      // For now, return empty array as we'd need to parse the text content
      console.log('Knowledge query result:', result);

      return entries;
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
      const result = await this.callTool('coordinator_upsert_knowledge', {
        collection: params.collection,
        text: params.text,
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