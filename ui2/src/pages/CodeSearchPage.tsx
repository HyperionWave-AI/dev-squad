import React, { useState, useEffect } from 'react';
import { Code, Search, RefreshCw, Folder } from 'lucide-react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import * as Select from '@radix-ui/react-select';
import { codeIndexService } from '@/services/codeIndexService';
import type { SearchResult, IndexStatus } from '@/types/codeIndex';
import { Button } from '@/components/atoms/Button';
import { Input } from '@/components/atoms/Input';
import { Badge } from '@/components/atoms/Badge';

export function CodeSearchPage() {
  const [query, setQuery] = useState('');
  const [retrieveMode, setRetrieveMode] = useState('chunk-m');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [indexStatus, setIndexStatus] = useState<IndexStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [selectedResult, setSelectedResult] = useState<SearchResult | null>(null);

  useEffect(() => {
    loadIndexStatus();
  }, []);

  const loadIndexStatus = async () => {
    try {
      const status = await codeIndexService.getStatus();
      setIndexStatus(status);
    } catch (error) {
      console.error('Failed to load index status:', error);
    }
  };

  const handleSearch = async () => {
    if (!query.trim()) return;
    try {
      setLoading(true);
      const { results: data } = await codeIndexService.search({
        query,
        retrieve: retrieveMode as any,
        limit: 20,
      });
      setResults(data);
      if (data.length > 0) {
        setSelectedResult(data[0]);
      }
    } catch (error) {
      console.error('Search failed:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleScan = async () => {
    try {
      setLoading(true);
      await codeIndexService.triggerScan();
      await loadIndexStatus();
    } catch (error) {
      console.error('Scan failed:', error);
    } finally {
      setLoading(false);
    }
  };

  const getLanguage = (filePath: string): string => {
    const ext = filePath.split('.').pop()?.toLowerCase();
    const langMap: Record<string, string> = {
      ts: 'typescript',
      tsx: 'typescript',
      js: 'javascript',
      jsx: 'javascript',
      py: 'python',
      go: 'go',
      rs: 'rust',
      java: 'java',
      cpp: 'cpp',
      c: 'c',
      md: 'markdown',
      json: 'json',
      yaml: 'yaml',
      yml: 'yaml',
    };
    return langMap[ext || ''] || 'text';
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <div className="container mx-auto p-6 space-y-6 max-w-7xl">
        {/* Header - Glassmorphic Container */}
        <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 shadow-lg">
          <div className="flex items-center gap-3">
            <div className="relative">
              <div className="absolute inset-0 bg-gradient-to-br from-blue-400 to-cyan-500 rounded-xl blur-lg opacity-30 animate-pulse"></div>
              <div className="relative p-3 bg-gradient-to-br from-blue-500 to-cyan-600 rounded-xl shadow-xl">
                <Code className="h-8 w-8 text-white" />
              </div>
            </div>
            <div>
              <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-600 via-cyan-600 to-teal-600 bg-clip-text text-transparent">
                Code Search
              </h1>
              <p className="text-gray-600 dark:text-gray-400 mt-1">
                Semantic code search with natural language queries
              </p>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="space-y-6">
            {/* Search Panel - Glassmorphic Container */}
            <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 space-y-4 shadow-lg">
            <h3 className="font-semibold flex items-center gap-2">
              <Search className="h-5 w-5" />
              Search
            </h3>
            <Input
              placeholder="e.g., 'authentication logic', 'error handling'..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
            />
            <div className="space-y-2">
              <label className="text-sm font-medium">Retrieve Mode</label>
              <Select.Root value={retrieveMode} onValueChange={setRetrieveMode}>
                <Select.Trigger className="w-full px-3 py-2 border border-border rounded-md bg-background text-sm">
                  <Select.Value />
                </Select.Trigger>
                <Select.Portal>
                  <Select.Content className="bg-background border border-border rounded-md shadow-lg z-50">
                    <Select.Viewport className="p-1">
                      <Select.Item value="chunk-s" className="px-3 py-2 text-sm hover:bg-accent cursor-pointer rounded">
                        <Select.ItemText>Small (50 lines)</Select.ItemText>
                      </Select.Item>
                      <Select.Item value="chunk-m" className="px-3 py-2 text-sm hover:bg-accent cursor-pointer rounded">
                        <Select.ItemText>Medium (100 lines)</Select.ItemText>
                      </Select.Item>
                      <Select.Item value="chunk-l" className="px-3 py-2 text-sm hover:bg-accent cursor-pointer rounded">
                        <Select.ItemText>Large (200 lines)</Select.ItemText>
                      </Select.Item>
                      <Select.Item value="chunk-xl" className="px-3 py-2 text-sm hover:bg-accent cursor-pointer rounded">
                        <Select.ItemText>XL (400 lines)</Select.ItemText>
                      </Select.Item>
                      <Select.Item value="full" className="px-3 py-2 text-sm hover:bg-accent cursor-pointer rounded">
                        <Select.ItemText>Full File</Select.ItemText>
                      </Select.Item>
                    </Select.Viewport>
                  </Select.Content>
                </Select.Portal>
              </Select.Root>
            </div>
            <Button onClick={handleSearch} disabled={loading || !query.trim()} className="w-full">
              {loading ? 'Searching...' : 'Search'}
            </Button>
          </div>

          {indexStatus && (
            <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 space-y-4 shadow-lg">
              <div className="flex justify-between items-center">
                <h3 className="font-semibold flex items-center gap-2">
                  <Folder className="h-5 w-5" />
                  Index Status
                </h3>
                <Button onClick={handleScan} variant="outline" size="sm" disabled={loading}>
                  <RefreshCw className="h-4 w-4" />
                </Button>
              </div>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Total Files:</span>
                  <span className="font-medium">{indexStatus.totalFiles}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Folders:</span>
                  <span className="font-medium">{indexStatus.folders.length}</span>
                </div>
                {indexStatus.lastScan && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Last Scan:</span>
                    <span className="font-medium text-xs">
                      {new Date(indexStatus.lastScan).toLocaleString()}
                    </span>
                  </div>
                )}
              </div>
              {indexStatus.folders.length > 0 && (
                <div className="space-y-1">
                  <div className="text-xs font-semibold text-muted-foreground">Indexed Folders:</div>
                  {indexStatus.folders.map((folder, i) => (
                    <div key={i} className="text-xs flex justify-between items-center">
                      <span className="truncate">{folder.path}</span>
                      <Badge variant="outline" className="ml-2">{folder.fileCount}</Badge>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

          <div className="lg:col-span-2 space-y-4">
            {results.length === 0 ? (
              <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-12 text-center shadow-lg">
                <Code className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>Enter a search query to find code</p>
              </div>
            ) : (
              <>
                <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 shadow-lg">
                <h3 className="font-semibold mb-3">Results ({results.length})</h3>
                <div className="space-y-2 max-h-64 overflow-y-auto">
                  {results.map((result, i) => (
                    <button
                      key={i}
                      onClick={() => setSelectedResult(result)}
                      className={'w-full text-left p-3 rounded-md border transition-colors ' +
                        (selectedResult === result
                          ? 'border-primary bg-accent'
                          : 'border-border hover:bg-accent')
                      }
                    >
                      <div className="flex justify-between items-start mb-1">
                        <span className="text-sm font-medium truncate">{result.filePath}</span>
                        <Badge variant="outline" className="ml-2">
                          {(result.score * 100).toFixed(0)}%
                        </Badge>
                      </div>
                      <div className="text-xs text-muted-foreground">
                        Lines {result.lineStart}-{result.lineEnd}
                      </div>
                    </button>
                  ))}
                </div>
              </div>

                {selectedResult && (
                  <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg overflow-hidden shadow-lg">
                    <div className="bg-white/30 dark:bg-gray-700/30 px-4 py-2 border-b border-white/30 dark:border-gray-700/30">
                    <div className="flex justify-between items-center">
                      <span className="text-sm font-medium">{selectedResult.filePath}</span>
                      <Badge variant="outline">
                        Lines {selectedResult.lineStart}-{selectedResult.lineEnd}
                      </Badge>
                    </div>
                  </div>
                  <div className="overflow-x-auto">
                    <SyntaxHighlighter
                      language={getLanguage(selectedResult.filePath)}
                      style={vscDarkPlus}
                      showLineNumbers
                      startingLineNumber={selectedResult.lineStart}
                      customStyle={{ margin: 0, fontSize: '0.875rem' }}
                    >
                      {selectedResult.content}
                    </SyntaxHighlighter>
                  </div>
                </div>
              )}
            </>
          )}
          </div>
        </div>
      </div>
    </div>
  );
}
