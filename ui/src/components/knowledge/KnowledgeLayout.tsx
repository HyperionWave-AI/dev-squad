import React, { createContext, useContext, useState, useMemo, useCallback } from 'react';
import type { ReactNode } from 'react';
import type { KnowledgeEntry, KnowledgeCollection } from '../../types/knowledge';

// Context shape
interface KnowledgeContextValue {
  selectedCollection: string;
  setSelectedCollection: (collection: string) => void;
  results: KnowledgeEntry[];
  setResults: (results: KnowledgeEntry[]) => void;
  filters: SearchFilters;
  setFilters: (filters: SearchFilters) => void;
  collections: KnowledgeCollection[];
  setCollections: (collections: KnowledgeCollection[]) => void;
}

export interface SearchFilters {
  limit: number;
  minScore?: number;
}

// Create context with null as default (will throw if used outside provider)
const KnowledgeContext = createContext<KnowledgeContextValue | null>(null);

interface KnowledgeProviderProps {
  children: ReactNode;
}

// Provider component
export const KnowledgeProvider: React.FC<KnowledgeProviderProps> = ({ children }) => {
  const [selectedCollection, setSelectedCollectionState] = useState<string>('');
  const [results, setResultsState] = useState<KnowledgeEntry[]>([]);
  const [collections, setCollectionsState] = useState<KnowledgeCollection[]>([]);
  const [filters, setFiltersState] = useState<SearchFilters>({
    limit: 10,
    minScore: undefined,
  });

  // Memoized setters to prevent unnecessary re-renders
  const setSelectedCollection = useCallback((collection: string) => {
    setSelectedCollectionState(collection);
  }, []);

  const setResults = useCallback((newResults: KnowledgeEntry[]) => {
    setResultsState(newResults);
  }, []);

  const setFilters = useCallback((newFilters: SearchFilters) => {
    setFiltersState(newFilters);
  }, []);

  const setCollections = useCallback((newCollections: KnowledgeCollection[]) => {
    setCollectionsState(newCollections);
  }, []);

  // Memoize context value to prevent unnecessary re-renders
  const value = useMemo<KnowledgeContextValue>(
    () => ({
      selectedCollection,
      setSelectedCollection,
      results,
      setResults,
      filters,
      setFilters,
      collections,
      setCollections,
    }),
    [selectedCollection, setSelectedCollection, results, setResults, filters, setFilters, collections, setCollections]
  );

  return (
    <KnowledgeContext.Provider value={value}>
      {children}
    </KnowledgeContext.Provider>
  );
};

// Custom hook with null check
export const useKnowledge = (): KnowledgeContextValue => {
  const context = useContext(KnowledgeContext);

  if (!context) {
    throw new Error('useKnowledge must be used within a KnowledgeProvider');
  }

  return context;
};
