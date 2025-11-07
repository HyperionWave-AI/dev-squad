import type { Decision, Lesson, Outcome } from '@/types/reflection';
import { fetchWithAuth } from './restClient';

const API_BASE = '';

export const reflectionService = {
  async listDecisions(limit: number = 50): Promise<{ decisions: Decision[] }> {
    const params = new URLSearchParams({ limit: String(limit) });
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/decisions?${params}`);
  },

  async listLessons(limit: number = 50): Promise<{ lessons: Lesson[] }> {
    const params = new URLSearchParams({ limit: String(limit) });
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/lessons?${params}`);
  },

  async recordDecision(decision: Omit<Decision, 'id' | 'timestamp'>): Promise<{ id: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/decisions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(decision),
    });
  },

  async recordOutcome(outcome: Omit<Outcome, 'id' | 'timestamp'>): Promise<{ id: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/outcomes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(outcome),
    });
  },

  async extractLesson(lesson: Omit<Lesson, 'id' | 'timestamp'>): Promise<{ id: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/lessons`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(lesson),
    });
  },

  async queryLessons(query: string, limit: number = 10): Promise<{ lessons: Lesson[]; count: number }> {
    const params = new URLSearchParams({
      q: query,
      limit: String(limit)
    });
    return fetchWithAuth(`${API_BASE}/api/v1/reflection/search?${params}`);
  },
};
