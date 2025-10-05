import React, { useState } from 'react';
import { knowledgeApi } from '../../services/knowledgeApi';
import type { KnowledgeEntry, KnowledgeCollection } from '../../types/knowledge';

interface KnowledgeSearchProps {
  collections: KnowledgeCollection[];
}

export const KnowledgeSearch: React.FC<KnowledgeSearchProps> = ({ collections }) => {
  const [selectedCollection, setSelectedCollection] = useState<string>('');
  const [query, setQuery] = useState<string>('');
  const [results, setResults] = useState<KnowledgeEntry[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!selectedCollection || !query.trim()) {
      setError('Please select a collection and enter a search query');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await knowledgeApi.searchKnowledge({
        collection: selectedCollection,
        query: query.trim(),
        limit: 10
      });
      setResults(response.results);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const handleClear = () => {
    setQuery('');
    setResults([]);
    setError(null);
  };

  return (
    <div className="space-y-4">
      {/* Search Form */}
      <form onSubmit={handleSearch} className="p-4 border-2 rounded-lg bg-white shadow-sm">
        <div className="space-y-3">
          {/* Collection Select */}
          <div>
            <label htmlFor="collection" className="block text-sm font-semibold mb-1">
              Collection
            </label>
            <select
              id="collection"
              value={selectedCollection}
              onChange={(e) => setSelectedCollection(e.target.value)}
              className="w-full p-2 border-2 border-gray-300 rounded focus:border-blue-500 focus:outline-none"
              required
            >
              <option value="">Select a collection...</option>
              {collections.map((col) => (
                <option key={col.name} value={col.name}>
                  {col.name} ({col.count} entries)
                </option>
              ))}
            </select>
          </div>

          {/* Search Input */}
          <div>
            <label htmlFor="query" className="block text-sm font-semibold mb-1">
              Search Query
            </label>
            <input
              id="query"
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Enter search terms..."
              className="w-full p-2 border-2 border-gray-300 rounded focus:border-blue-500 focus:outline-none"
              required
            />
          </div>

          {/* Action Buttons */}
          <div className="flex gap-2">
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 bg-blue-600 text-white rounded font-semibold hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
            >
              {loading ? 'Searching...' : 'Search'}
            </button>
            <button
              type="button"
              onClick={handleClear}
              className="px-4 py-2 bg-gray-200 text-gray-800 rounded font-semibold hover:bg-gray-300 transition-colors"
            >
              Clear
            </button>
          </div>
        </div>
      </form>

      {/* Error Display */}
      {error && (
        <div className="p-4 border-2 border-red-300 bg-red-50 rounded-lg">
          <p className="text-red-800 font-semibold">Error: {error}</p>
        </div>
      )}

      {/* Results List */}
      {results.length > 0 && (
        <div className="space-y-2">
          <h3 className="text-lg font-bold">
            Search Results ({results.length})
          </h3>
          {results.map((entry) => (
            <div
              key={entry.id}
              className="p-4 border-2 border-gray-200 rounded-lg bg-white shadow-sm hover:shadow-md transition-shadow"
            >
              <div className="flex justify-between items-start mb-2">
                <div className="flex-1">
                  <p className="text-sm mb-2 whitespace-pre-wrap">{entry.text}</p>

                  {entry.metadata && Object.keys(entry.metadata).length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-2">
                      {Object.entries(entry.metadata).map(([key, value]) => (
                        <span
                          key={key}
                          className="px-2 py-0.5 bg-gray-100 rounded text-xs"
                        >
                          <span className="font-semibold">{key}:</span> {String(value)}
                        </span>
                      ))}
                    </div>
                  )}
                </div>

                {entry.score !== undefined && (
                  <div className="ml-4 px-3 py-1 bg-green-100 text-green-800 rounded font-semibold text-sm">
                    Score: {entry.score.toFixed(3)}
                  </div>
                )}
              </div>

              {entry.createdAt && (
                <div className="text-xs text-gray-600 mt-2">
                  Created: {new Date(entry.createdAt).toLocaleString()}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* No Results */}
      {!loading && results.length === 0 && query && !error && (
        <div className="p-4 border-2 border-gray-200 bg-gray-50 rounded-lg text-center">
          <p className="text-gray-600">No results found for "{query}"</p>
        </div>
      )}
    </div>
  );
};
