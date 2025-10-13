// Knowledge Base Type Definitions
// Simplified for immediate ui-dev implementation

export interface KnowledgeEntry {
  id: string;
  text: string;
  score?: number;
  metadata?: Record<string, any>;
  createdAt?: string;
}

export interface KnowledgeCollection {
  name: string;
  count: number;
  category: string;
}

export interface SearchRequest {
  collection: string;
  query: string;
  limit?: number;
}

export interface SearchResponse {
  results: KnowledgeEntry[];
  total: number;
}

export interface CreateRequest {
  collection: string;
  text: string;
  metadata?: Record<string, any>;
}

export interface CreateResponse {
  id: string;
  collection: string;
  createdAt: string;
}
