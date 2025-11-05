/**
 * Global test setup for Vitest
 * Sets up testing-library matchers, mocks, and global utilities
 */
import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach, vi } from 'vitest';

// Cleanup after each test
afterEach(() => {
  cleanup();
});

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn((key: string) => {
    return localStorageMock.store[key] || null;
  }),
  setItem: vi.fn((key: string, value: string) => {
    localStorageMock.store[key] = value;
  }),
  removeItem: vi.fn((key: string) => {
    delete localStorageMock.store[key];
  }),
  clear: vi.fn(() => {
    localStorageMock.store = {};
  }),
  store: {} as Record<string, string>
};

global.localStorage = localStorageMock as unknown as Storage;

// Mock window.matchMedia for theme tests
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

// Mock fetch for API calls
global.fetch = vi.fn((url: string | URL | Request, init?: RequestInit) => {
  const urlStr = typeof url === 'string' ? url : url instanceof URL ? url.toString() : url.url;

  // Default mock responses for common API endpoints
  if (urlStr.includes('/api/mcp/tools/list')) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: async () => ({ tools: [] }),
    } as Response);
  }

  if (urlStr.includes('/api/v1/tasks')) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: async () => ({ tasks: [] }),
    } as Response);
  }

  if (urlStr.includes('/api/knowledge/collections')) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: async () => ({ collections: [] }),
    } as Response);
  }

  if (urlStr.includes('/bridge-health')) {
    return Promise.resolve({
      ok: true,
      status: 200,
      json: async () => ({ status: 'healthy' }),
    } as Response);
  }

  // Default fallback
  return Promise.resolve({
    ok: true,
    status: 200,
    json: async () => ({}),
  } as Response);
});
