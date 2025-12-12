import type { Subagent } from '@/types/subagent';
import { fetchWithAuth } from './restClient';

const API_BASE = '';

interface AgentSessionResponse {
  session: {
    id: string;
    title: string;
    userId: string;
    companyId: string;
    createdAt: string;
    updatedAt: string;
    activeSubagentId?: string;
  };
  agentName: string;
}

export const subagentsService = {
  async listSubagents(): Promise<{ subagents: Subagent[] }> {
    return fetchWithAuth(`${API_BASE}/api/v1/subagents`);
  },

  /**
   * Create a new chat session dedicated to a specific agent
   * @param agentName - The name of the agent (e.g., "go-dev", "ui-dev")
   * @returns The created session with agent context
   */
  async createAgentSession(agentName: string): Promise<AgentSessionResponse> {
    return fetchWithAuth(`${API_BASE}/api/v1/subagents/${agentName}/sessions`, {
      method: 'POST',
    });
  },
};
