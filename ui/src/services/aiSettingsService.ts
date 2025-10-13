/**
 * AI Settings Service API Client
 *
 * Provides REST API interface for managing system prompts and subagents.
 * Follows chatService.ts patterns with typed responses.
 */

const BASE_URL = '/api/v1';

// ============================================================
// TYPE DEFINITIONS
// ============================================================

export interface SystemPromptResponse {
  systemPrompt: string;
}

export interface Subagent {
  id: string;
  name: string;
  description: string;
  systemPrompt: string;
  createdAt: string;
  updatedAt: string;
}

export interface SubagentCreate {
  name: string;
  description: string;
  systemPrompt: string;
}

export interface SubagentUpdate {
  name?: string;
  description?: string;
  systemPrompt?: string;
}

// ============================================================
// REST API FUNCTIONS
// ============================================================

/**
 * Generic fetch wrapper with error handling
 */
async function fetchJSON<T>(
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
      throw new Error(`API Error: ${errorMessage}`);
    }

    return await response.json();
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error(`Request failed: ${String(error)}`);
  }
}

// ============================================================
// SYSTEM PROMPT API
// ============================================================

/**
 * Get current system prompt
 */
export async function getSystemPrompt(): Promise<string> {
  const response = await fetchJSON<SystemPromptResponse>('/ai/system-prompt', {
    method: 'GET',
  });

  return response.systemPrompt || '';
}

/**
 * Update system prompt
 */
export async function updateSystemPrompt(systemPrompt: string): Promise<void> {
  await fetchJSON<{ success: boolean }>('/ai/system-prompt', {
    method: 'PUT',
    body: JSON.stringify({ systemPrompt }),
  });
}

// ============================================================
// SUBAGENTS API
// ============================================================

/**
 * Get all subagents
 */
export async function listSubagents(): Promise<Subagent[]> {
  const response = await fetchJSON<{ subagents: Subagent[] }>('/ai/subagents', {
    method: 'GET',
  });

  return response.subagents || [];
}

/**
 * Create a new subagent
 */
export async function createSubagent(data: SubagentCreate): Promise<Subagent> {
  const response = await fetchJSON<{ subagent: Subagent }>('/ai/subagents', {
    method: 'POST',
    body: JSON.stringify(data),
  });

  if (!response.subagent) {
    throw new Error('Failed to create subagent');
  }

  return response.subagent;
}

/**
 * Update an existing subagent
 */
export async function updateSubagent(
  id: string,
  data: SubagentUpdate
): Promise<Subagent> {
  const response = await fetchJSON<{ subagent: Subagent }>(`/ai/subagents/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });

  if (!response.subagent) {
    throw new Error('Failed to update subagent');
  }

  return response.subagent;
}

/**
 * Delete a subagent
 */
export async function deleteSubagent(id: string): Promise<void> {
  const response = await fetchJSON<{ success: boolean }>(`/ai/subagents/${id}`, {
    method: 'DELETE',
  });

  if (!response.success) {
    throw new Error('Failed to delete subagent');
  }
}
