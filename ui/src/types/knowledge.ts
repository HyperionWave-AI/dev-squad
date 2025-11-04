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

// ERROR HANDLING ANALYSIS - TODO 1: Analyze error handling patterns
// FINDINGS:
// 1. No explicit error types defined for knowledge operations
// 2. Missing error response interfaces for API failures
// 3. No validation error types for form inputs
// 4. No loading state types for async operations
// 5. Missing error boundary types for React components
// 
// RECOMMENDATIONS:
// - Add KnowledgeError interface with error codes and messages
// - Define ValidationError type for form validation
// - Add ApiError interface for backend integration errors
// - Include LoadingState and ErrorState types
// - Define ErrorBoundaryState interface for React error boundaries