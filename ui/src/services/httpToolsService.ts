/**
 * HTTP Tools Service
 *
 * Provides API client for managing HTTP-based tool definitions.
 * All requests use fetch() with proper error handling and JWT authentication.
 */

const BASE_URL = '/api/v1/tools/http';

// Type definitions
export interface HTTPToolDefinition {
  id?: string;
  toolName: string;
  description: string;
  endpoint: string;
  httpMethod: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  headers?: Array<{ key: string; value: string }>;
  parameters?: Array<{
    name: string;
    type: 'string' | 'number' | 'boolean' | 'object';
    required: boolean;
    description?: string;
  }>;
  authType?: 'none' | 'bearer' | 'apiKey' | 'basic';
  authConfig?: Record<string, string>;
  companyId?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface ListHTTPToolsResponse {
  tools: HTTPToolDefinition[];
  total: number;
  page: number;
  limit: number;
}

/**
 * Authenticated fetch wrapper with error handling
 */
async function authFetch<T>(
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
      credentials: 'include', // Include cookies for JWT auth
    });

    if (!response.ok) {
      const errorText = await response.text();
      let errorMessage: string;
      try {
        const errorData = JSON.parse(errorText);
        errorMessage = errorData.error || errorData.message || `HTTP ${response.status}`;
      } catch {
        errorMessage = errorText || `HTTP ${response.status}: ${response.statusText}`;
      }
      throw new Error(errorMessage);
    }

    // Handle empty responses
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
      return await response.json();
    }
    return {} as T;
  } catch (error) {
    if (error instanceof Error) {
      throw new Error(`Failed to ${options?.method || 'GET'} HTTP tool: ${error.message}`);
    }
    throw new Error(`Request failed: ${String(error)}`);
  }
}

/**
 * HTTP Tools Service API
 */
export const httpToolsService = {
  /**
   * Create a new HTTP tool definition
   */
  async addHTTPTool(tool: HTTPToolDefinition): Promise<{ id: string; message: string }> {
    return authFetch<{ id: string; message: string }>('', {
      method: 'POST',
      body: JSON.stringify(tool),
    });
  },

  /**
   * List HTTP tools with pagination
   */
  async listHTTPTools(page: number = 1, limit: number = 20): Promise<ListHTTPToolsResponse> {
    return authFetch<ListHTTPToolsResponse>(`?page=${page}&limit=${limit}`);
  },

  /**
   * Get a specific HTTP tool by ID
   */
  async getHTTPTool(id: string): Promise<HTTPToolDefinition> {
    return authFetch<HTTPToolDefinition>(`/${id}`);
  },

  /**
   * Delete an HTTP tool by ID
   */
  async deleteHTTPTool(id: string): Promise<{ message: string }> {
    return authFetch<{ message: string }>(`/${id}`, {
      method: 'DELETE',
    });
  },
};

export default httpToolsService;
