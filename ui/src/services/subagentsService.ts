import type { Subagent } from '@/types/subagent';
import { fetchWithAuth } from './restClient';

const API_BASE = '';

export const subagentsService = {
  async listSubagents(): Promise<{ subagents: Subagent[] }> {
    return fetchWithAuth(`${API_BASE}/api/v1/subagents`);
  },
};
