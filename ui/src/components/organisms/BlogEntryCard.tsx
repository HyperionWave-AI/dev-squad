import React from 'react';
import { Sparkles, Calendar, ChevronDown, ChevronUp } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import { cn } from '@/utils';

interface BlogEntryCardProps {
  id: string;
  title: string;
  content: string;
  date: string;
  isExpanded: boolean;
  onToggleExpand: () => void;
  className?: string;
}

export const BlogEntryCard: React.FC<BlogEntryCardProps> = ({
  title,
  content,
  date,
  isExpanded,
  onToggleExpand,
  className,
}) => {
  // Format date relative to now
  const formatRelativeDate = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffDays === 0) return 'Today';
    if (diffDays === 1) return 'Yesterday';
    if (diffDays < 7) return `${diffDays} days ago`;
    if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
    if (diffDays < 365) return `${Math.floor(diffDays / 30)} months ago`;
    return date.toLocaleDateString();
  };

  return (
    <div
      className={cn(
        // Glassmorphic card
        'backdrop-blur-md bg-white/70 dark:bg-gray-800/70',
        'border border-white/30 dark:border-gray-700/30',
        'rounded-lg shadow-lg',
        'transition-all duration-300',
        'hover:shadow-xl hover:shadow-purple-500/20 dark:hover:shadow-purple-500/30',
        'hover:border-purple-300/50 dark:hover:border-purple-500/50',
        className
      )}
    >
      {/* Gradient accent bar */}
      <div className="h-1 bg-gradient-to-r from-pink-500 to-purple-600 rounded-t-lg" />

      {/* Header */}
      <div className="p-5 pb-4 border-b border-white/20 dark:border-gray-700/20">
        <div className="flex items-start justify-between gap-4">
          <div className="flex items-start gap-3 flex-1 min-w-0">
            <div className="relative flex-shrink-0">
              <div className="absolute inset-0 rounded-lg blur-md opacity-20 bg-gradient-to-r from-pink-500 to-purple-500" />
              <div className="relative p-2 rounded-lg bg-gradient-to-r from-pink-500 to-purple-600">
                <Sparkles className="h-5 w-5 text-white" />
              </div>
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
                {title}
              </h3>
              <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                <Calendar className="h-4 w-4" />
                <span>{formatRelativeDate(date)}</span>
              </div>
            </div>
          </div>

          {/* Expand/Collapse Button */}
          <button
            onClick={onToggleExpand}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-gradient-to-r from-pink-500 to-purple-600 text-white font-medium hover:from-pink-600 hover:to-purple-700 transition-all shadow-lg shadow-purple-500/30 whitespace-nowrap"
          >
            {isExpanded ? (
              <>
                <span>Collapse</span>
                <ChevronUp className="h-4 w-4" />
              </>
            ) : (
              <>
                <span>Expand Full</span>
                <ChevronDown className="h-4 w-4" />
              </>
            )}
          </button>
        </div>
      </div>

      {/* Content */}
      <div
        className={cn(
          'overflow-y-auto transition-all duration-300',
          isExpanded ? 'max-h-[80vh]' : 'max-h-[50vh]'
        )}
      >
        <div className="p-6 prose prose-sm dark:prose-invert max-w-none">
          <ReactMarkdown
            components={{
              h1: ({node, ...props}) => <h1 className="text-3xl font-bold mb-4 bg-clip-text text-transparent bg-gradient-to-r from-pink-600 to-purple-600 dark:from-pink-400 dark:to-purple-400" {...props} />,
              h2: ({node, ...props}) => <h2 className="text-2xl font-bold mt-6 mb-3 text-gray-900 dark:text-white" {...props} />,
              h3: ({node, ...props}) => <h3 className="text-xl font-semibold mt-4 mb-2 text-gray-800 dark:text-gray-100" {...props} />,
              p: ({node, ...props}) => <p className="mb-3 text-gray-700 dark:text-gray-300 leading-relaxed" {...props} />,
              ul: ({node, ...props}) => <ul className="list-disc list-inside mb-3 space-y-1 text-gray-700 dark:text-gray-300" {...props} />,
              ol: ({node, ...props}) => <ol className="list-decimal list-inside mb-3 space-y-1 text-gray-700 dark:text-gray-300" {...props} />,
              strong: ({node, ...props}) => <strong className="font-bold text-gray-900 dark:text-white" {...props} />,
              code: ({node, ...props}) => <code className="px-1.5 py-0.5 bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300 rounded text-sm" {...props} />,
              hr: ({node, ...props}) => <hr className="my-6 border-gray-200 dark:border-gray-700" {...props} />,
            }}
          >
            {content}
          </ReactMarkdown>
        </div>
      </div>
    </div>
  );
};
