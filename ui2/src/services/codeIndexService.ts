import type { SearchResult, IndexStatus, SearchRequest } from '@/types/codeIndex';
import { fetchWithAuth } from './restClient';

const API_BASE = '';

export const codeIndexService = {
  async getStatus(): Promise<IndexStatus> {
    return fetchWithAuth(`${API_BASE}/api/v1/code-index/status`);
  },

  async search(request: SearchRequest): Promise<{ results: SearchResult[] }> {
    return fetchWithAuth(`${API_BASE}/api/v1/code-index/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
  },

  async triggerScan(folderPath?: string): Promise<{ message: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/code-index/scan`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ folderPath }),
    });
  },
};
