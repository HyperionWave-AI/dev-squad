import React, { useState } from 'react';
import type { KnowledgeEntry } from '../types/coordinator';
import { mcpClient } from '../services/mcpClient';

export const KnowledgeBrowser: React.FC = () => {
  const [query, setQuery] = useState('');
  const [collection, setCollection] = useState('');
  const [results, setResults] = useState<KnowledgeEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const collections = [
    'All Collections',
    'task',
    'adr',
    'data-contracts',
    'technical-knowledge',
    'workflow-context',
    'team-coordination'
  ];

  const handleSearch = async () => {
    if (!query.trim()) return;

    try {
      setLoading(true);
      setError(null);

      const entries = await mcpClient.queryKnowledge({
        collection: collection === 'All Collections' ? undefined : collection,
        query,
        limit: 20
      });

      setResults(entries);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
      console.error('Search error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-gray-800 mb-2">Knowledge Browser</h2>
        <p className="text-gray-600 text-sm">
          Search across knowledge collections
        </p>
      </div>

      <div className="bg-white p-4 rounded-lg border shadow-sm space-y-4">
        <div className="flex gap-2">
          <select
            value={collection}
            onChange={(e) => setCollection(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {collections.map((col) => (
              <option key={col} value={col}>
                {col}
              </option>
            ))}
          </select>

          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder="Search knowledge base..."
            className="flex-1 px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          />

          <button
            onClick={handleSearch}
            disabled={loading || !query.trim()}
            className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? '🔍 Searching...' : '🔍 Search'}
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <div className="flex items-center gap-2">
            <span className="text-xl">❌</span>
            <p className="text-red-600 text-sm">{error}</p>
          </div>
        </div>
      )}

      {results.length > 0 && (
        <div className="space-y-3">
          <h3 className="font-semibold text-gray-700">
            {results.length} result{results.length !== 1 ? 's' : ''} found
          </h3>

          {results.map((entry) => (
            <div
              key={entry.id}
              className="bg-white p-4 rounded-lg border shadow-sm hover:shadow-md transition-shadow"
            >
              <div className="flex justify-between items-start mb-2">
                <span className="px-2 py-1 bg-blue-100 text-blue-800 rounded text-xs font-semibold">
                  {entry.collection}
                </span>
                <span className="text-xs text-gray-500">
                  {new Date(entry.createdAt).toLocaleDateString()}
                </span>
              </div>

              <p className="text-sm text-gray-800 mb-2 whitespace-pre-wrap">
                {entry.text}
              </p>

              <div className="flex items-center justify-between">
                {entry.tags.length > 0 && (
                  <div className="flex gap-1 flex-wrap">
                    {entry.tags.map((tag) => (
                      <span
                        key={tag}
                        className="px-2 py-0.5 bg-gray-100 text-gray-600 rounded text-xs"
                      >
                        {tag}
                      </span>
                    ))}
                  </div>
                )}

                <span className="text-xs text-gray-500">
                  by {entry.createdBy}
                </span>
              </div>

              {entry.metadata && Object.keys(entry.metadata).length > 0 && (
                <div className="mt-2 pt-2 border-t border-gray-100">
                  <details className="text-xs text-gray-600">
                    <summary className="cursor-pointer hover:text-gray-800">
                      Metadata
                    </summary>
                    <pre className="mt-1 p-2 bg-gray-50 rounded overflow-x-auto">
                      {JSON.stringify(entry.metadata, null, 2)}
                    </pre>
                  </details>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {!loading && results.length === 0 && query && (
        <div className="text-center py-12 bg-gray-50 rounded-lg">
          <div className="text-4xl mb-2">🔍</div>
          <p className="text-gray-600">No results found for "{query}"</p>
        </div>
      )}
    </div>
  );
};