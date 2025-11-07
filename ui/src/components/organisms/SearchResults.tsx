import { FileText, Calendar, Tag } from 'lucide-react';
import { Badge } from '@atoms/Badge';
import { Card } from '@molecules/Card';
import { cn } from '@/utils';
import type { KnowledgeEntry } from '@/types/knowledge';

export interface SearchResultsProps {
  results: KnowledgeEntry[];
  loading?: boolean;
  query?: string;
}

export function SearchResults({ results, loading = false, query = '' }: SearchResultsProps) {
  // Highlight matched text in results
  const highlightText = (text: string, query: string) => {
    if (!query.trim()) return text;

    const parts = text.split(new RegExp(`(${query})`, 'gi'));
    return parts.map((part, index) =>
      part.toLowerCase() === query.toLowerCase() ? (
        <mark key={index} className="bg-yellow-200 dark:bg-yellow-900/50 px-0.5 rounded">
          {part}
        </mark>
      ) : (
        part
      )
    );
  };

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'Unknown date';
    try {
      return new Date(dateString).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    } catch {
      return 'Invalid date';
    }
  };

  if (loading) {
    return (
      <div className="space-y-4">
        {[...Array(3)].map((_, index) => (
          <Card key={index} className="p-4 animate-pulse">
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/4"></div>
                <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-16"></div>
              </div>
              <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-full"></div>
              <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-5/6"></div>
              <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-4/6"></div>
            </div>
          </Card>
        ))}
      </div>
    );
  }

  if (results.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
        <div className="rounded-full bg-gray-100 dark:bg-gray-800 p-6 mb-4">
          <FileText className="h-12 w-12 text-gray-400 dark:text-gray-600" />
        </div>
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
          No results found
        </h3>
        <p className="text-sm text-gray-500 dark:text-gray-400 max-w-sm">
          {query
            ? `No knowledge entries match your search query "${query}".`
            : 'Enter a search query to find knowledge entries.'}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Results Header */}
      <div className="flex items-center justify-between pb-2 border-b border-gray-200 dark:border-gray-700">
        <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
          Found {results.length} result{results.length !== 1 ? 's' : ''}
        </h3>
        {query && (
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Search: <span className="font-medium">{query}</span>
          </p>
        )}
      </div>

      {/* Results List */}
      <div className="space-y-3">
        {results.map((result) => (
          <Card
            key={result.id}
            className={cn(
              'p-4 hover:shadow-md transition-shadow cursor-pointer',
              'border-l-4',
              result.score && result.score > 0.8
                ? 'border-l-green-500'
                : result.score && result.score > 0.6
                ? 'border-l-blue-500'
                : 'border-l-gray-300'
            )}
          >
            <div className="space-y-3">
              {/* Header with Score */}
              <div className="flex items-start justify-between gap-2">
                <div className="flex items-center gap-2 flex-1 min-w-0">
                  <FileText className="h-4 w-4 text-gray-400 shrink-0" />
                  <span className="text-xs text-gray-500 dark:text-gray-400 truncate">
                    ID: {result.id}
                  </span>
                </div>
                {result.score !== undefined && (
                  <Badge
                    variant={
                      result.score > 0.8
                        ? 'success'
                        : result.score > 0.6
                        ? 'primary'
                        : 'secondary'
                    }
                    className="shrink-0"
                  >
                    {(result.score * 100).toFixed(0)}%
                  </Badge>
                )}
              </div>

              {/* Content Preview */}
              <div className="prose prose-sm dark:prose-invert max-w-none">
                <p className="text-sm text-gray-700 dark:text-gray-300 line-clamp-4">
                  {highlightText(
                    result.text.length > 300
                      ? result.text.substring(0, 300) + '...'
                      : result.text,
                    query
                  )}
                </p>
              </div>

              {/* Metadata Footer */}
              <div className="flex items-center gap-4 pt-2 border-t border-gray-100 dark:border-gray-700">
                {result.createdAt && (
                  <div className="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
                    <Calendar className="h-3 w-3" />
                    <span>{formatDate(result.createdAt)}</span>
                  </div>
                )}

                {result.metadata && Object.keys(result.metadata).length > 0 && (
                  <div className="flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400">
                    <Tag className="h-3 w-3" />
                    <span>{Object.keys(result.metadata).length} metadata fields</span>
                  </div>
                )}
              </div>

              {/* Display metadata tags if available */}
              {result.metadata && (
                <div className="flex flex-wrap gap-1">
                  {Object.entries(result.metadata)
                    .slice(0, 5)
                    .map(([key, value]) => (
                      <Badge key={key} variant="outline" className="text-xs">
                        {key}: {String(value).substring(0, 20)}
                        {String(value).length > 20 ? '...' : ''}
                      </Badge>
                    ))}
                  {Object.keys(result.metadata).length > 5 && (
                    <Badge variant="outline" className="text-xs">
                      +{Object.keys(result.metadata).length - 5} more
                    </Badge>
                  )}
                </div>
              )}
            </div>
          </Card>
        ))}
      </div>
    </div>
  );
}
