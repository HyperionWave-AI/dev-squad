import React, { useState } from 'react';
import * as Select from '@radix-ui/react-select';
import { Search, ChevronDown, Check } from 'lucide-react';
import { Button } from '@atoms/Button';
import { Input } from '@atoms/Input';
import { Label } from '@atoms/Label';
import { cn } from '@/utils';
import type { KnowledgeCollection } from '@/types/knowledge';

export interface KnowledgeSearchProps {
  collections: KnowledgeCollection[];
  selectedCollection: string | null;
  onCollectionChange: (collection: string) => void;
  onSearch: (query: string, collection: string) => void;
  loading?: boolean;
}

export function KnowledgeSearch({
  collections,
  selectedCollection,
  onCollectionChange,
  onSearch,
  loading = false,
}: KnowledgeSearchProps) {
  const [query, setQuery] = useState('');
  const [limit, setLimit] = useState(10);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim() && selectedCollection) {
      onSearch(query.trim(), selectedCollection);
    }
  };

  const handleClear = () => {
    setQuery('');
  };

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
          Search Knowledge
        </h3>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Collection Selector */}
        <div className="space-y-2">
          <Label htmlFor="collection-select">Collection</Label>
          <Select.Root
            value={selectedCollection || ''}
            onValueChange={onCollectionChange}
          >
            <Select.Trigger
              id="collection-select"
              className={cn(
                'flex h-10 w-full items-center justify-between rounded-md border border-gray-300 dark:border-gray-600',
                'bg-white dark:bg-gray-800 px-3 py-2 text-sm',
                'hover:bg-gray-50 dark:hover:bg-gray-700',
                'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2',
                'disabled:cursor-not-allowed disabled:opacity-50'
              )}
              disabled={loading || collections.length === 0}
            >
              <Select.Value placeholder="Select a collection..." />
              <Select.Icon>
                <ChevronDown className="h-4 w-4 opacity-50" />
              </Select.Icon>
            </Select.Trigger>

            <Select.Portal>
              <Select.Content
                className={cn(
                  'relative z-50 min-w-[8rem] overflow-hidden rounded-md border',
                  'bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700',
                  'shadow-md animate-in fade-in-80'
                )}
                position="popper"
                sideOffset={4}
              >
                <Select.Viewport className="p-1">
                  {collections.length === 0 ? (
                    <Select.Item
                      value="none"
                      disabled
                      className="px-3 py-2 text-sm text-gray-500"
                    >
                      No collections available
                    </Select.Item>
                  ) : (
                    collections.map((col) => (
                      <Select.Item
                        key={col.name}
                        value={col.name}
                        className={cn(
                          'relative flex w-full cursor-pointer select-none items-center',
                          'rounded-sm py-2 pl-8 pr-2 text-sm outline-none',
                          'hover:bg-gray-100 dark:hover:bg-gray-700',
                          'focus:bg-gray-100 dark:focus:bg-gray-700',
                          'data-[disabled]:pointer-events-none data-[disabled]:opacity-50'
                        )}
                      >
                        <span className="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
                          <Select.ItemIndicator>
                            <Check className="h-4 w-4" />
                          </Select.ItemIndicator>
                        </span>
                        <Select.ItemText>
                          {col.name} ({col.count} entries)
                        </Select.ItemText>
                      </Select.Item>
                    ))
                  )}
                </Select.Viewport>
              </Select.Content>
            </Select.Portal>
          </Select.Root>
        </div>

        {/* Search Query Input */}
        <div className="space-y-2">
          <Label htmlFor="search-query">Search Query</Label>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
            <Input
              id="search-query"
              type="text"
              placeholder="Enter search terms..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="pl-10"
              disabled={loading}
              required
            />
          </div>
        </div>

        {/* Result Limit */}
        <div className="space-y-2">
          <Label htmlFor="result-limit">Result Limit: {limit}</Label>
          <input
            id="result-limit"
            type="range"
            min="5"
            max="20"
            step="5"
            value={limit}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer accent-blue-600"
            disabled={loading}
          />
          <div className="flex justify-between text-xs text-gray-500">
            <span>5</span>
            <span>10</span>
            <span>15</span>
            <span>20</span>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex gap-2">
          <Button
            type="submit"
            variant="primary"
            disabled={loading || !query.trim() || !selectedCollection}
            className="flex-1"
          >
            {loading ? 'Searching...' : 'Search'}
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={handleClear}
            disabled={loading || !query}
            className="px-4"
          >
            Clear
          </Button>
        </div>
      </form>

      {/* Keyboard Shortcuts Hint */}
      <div className="p-3 bg-gray-50 dark:bg-gray-900 rounded-md">
        <p className="text-xs text-gray-600 dark:text-gray-400">
          <strong>Tip:</strong> Use semantic search to find knowledge entries by meaning, not just keywords.
        </p>
      </div>
    </div>
  );
}
