import React, { useState, useEffect } from 'react';
import { knowledgeApi } from '../../services/knowledgeApi';
import type { KnowledgeCollection } from '../../types/knowledge';

interface CollectionBrowserProps {
  onCollectionSelect?: (collectionName: string) => void;
}

const categoryIcons: Record<string, string> = {
  Tech: '🔧',
  Task: '📋',
  UI: '🎨',
  Ops: '⚙️',
  Other: '📚'
};

export const CollectionBrowser: React.FC<CollectionBrowserProps> = ({ onCollectionSelect }) => {
  const [collections, setCollections] = useState<KnowledgeCollection[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<string>('All');

  useEffect(() => {
    const fetchCollections = async () => {
      setLoading(true);
      setError(null);

      try {
        const response = await knowledgeApi.listCollections();
        setCollections(response.collections);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load collections');
        setCollections([]);
      } finally {
        setLoading(false);
      }
    };

    fetchCollections();
  }, []);

  const categories = ['All', ...Array.from(new Set(collections.map(c => c.category)))];
  const filteredCollections = selectedCategory === 'All'
    ? collections
    : collections.filter(c => c.category === selectedCategory);

  const handleCollectionClick = (collectionName: string) => {
    if (onCollectionSelect) {
      onCollectionSelect(collectionName);
    }
  };

  if (loading) {
    return (
      <div className="p-4 border-2 border-gray-200 rounded-lg bg-white">
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <span className="ml-3 text-gray-600">Loading collections...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 border-2 border-red-300 bg-red-50 rounded-lg">
        <p className="text-red-800 font-semibold">Error: {error}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h2 className="text-xl font-bold">Knowledge Collections</h2>

      {/* Category Tabs */}
      <div
        className="flex gap-2 border-b-2 border-gray-200 pb-2"
        role="tablist"
        aria-label="Collection categories"
      >
        {categories.map((category) => (
          <button
            key={category}
            role="tab"
            aria-selected={selectedCategory === category}
            aria-controls={`panel-${category}`}
            onClick={() => setSelectedCategory(category)}
            className={`px-4 py-2 rounded-t font-semibold transition-colors ${
              selectedCategory === category
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            {category}
          </button>
        ))}
      </div>

      {/* Collection Grid */}
      <div
        role="tabpanel"
        id={`panel-${selectedCategory}`}
        aria-labelledby={`tab-${selectedCategory}`}
        className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
      >
        {filteredCollections.length === 0 ? (
          <div className="col-span-full p-8 text-center text-gray-600 bg-gray-50 rounded-lg border-2 border-gray-200">
            No collections found in this category
          </div>
        ) : (
          filteredCollections.map((collection) => (
            <div
              key={collection.name}
              onClick={() => handleCollectionClick(collection.name)}
              className="p-4 border-2 border-gray-200 rounded-lg bg-white shadow-sm hover:shadow-md hover:border-blue-400 cursor-pointer transition-all"
            >
              {/* Collection Header */}
              <div className="flex items-start justify-between mb-2">
                <div className="flex items-center gap-2">
                  <span className="text-2xl" role="img" aria-label={collection.category}>
                    {categoryIcons[collection.category] || categoryIcons.Other}
                  </span>
                  <h3 className="font-bold text-base leading-tight">
                    {collection.name}
                  </h3>
                </div>
                <span className="px-2 py-1 bg-blue-100 text-blue-800 rounded font-semibold text-sm">
                  {collection.count}
                </span>
              </div>

              {/* Collection Category */}
              <div className="text-xs text-gray-600">
                <span className="font-semibold">Category:</span> {collection.category}
              </div>
            </div>
          ))
        )}
      </div>

      {/* Summary */}
      {filteredCollections.length > 0 && (
        <div className="p-3 bg-gray-50 rounded border border-gray-200 text-sm text-gray-700">
          <span className="font-semibold">
            {filteredCollections.length} collection{filteredCollections.length !== 1 ? 's' : ''}
          </span>
          {' · '}
          <span>
            {filteredCollections.reduce((sum, c) => sum + c.count, 0)} total entries
          </span>
        </div>
      )}
    </div>
  );
};
