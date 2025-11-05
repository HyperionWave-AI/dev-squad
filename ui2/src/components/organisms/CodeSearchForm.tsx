import React, { useState } from 'react';
import { Search } from 'lucide-react';
import { Button } from '../atoms/Button';
import { Input } from '../atoms/Input';
import { Badge } from '../atoms/Badge';
import type { SearchOptions } from '../../types/codeSearch';

interface CodeSearchFormProps {
  onSearch: (query: string, options: SearchOptions) => void;
  loading?: boolean;
}

const FILE_TYPES = [
  { id: 'go', label: 'Go', extension: '.go' },
  { id: 'ts', label: 'TypeScript', extension: '.ts' },
  { id: 'tsx', label: 'TSX', extension: '.tsx' },
  { id: 'py', label: 'Python', extension: '.py' },
  { id: 'js', label: 'JavaScript', extension: '.js' },
];

export const CodeSearchForm: React.FC<CodeSearchFormProps> = ({ onSearch, loading }) => {
  const [query, setQuery] = useState('');
  const [selectedFileTypes, setSelectedFileTypes] = useState<string[]>([]);
  const [minRelevance, setMinRelevance] = useState(0);
  const [maxResults, setMaxResults] = useState(10);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;

    const options: SearchOptions = {
      fileTypes: selectedFileTypes.length > 0 ? selectedFileTypes : undefined,
      minRelevanceScore: minRelevance,
      maxResults,
    };

    onSearch(query, options);
  };

  const toggleFileType = (typeId: string) => {
    setSelectedFileTypes(prev =>
      prev.includes(typeId)
        ? prev.filter(t => t !== typeId)
        : [...prev, typeId]
    );
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-6">
      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Search Input */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Search Query
          </label>
          <div className="relative">
            <Input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search code semantically... (e.g., authentication logic)"
              className="pr-10"
              disabled={loading}
            />
            <Search className="absolute right-3 top-1/2 -translate-y-1/2 h-5 w-5 text-gray-400" />
          </div>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Use natural language to describe what you're looking for
          </p>
        </div>

        {/* File Type Filters */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            File Types
          </label>
          <div className="flex flex-wrap gap-2">
            {FILE_TYPES.map((type) => (
              <Badge
                key={type.id}
                variant={selectedFileTypes.includes(type.id) ? 'primary' : 'outline'}
                className="cursor-pointer hover:opacity-80 transition-opacity"
                onClick={() => toggleFileType(type.id)}
              >
                {type.label}
              </Badge>
            ))}
          </div>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            {selectedFileTypes.length === 0
              ? 'All file types'
              : `${selectedFileTypes.length} type${selectedFileTypes.length > 1 ? 's' : ''} selected`}
          </p>
        </div>

        {/* Min Relevance Slider */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Minimum Relevance Score: {minRelevance.toFixed(2)}
          </label>
          <input
            type="range"
            min="0"
            max="1"
            step="0.01"
            value={minRelevance}
            onChange={(e) => setMinRelevance(parseFloat(e.target.value))}
            className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700 accent-primary-500"
            disabled={loading}
          />
          <div className="flex justify-between text-xs text-gray-500 dark:text-gray-400">
            <span>0.00 (All results)</span>
            <span>1.00 (Perfect match)</span>
          </div>
        </div>

        {/* Max Results Slider */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Maximum Results: {maxResults}
          </label>
          <input
            type="range"
            min="1"
            max="50"
            step="1"
            value={maxResults}
            onChange={(e) => setMaxResults(parseInt(e.target.value))}
            className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700 accent-primary-500"
            disabled={loading}
          />
          <div className="flex justify-between text-xs text-gray-500 dark:text-gray-400">
            <span>1</span>
            <span>50</span>
          </div>
        </div>

        {/* Submit Button */}
        <Button
          type="submit"
          variant="primary"
          className="w-full"
          disabled={loading || !query.trim()}
        >
          {loading ? (
            <>
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2" />
              Searching...
            </>
          ) : (
            <>
              <Search className="h-4 w-4 mr-2" />
              Search Code
            </>
          )}
        </Button>
      </form>
    </div>
  );
};
