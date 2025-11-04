// Knowledge API Client - uses existing MCP client patterns
import type {
  SearchRequest,
  SearchResponse,
  CreateRequest,
  CreateResponse,
  KnowledgeCollection,
  CreateCollectionRequest,
  CreateCollectionResponse
} from '../types/knowledge';

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
    return fetchWithAuth(`${API_BASE}/api/v1/knowledge/search?${params}`);
  },

  async queryKnowledge(request: { collection: string; query: string; limit?: number; taskId?: string }): Promise<{ entries: any[] }> {
    return fetchWithAuth(`${API_BASE}/api/v1/knowledge/query`, {
      method: 'POST',
      body: JSON.stringify(request)
    });
  },

  async createKnowledge(request: CreateRequest): Promise<CreateResponse> {
    return fetchWithAuth(`${API_BASE}/api/v1/knowledge`, {
      method: 'POST',
      body: JSON.stringify({
        collectionName: request.collection,
        information: request.text,
        metadata: request.metadata
      })
    });
  },

  async listCollections(): Promise<{ collections: KnowledgeCollection[] }> {
    return fetchWithAuth(`${API_BASE}/api/v1/knowledge/collections`);
  },

  async createCollection(request: CreateCollectionRequest): Promise<CreateCollectionResponse> {
    return fetchWithAuth(`${API_BASE}/api/v1/knowledge/collections`, {
      method: 'POST',
      body: JSON.stringify(request)
    });
  },

  async getEntries(collection: string, limit: number = 50): Promise<{ entries: any[] }> {
    const params = new URLSearchParams({
      collection,
      limit: String(limit)
    });
    return fetchWithAuth(`${API_BASE}/api/v1/knowledge/browse?${params}`);
  },

  async updateEntry(id: string, data: { text: string; metadata?: Record<string, any> }): Promise<{ entry: any }> {
    return fetchWithAuth(`${API_BASE}/api/v1/knowledge/entries/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(data)
    });
  },

  async deleteEntry(id: string): Promise<void> {
    await fetchWithAuth(`${API_BASE}/api/v1/knowledge/entries/${encodeURIComponent(id)}`, {
      method: 'DELETE'
    });
  }
};
