// Knowledge API Client - uses existing MCP client patterns
import type { SearchRequest, SearchResponse, CreateRequest, CreateResponse, KnowledgeCollection } from '../types/knowledge';

// Use relative URL so Vite dev proxy handles routing (same pattern as mcpClient.ts)
// In production, nginx will proxy /api/knowledge to the coordinator MCP server
const API_BASE = '';

async function fetchWithAuth(url: string, options: RequestInit = {}) {
  const token = localStorage.getItem('authToken');
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
      ...options.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }

  return response.json();
}

export const knowledgeApi = {
  async searchKnowledge(request: SearchRequest): Promise<SearchResponse> {
    const params = new URLSearchParams({
      collectionName: request.collection,
      query: request.query,
      limit: String(request.limit || 10)
    });
    return fetchWithAuth(`${API_BASE}/api/knowledge/search?${params}`);
  },

  async createKnowledge(request: CreateRequest): Promise<CreateResponse> {
    return fetchWithAuth(`${API_BASE}/api/knowledge`, {
      method: 'POST',
      body: JSON.stringify({
        collectionName: request.collection,
        information: request.text,
        metadata: request.metadata
      })
    });
  },

  async listCollections(): Promise<{ collections: KnowledgeCollection[] }> {
    return fetchWithAuth(`${API_BASE}/api/knowledge/collections`);
  }
};
