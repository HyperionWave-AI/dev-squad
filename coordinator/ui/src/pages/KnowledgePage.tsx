import React, { useState, useEffect } from 'react';
import { KnowledgeSearch, KnowledgeCreate, CollectionBrowser } from '../components/knowledge';
import { knowledgeApi } from '../services/knowledgeApi';
import type { KnowledgeCollection } from '../types/knowledge';

export const KnowledgePage: React.FC = () => {
  const [collections, setCollections] = useState<KnowledgeCollection[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [selectedCollection, setSelectedCollection] = useState<string>('');
  const [showCreateForm, setShowCreateForm] = useState<boolean>(false);

  useEffect(() => {
    loadCollections();
  }, []);

  const loadCollections = async () => {
    setLoading(true);
    try {
      const response = await knowledgeApi.listCollections();
      setCollections(response.collections);
    } catch (error) {
      console.error('Failed to load collections:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCollectionSelect = (collectionName: string) => {
    setSelectedCollection(collectionName);
  };

  const handleCreateSuccess = () => {
    loadCollections(); // Refresh collections after creating new entry
    setShowCreateForm(false);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        <span className="ml-3 text-gray-600 text-lg">Loading knowledge base...</span>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50 p-6">
      <div className="max-w-7xl mx-auto space-y-6">
        {/* Page Header */}
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">Knowledge Base</h1>
            <p className="text-gray-600 mt-1">
              Search, browse, and create knowledge entries across collections
            </p>
          </div>
          <button
            onClick={() => setShowCreateForm(!showCreateForm)}
            className="px-6 py-3 bg-blue-600 text-white rounded-lg font-semibold hover:bg-blue-700 transition-colors shadow-sm"
          >
            {showCreateForm ? 'Hide Form' : '+ Create Entry'}
          </button>
        </div>

        {/* Create Form (Collapsible) */}
        {showCreateForm && (
          <div className="transition-all">
            <KnowledgeCreate
              collections={collections}
              onSuccess={handleCreateSuccess}
            />
          </div>
        )}

        {/* Main Layout: Search (left) + Collections (right) */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Search Section (2/3 width on large screens) */}
          <div className="lg:col-span-2">
            <KnowledgeSearch collections={collections} />
          </div>

          {/* Collections Browser (1/3 width on large screens) */}
          <div className="lg:col-span-1">
            <CollectionBrowser onCollectionSelect={handleCollectionSelect} />
          </div>
        </div>

        {/* Selected Collection Info */}
        {selectedCollection && (
          <div className="p-4 bg-blue-50 border-2 border-blue-200 rounded-lg">
            <p className="text-blue-800">
              <span className="font-semibold">Selected Collection:</span> {selectedCollection}
            </p>
            <p className="text-sm text-blue-600 mt-1">
              Use the search form above to search within this collection
            </p>
          </div>
        )}
      </div>
    </div>
  );
};
