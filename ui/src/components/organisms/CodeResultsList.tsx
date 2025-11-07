import React, { lazy, Suspense } from 'react';
import { Copy, Eye, FileCode } from 'lucide-react';
import { Button } from '../atoms/Button';
import { Badge } from '../atoms/Badge';
import type { CodeResult } from '../../types/codeSearch';

// Lazy load syntax highlighter to reduce initial bundle size
const SyntaxHighlighter = lazy(() =>
  import('react-syntax-highlighter').then(module => ({
    default: module.Prism
  }))
);

// Import a theme
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

interface CodeResultsListProps {
  results: CodeResult[];
  loading?: boolean;
  onInspect: (fileId: string) => void;
}

const LANGUAGE_MAP: Record<string, string> = {
  go: 'Go',
  ts: 'TypeScript',
  tsx: 'TSX',
  js: 'JavaScript',
  jsx: 'JSX',
  py: 'Python',
  rb: 'Ruby',
  java: 'Java',
  cpp: 'C++',
  c: 'C',
  rs: 'Rust',
};

export const CodeResultsList: React.FC<CodeResultsListProps> = ({
  results,
  loading,
  onInspect,
}) => {
  const handleCopyCode = (content: string) => {
    navigator.clipboard.writeText(content);
    // You could add a toast notification here
  };

  const getLanguageLabel = (lang: string): string => {
    return LANGUAGE_MAP[lang] || lang.toUpperCase();
  };

  const getRelevanceColor = (score: number): string => {
    if (score >= 0.8) return 'bg-green-500';
    if (score >= 0.6) return 'bg-blue-500';
    if (score >= 0.4) return 'bg-yellow-500';
    return 'bg-gray-400';
  };

  const getRelevanceLabel = (score: number): string => {
    if (score >= 0.8) return 'Excellent';
    if (score >= 0.6) return 'Good';
    if (score >= 0.4) return 'Fair';
    return 'Low';
  };

  if (loading) {
    return (
      <div className="space-y-4">
        {[...Array(3)].map((_, i) => (
          <div
            key={i}
            className="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-6 animate-pulse"
          >
            <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-3/4 mb-4" />
            <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-1/4 mb-4" />
            <div className="h-24 bg-gray-200 dark:bg-gray-700 rounded" />
          </div>
        ))}
      </div>
    );
  }

  if (results.length === 0) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 p-12 text-center">
        <FileCode className="h-16 w-16 text-gray-400 mx-auto mb-4" />
        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
          No results found
        </h3>
        <p className="text-gray-600 dark:text-gray-400">
          Try adjusting your search query or filters
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between mb-2">
        <p className="text-sm text-gray-600 dark:text-gray-400">
          Found {results.length} result{results.length !== 1 ? 's' : ''}
        </p>
      </div>

      {results.map((result) => (
        <div
          key={result.id}
          className="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700 overflow-hidden hover:shadow-lg transition-shadow"
        >
          {/* Header */}
          <div className="p-4 border-b border-gray-200 dark:border-gray-700">
            <div className="flex items-start justify-between gap-4">
              <div className="flex-1 min-w-0">
                <h3 className="text-sm font-mono text-gray-900 dark:text-gray-100 truncate mb-1">
                  {result.filePath}
                </h3>
                <div className="flex items-center gap-2">
                  <Badge variant="secondary" className="text-xs">
                    {getLanguageLabel(result.language)}
                  </Badge>
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    Lines {result.lineStart}-{result.lineEnd}
                  </span>
                </div>
              </div>

              <div className="flex items-center gap-2 flex-shrink-0">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleCopyCode(result.content)}
                  title="Copy code"
                >
                  <Copy className="h-4 w-4" />
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onInspect(result.id)}
                  title="Inspect file"
                >
                  <Eye className="h-4 w-4 mr-1" />
                  Inspect
                </Button>
              </div>
            </div>

            {/* Relevance Score Bar */}
            <div className="mt-3">
              <div className="flex items-center justify-between mb-1">
                <span className="text-xs font-medium text-gray-600 dark:text-gray-400">
                  Relevance
                </span>
                <span className="text-xs font-semibold text-gray-900 dark:text-gray-100">
                  {getRelevanceLabel(result.relevanceScore)} ({Math.round(result.relevanceScore * 100)}%)
                </span>
              </div>
              <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2 overflow-hidden">
                <div
                  className={`h-full ${getRelevanceColor(result.relevanceScore)} transition-all duration-300`}
                  style={{ width: `${result.relevanceScore * 100}%` }}
                />
              </div>
            </div>
          </div>

          {/* Code Content */}
          <div className="bg-gray-50 dark:bg-gray-900">
            <Suspense
              fallback={
                <div className="p-4 text-sm text-gray-500 dark:text-gray-400">
                  Loading syntax highlighting...
                </div>
              }
            >
              <SyntaxHighlighter
                language={result.language}
                style={vscDarkPlus}
                customStyle={{
                  margin: 0,
                  padding: '1rem',
                  fontSize: '0.875rem',
                  borderRadius: 0,
                }}
                showLineNumbers
                startingLineNumber={result.lineStart}
                wrapLines
              >
                {result.content}
              </SyntaxHighlighter>
            </Suspense>
          </div>
        </div>
      ))}
    </div>
  );
};
