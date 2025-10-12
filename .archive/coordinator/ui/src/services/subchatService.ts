/**
 * Subchat Service Client for Parallel Workflow Management
 *
 * Provides REST API interface for:
 * - Listing available subagents
 * - Creating subchats assigned to subagents
 * - Retrieving subchats by parent chat ID
 */

const BASE_URL = '/api/v1';

export interface Subagent {
  name: string;
  description: string;
  tools: string[];
  category: string;
}

export interface Subchat {
  id: string;
  parentChatId: string;
  subagentName: string;
  assignedTaskId?: string;
  assignedTodoId?: string;
  createdAt: string;
  updatedAt: string;
  status: 'active' | 'completed' | 'failed';
}

export interface CreateSubchatParams {
  parentChatId: string;
  subagentName: string;
  taskId?: string;
  todoId?: string;
}

/**
 * Subchat Service Client class
 */
class SubchatService {
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
        throw new Error(`Subchat Service Error: ${errorMessage}`);
      }

      return await response.json();
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error(`Subchat Service Request failed: ${String(error)}`);
    }
  }

  // ============================================================
  // SUBAGENTS
  // ============================================================

  /**
   * List all available subagents
   */
  async listSubagents(): Promise<Subagent[]> {
    const data = await this.fetchJSON<{ subagents: Subagent[] }>('/subagents');
    return data.subagents || [];
  }

  // ============================================================
  // SUBCHATS
  // ============================================================

  /**
   * Create a new subchat
   */
  async createSubchat(params: CreateSubchatParams): Promise<Subchat> {
    return await this.fetchJSON<Subchat>('/subchats', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  }

  /**
   * Get a single subchat by ID
   */
  async getSubchat(id: string): Promise<Subchat> {
    return await this.fetchJSON<Subchat>(`/subchats/${id}`);
  }

  /**
   * Get all subchats for a parent chat ID
   */
  async getSubchatsByParent(parentChatId: string): Promise<Subchat[]> {
    const data = await this.fetchJSON<{ subchats: Subchat[] }>(
      `/chats/${parentChatId}/subchats`
    );
    return data.subchats || [];
  }
}

// Export singleton instance
export const subchatService = new SubchatService();
export default subchatService;
