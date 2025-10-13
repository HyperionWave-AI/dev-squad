/**
 * AI Service Client for System Prompt and Subagents Management
 *
 * Provides REST API interface for:
 * - System prompt CRUD operations
 * - Subagents CRUD operations
 * - Chat session subagent assignment
 */

const BASE_URL = '/api/v1/ai';

export interface Subagent {
  id: string;
  name: string;
  description?: string;
  systemPrompt: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateSubagentParams {
  name: string;
  description?: string;
  systemPrompt: string;
}

export interface UpdateSubagentParams {
  name?: string;
  description?: string;
  systemPrompt?: string;
}

export interface ClaudeAgent {
  name: string;
  description: string;
  systemPrompt: string;
}

export interface ImportClaudeAgentsResult {
  imported: number;
  errors: string[];
  success: boolean;
}

/**
 * AI Service Client class
 */
class AIService {
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
        throw new Error(`AI Service Error: ${errorMessage}`);
      }

      return await response.json();
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error(`AI Service Request failed: ${String(error)}`);
    }
  }

  // ============================================================
  // SYSTEM PROMPT
  // ============================================================

  /**
   * Get the current system prompt
   */
  async getSystemPrompt(): Promise<string> {
    const data = await this.fetchJSON<{ systemPrompt: string }>('/system-prompt');
    return data.systemPrompt || '';
  }

  /**
   * Update the system prompt
   */
  async updateSystemPrompt(systemPrompt: string): Promise<{ success: boolean }> {
    await this.fetchJSON<{ success: boolean }>('/system-prompt', {
      method: 'PUT',
      body: JSON.stringify({ systemPrompt }),
    });
    return { success: true };
  }

  // ============================================================
  // SUBAGENTS
  // ============================================================

  /**
   * List all subagents
   */
  async listSubagents(): Promise<Subagent[]> {
    const data = await this.fetchJSON<{ subagents: Subagent[]; count: number }>('/subagents');
    return data.subagents || [];
  }

  /**
   * Get a single subagent by ID
   */
  async getSubagent(id: string): Promise<Subagent> {
    return await this.fetchJSON<Subagent>(`/subagents/${id}`);
  }

  /**
   * Create a new subagent
   */
  async createSubagent(params: CreateSubagentParams): Promise<Subagent> {
    return await this.fetchJSON<Subagent>('/subagents', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  }

  /**
   * Update an existing subagent
   */
  async updateSubagent(id: string, params: UpdateSubagentParams): Promise<Subagent> {
    return await this.fetchJSON<Subagent>(`/subagents/${id}`, {
      method: 'PUT',
      body: JSON.stringify(params),
    });
  }

  /**
   * Delete a subagent
   */
  async deleteSubagent(id: string): Promise<{ success: boolean }> {
    await this.fetchJSON<{ success: boolean }>(`/subagents/${id}`, {
      method: 'DELETE',
    });
    return { success: true };
  }

  /**
   * List available Claude agents from .claude/agents directory
   */
  async listClaudeAgents(): Promise<ClaudeAgent[]> {
    const data = await this.fetchJSON<{ agents: ClaudeAgent[] }>('/claude-agents');
    return data.agents || [];
  }

  /**
   * Import selected Claude agents as subagents
   */
  async importClaudeAgents(agentNames: string[]): Promise<ImportClaudeAgentsResult> {
    return await this.fetchJSON<ImportClaudeAgentsResult>('/subagents/import-claude', {
      method: 'POST',
      body: JSON.stringify({ agentNames }),
    });
  }

  /**
   * Import all available Claude agents as subagents
   */
  async importAllClaudeAgents(): Promise<ImportClaudeAgentsResult> {
    return await this.fetchJSON<ImportClaudeAgentsResult>('/subagents/import-all-claude', {
      method: 'POST',
      body: JSON.stringify({}),
    });
  }

  // ============================================================
  // CHAT SESSION SUBAGENT ASSIGNMENT
  // ============================================================

  /**
   * Set the subagent for a chat session
   * @param sessionId - Chat session ID
   * @param subagentId - Subagent ID or null for default AI
   */
  async setChatSessionSubagent(sessionId: string, subagentId: string | null): Promise<{ success: boolean }> {
    await this.fetchJSON<{ success: boolean }>(`/chat/sessions/${sessionId}/subagent`, {
      method: 'PUT',
      body: JSON.stringify({ subagentId }),
    });
    return { success: true };
  }
}

// Export singleton instance
export const aiService = new AIService();
export default aiService;
