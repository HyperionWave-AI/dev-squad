// Reflection API Service - Metacognitive self-awareness system
import type {
  Decision,
  Outcome,
  Lesson,
  ListResponse,
  CreateDecisionRequest,
  CreateOutcomeRequest,
  CreateLessonRequest
} from '../types/reflection';

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

export const reflectionService = {
  // GET endpoints
  async listDecisions(params?: {
    chatId?: string;
    taskId?: string;
    limit?: number;
  }): Promise<ListResponse<Decision>> {
    const searchParams = new URLSearchParams();
    if (params?.chatId) searchParams.set('chatId', params.chatId);
    if (params?.taskId) searchParams.set('taskId', params.taskId);
    if (params?.limit) searchParams.set('limit', String(params.limit));

    const queryString = searchParams.toString();
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/decisions${queryString ? `?${queryString}` : ''}`);
  },

  async listOutcomes(params?: {
    decisionId?: string;
    limit?: number;
  }): Promise<ListResponse<Outcome>> {
    const searchParams = new URLSearchParams();
    if (params?.decisionId) searchParams.set('decisionId', params.decisionId);
    if (params?.limit) searchParams.set('limit', String(params.limit));

    const queryString = searchParams.toString();
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/outcomes${queryString ? `?${queryString}` : ''}`);
  },

  async listLessons(params?: {
    pattern?: string;
    tag?: string;
    limit?: number;
  }): Promise<ListResponse<Lesson>> {
    const searchParams = new URLSearchParams();
    if (params?.pattern) searchParams.set('pattern', params.pattern);
    if (params?.tag) searchParams.set('tag', params.tag);
    if (params?.limit) searchParams.set('limit', String(params.limit));

    const queryString = searchParams.toString();
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/lessons${queryString ? `?${queryString}` : ''}`);
  },

  // POST endpoints
  async createDecision(request: CreateDecisionRequest): Promise<{ decisionId: string; stored: boolean }> {
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/decision`, {
      method: 'POST',
      body: JSON.stringify(request)
    });
  },

  async createOutcome(request: CreateOutcomeRequest): Promise<{ outcomeId: string; linked: boolean; calibration: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/outcome`, {
      method: 'POST',
      body: JSON.stringify(request)
    });
  },

  async createLesson(request: CreateLessonRequest): Promise<{ lessonId: string; indexed: boolean; pattern: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/lesson`, {
      method: 'POST',
      body: JSON.stringify(request)
    });
  }
};
