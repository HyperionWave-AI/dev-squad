import { renderHook, act } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { KnowledgeProvider, useKnowledge } from '../../components/knowledge/KnowledgeLayout';
import type { KnowledgeEntry, KnowledgeCollection } from '../../types/knowledge';

describe('useKnowledge', () => {
  const mockCollections: KnowledgeCollection[] = [
    { name: 'test-collection', count: 10, category: 'Tech' },
    { name: 'another-collection', count: 5, category: 'Task' },
  ];

  const mockResults: KnowledgeEntry[] = [
    { id: '1', text: 'Test entry 1', score: 0.9 },
    { id: '2', text: 'Test entry 2', score: 0.7 },
  ];

  it('should throw error when used outside provider', () => {
    // Suppress console.error for this test
    const originalError = console.error;
    console.error = () => {};

    expect(() => {
      renderHook(() => useKnowledge());
    }).toThrow('useKnowledge must be used within a KnowledgeProvider');

    console.error = originalError;
  });

  it('should initialize with default state', () => {
    const { result } = renderHook(() => useKnowledge(), {
      wrapper: KnowledgeProvider,
    });

    expect(result.current.selectedCollection).toBe('');
    expect(result.current.results).toEqual([]);
    expect(result.current.collections).toEqual([]);
    expect(result.current.filters).toEqual({
      limit: 10,
      minScore: undefined,
    });
  });

  it('should update selectedCollection when setSelectedCollection is called', () => {
    const { result } = renderHook(() => useKnowledge(), {
      wrapper: KnowledgeProvider,
    });

    act(() => {
      result.current.setSelectedCollection('test-collection');
    });

    expect(result.current.selectedCollection).toBe('test-collection');
  });

  it('should update results when setResults is called', () => {
    const { result } = renderHook(() => useKnowledge(), {
      wrapper: KnowledgeProvider,
    });

    act(() => {
      result.current.setResults(mockResults);
    });

    expect(result.current.results).toEqual(mockResults);
  });

  it('should update collections when setCollections is called', () => {
    const { result } = renderHook(() => useKnowledge(), {
      wrapper: KnowledgeProvider,
    });

    act(() => {
      result.current.setCollections(mockCollections);
    });

    expect(result.current.collections).toEqual(mockCollections);
  });

  it('should update filters when setFilters is called', () => {
    const { result } = renderHook(() => useKnowledge(), {
      wrapper: KnowledgeProvider,
    });

    const newFilters = { limit: 20, minScore: 0.5 };

    act(() => {
      result.current.setFilters(newFilters);
    });

    expect(result.current.filters).toEqual(newFilters);
  });

  it('should maintain state across multiple updates', () => {
    const { result } = renderHook(() => useKnowledge(), {
      wrapper: KnowledgeProvider,
    });

    act(() => {
      result.current.setSelectedCollection('test-collection');
      result.current.setResults(mockResults);
      result.current.setCollections(mockCollections);
    });

    expect(result.current.selectedCollection).toBe('test-collection');
    expect(result.current.results).toEqual(mockResults);
    expect(result.current.collections).toEqual(mockCollections);
  });

  it('should clear results when setResults is called with empty array', () => {
    const { result } = renderHook(() => useKnowledge(), {
      wrapper: KnowledgeProvider,
    });

    // First set some results
    act(() => {
      result.current.setResults(mockResults);
    });

    expect(result.current.results).toEqual(mockResults);

    // Then clear them
    act(() => {
      result.current.setResults([]);
    });

    expect(result.current.results).toEqual([]);
  });
});
