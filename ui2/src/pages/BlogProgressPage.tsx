import { useState, useEffect } from 'react';
import { Sparkles, RefreshCw, Calendar, TrendingUp } from 'lucide-react';
import { Button } from '@atoms/Button';
import { BlogEntryCard } from '@organisms/BlogEntryCard';
import { PageHeader } from '@organisms/PageHeader';
import { knowledgeService, type KnowledgeEntry } from '@/services/knowledgeService';

export const BlogProgressPage = () => {
  const [entries, setEntries] = useState<KnowledgeEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [expandedEntry, setExpandedEntry] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Fetch blog entries on mount
  useEffect(() => {
    loadEntries();
  }, []);

  const loadEntries = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await knowledgeService.getEntries('progress-blog', 100);
      // Sort by date descending (newest first)
      const sortedEntries = (response.entries || []).sort((a, b) => {
        const dateA = new Date(a.createdAt || 0).getTime();
        const dateB = new Date(b.createdAt || 0).getTime();
        return dateB - dateA;
      });
      setEntries(sortedEntries);
    } catch (err) {
      console.error('Failed to load blog entries:', err);
      // If collection doesn't exist yet, treat as empty (not an error)
      const errorMsg = err instanceof Error ? err.message : '';
      if (errorMsg.includes('not found') ||
          errorMsg.includes('does not exist') ||
          errorMsg.includes('Failed to browse knowledge base')) {
        // Collection doesn't exist yet - this is normal, just show empty state
        setEntries([]);
        setError(null);
      } else {
        setError(errorMsg || 'Failed to load entries');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleGenerateEntry = async () => {
    try {
      setGenerating(true);
      setError(null);
      const result = await knowledgeService.generateBlogEntry();
      // Refresh the list
      await loadEntries();
      // Expand the newly created entry
      if (result.entry) {
        setExpandedEntry(result.entry.id);
      }
    } catch (err) {
      console.error('Failed to generate blog entry:', err);
      const errorMsg = err instanceof Error ? err.message : 'Failed to generate entry';

      // Provide helpful error messages
      if (errorMsg.includes('404') || errorMsg.includes('Not Found')) {
        setError('✨ AI blog generation endpoint is still being set up. Our AI writer will be ready soon!');
      } else if (errorMsg.includes('Failed to fetch')) {
        setError('Cannot connect to backend. Please ensure the server is running.');
      } else {
        setError(errorMsg);
      }
    } finally {
      setGenerating(false);
    }
  };

  const handleToggleExpand = (entryId: string) => {
    setExpandedEntry(expandedEntry === entryId ? null : entryId);
  };

  // Extract title from markdown (first # heading or fallback)
  const extractTitle = (text: string): string => {
    const match = text.match(/^#\s+(.+)$/m);
    if (match) return match[1];
    return 'Progress Update';
  };

  return (
    <div className="h-full flex flex-col bg-gradient-to-br from-purple-50/50 via-pink-50/30 to-blue-50/50 dark:from-gray-900 dark:via-purple-900/10 dark:to-gray-900">
      {/* Glassmorphic Header */}
      <div className="flex-none p-6 pb-0">
        <PageHeader
          title="AI Progress Blog"
          description="Let AI craft beautiful progress reports from your completed tasks"
          icon={<Sparkles className="h-8 w-8" />}
          gradientFrom="#ec4899"
          gradientTo="#8b5cf6"
        />
      </div>

      {/* Action Bar */}
      <div className="flex-none p-6">
        <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-4 shadow-lg">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="relative">
                <div className="absolute inset-0 rounded-full blur-md opacity-30 animate-pulse bg-gradient-to-r from-pink-500 to-purple-500" />
                <div className="relative p-2 rounded-full bg-gradient-to-r from-pink-500 to-purple-500">
                  <TrendingUp className="h-5 w-5 text-white" />
                </div>
              </div>
              <div>
                <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
                  {entries.length === 0 ? 'Ready to create your first report' : `${entries.length} progress ${entries.length === 1 ? 'report' : 'reports'}`}
                </h3>
                <p className="text-xs text-gray-600 dark:text-gray-400">
                  AI will analyze completed tasks and create a beautiful summary
                </p>
              </div>
            </div>
            <Button
              onClick={handleGenerateEntry}
              disabled={generating}
              variant="primary"
              className="flex items-center gap-2 bg-gradient-to-r from-pink-500 to-purple-600 hover:from-pink-600 hover:to-purple-700 shadow-lg shadow-purple-500/50"
            >
              {generating ? (
                <>
                  <RefreshCw className="h-5 w-5 animate-spin" />
                  AI is writing...
                </>
              ) : (
                <>
                  <Sparkles className="h-5 w-5" />
                  Create Progress Blogpost
                </>
              )}
            </Button>
          </div>
        </div>
      </div>

      {/* Success/Error messages */}
      {error && (
        <div className="flex-none mx-6 mb-4">
          <div className="backdrop-blur-md bg-red-50/90 dark:bg-red-900/30 border border-red-200/50 dark:border-red-800/50 rounded-lg p-4 shadow-lg">
            <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
          </div>
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-auto px-6 pb-6">
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="relative">
              <div className="absolute inset-0 rounded-full blur-md opacity-30 animate-pulse bg-gradient-to-r from-pink-500 to-purple-500" />
              <RefreshCw className="h-8 w-8 animate-spin text-purple-600 dark:text-purple-400 relative" />
            </div>
          </div>
        ) : entries.length === 0 ? (
          <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-12 shadow-lg">
            <div className="flex flex-col items-center justify-center text-center max-w-lg mx-auto">
              <div className="relative mb-6">
                <div className="absolute inset-0 rounded-2xl blur-xl opacity-30 animate-pulse bg-gradient-to-r from-pink-500 to-purple-500" />
                <div className="relative p-6 rounded-2xl bg-gradient-to-r from-pink-500 to-purple-600 shadow-xl">
                  <Sparkles className="h-16 w-16 text-white" />
                </div>
              </div>
              <h3 className="text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-pink-600 to-purple-600 dark:from-pink-400 dark:to-purple-400 mb-3">
                Let AI Tell Your Story
              </h3>
              <p className="text-gray-600 dark:text-gray-400 mb-6 text-lg">
                Click "Create Progress Blogpost" and our AI will analyze your completed tasks to craft a beautiful, insightful progress report
              </p>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 w-full mt-4">
                <div className="backdrop-blur-sm bg-white/50 dark:bg-gray-700/50 rounded-lg p-4 border border-white/30 dark:border-gray-600/30">
                  <div className="text-3xl mb-2">🤖</div>
                  <p className="text-sm font-medium text-gray-900 dark:text-white">AI-Powered</p>
                  <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">Smart analysis of your work</p>
                </div>
                <div className="backdrop-blur-sm bg-white/50 dark:bg-gray-700/50 rounded-lg p-4 border border-white/30 dark:border-gray-600/30">
                  <div className="text-3xl mb-2">✨</div>
                  <p className="text-sm font-medium text-gray-900 dark:text-white">Beautiful Reports</p>
                  <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">Well-formatted summaries</p>
                </div>
                <div className="backdrop-blur-sm bg-white/50 dark:bg-gray-700/50 rounded-lg p-4 border border-white/30 dark:border-gray-600/30">
                  <div className="text-3xl mb-2">📊</div>
                  <p className="text-sm font-medium text-gray-900 dark:text-white">Insights</p>
                  <p className="text-xs text-gray-600 dark:text-gray-400 mt-1">Discover patterns & trends</p>
                </div>
              </div>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {entries.map((entry) => (
              <BlogEntryCard
                key={entry.id}
                id={entry.id}
                title={extractTitle(entry.text)}
                content={entry.text}
                date={entry.createdAt || new Date().toISOString()}
                isExpanded={expandedEntry === entry.id}
                onToggleExpand={() => handleToggleExpand(entry.id)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
};
