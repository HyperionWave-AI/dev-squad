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
      collection: request.collection,
      query: request.query,
      limit: String(request.limit || 10)
    });
    return fetchWithAuth(`${API_BASE}/api/v1/knowledge/browse?${params}`);
  },

  async queryKnowledge(request: { collection: string; query: string; limit?: number; taskId?: string }): Promise<{ entries: any[] }> {
    return fetchWithAuth(`${API_BASE}/api/v1/knowledge/query`, {
      method: 'POST',
      body: JSON.stringify(request)
    });
  },

  async browseKnowledge(collection?: string, limit?: number): Promise<SearchResponse> {
    const params = new URLSearchParams();
    if (collection) params.append('collection', collection);
    if (limit) params.append('limit', String(limit));

    const queryString = params.toString();
    const url = `${API_BASE}/api/v1/knowledge/browse${queryString ? `?${queryString}` : ''}`;
    return fetchWithAuth(url);
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
  }
};
