import React, { useState } from 'react';
import { Search, X } from 'lucide-react';
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

const RETRIEVE_MODES = [
  { value: 'chunk', label: 'Chunk (default)', description: 'Single matching chunk' },
  { value: 'full', label: 'Full File', description: 'Entire file content' },
];

export const CodeSearchForm: React.FC<CodeSearchFormProps> = ({ onSearch, loading }) => {
  const [query, setQuery] = useState('');
  const [selectedFileTypes, setSelectedFileTypes] = useState<string[]>([]);
  const [minRelevance, setMinRelevance] = useState(0);
  const [maxResults, setMaxResults] = useState(10);
  const [folderPath, setFolderPath] = useState('');
  const [retrieveMode, setRetrieveMode] = useState<'chunk' | 'full'>('chunk');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;

    const options: SearchOptions = {
      fileTypes: selectedFileTypes.length > 0 ? selectedFileTypes : undefined,
      minRelevanceScore: minRelevance,
      maxResults,
      folderPath: folderPath.trim() || undefined,
      retrieve: retrieveMode,
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

        {/* Folder Path Filter */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Folder Path (optional)
          </label>
          <div className="relative">
            <Input
              type="text"
              value={folderPath}
              onChange={(e) => setFolderPath(e.target.value)}
              placeholder="/path/to/folder"
              disabled={loading}
            />
            {folderPath && (
              <button
                type="button"
                onClick={() => setFolderPath('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                title="Clear folder path"
              >
                <X className="h-4 w-4" />
              </button>
            )}
          </div>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Search within a specific folder only
          </p>
        </div>

        {/* Retrieval Mode Selection */}
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
            Content Retrieval Mode
          </label>
          <select
            value={retrieveMode}
            onChange={(e) => setRetrieveMode(e.target.value as typeof retrieveMode)}
            className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-primary-500 focus:border-transparent disabled:opacity-50 disabled:cursor-not-allowed"
            disabled={loading}
          >
            {RETRIEVE_MODES.map((mode) => (
              <option key={mode.value} value={mode.value}>
                {mode.label} - {mode.description}
              </option>
            ))}
          </select>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Controls how much content to retrieve per match
          </p>
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
