import React from 'react';
import type { KnowledgeEntry } from '@/types/knowledge';

interface ArticleListProps {
  entries: KnowledgeEntry[];
  selectedEntryId: string | null;
  onSelectEntry: (entry: KnowledgeEntry) => void;
}

export const ArticleList: React.FC<ArticleListProps> = ({
  entries,
  selectedEntryId,
  onSelectEntry,
}) => {
  const getTitle = (text: string): string => {
    const firstLine = text.split('\n')[0];
    // Remove markdown heading markers
    return firstLine.replace(/^#+\s*/, '').trim();
  };

  const getPreview = (text: string): string => {
    // Remove first line (title) and get preview
    const lines = text.split('\n').slice(1);
    const preview = lines.join(' ').trim();

    // Truncate to 150 characters
    if (preview.length > 150) {
      return preview.substring(0, 150) + '...';
    }
    return preview || 'No preview available';
  };

  const formatDate = (dateString?: string): string => {
    if (!dateString) return '';
    try {
      return new Date(dateString).toLocaleDateString();
    } catch {
      return '';
    }
  };

  if (entries.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-8 text-center">
        <div className="relative mb-6">
          <div className="absolute inset-0 bg-gray-500/10 dark:bg-gray-400/10 rounded-full blur-2xl animate-pulse"></div>
          <div className="relative rounded-full bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-900 p-8 shadow-lg">
            <svg className="h-16 w-16 text-gray-400 dark:text-gray-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
            </svg>
          </div>
        </div>
        <h3 className="text-lg font-bold text-gray-900 dark:text-gray-100 mb-2">No Entries</h3>
        <p className="text-sm text-gray-600 dark:text-gray-400">
          This collection is empty
        </p>
      </div>
    );
  }

  return (
    <div className="p-4">
      <div className="mb-4 pb-3 border-b border-gray-200 dark:border-gray-700">
        <h2 className="text-lg font-bold text-gray-900 dark:text-gray-100 flex items-center gap-2">
          <span className="px-3 py-1 bg-gradient-to-r from-blue-500 to-blue-600 text-white text-sm font-bold rounded-lg shadow-md">
            {entries.length}
          </span>
          {entries.length === 1 ? 'Entry' : 'Entries'}
        </h2>
      </div>

      <div className="flex flex-col gap-3">
        {entries.map((entry) => {
          const isSelected = selectedEntryId === entry.id;

          return (
            <div
              key={entry.id}
              onClick={() => onSelectEntry(entry)}
              className={`
                p-5 rounded-xl border-2 cursor-pointer transition-all duration-200
                hover:shadow-xl hover:-translate-y-1
                ${isSelected
                  ? 'border-blue-500 bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 shadow-lg scale-[1.02]'
                  : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 hover:border-blue-400 dark:hover:border-blue-600 shadow-md'
                }
              `}
            >
              {/* Title */}
              <h3 className="text-base font-bold text-gray-900 dark:text-gray-100 mb-3 flex items-start gap-2">
                {isSelected && (
                  <span className="shrink-0 mt-0.5 text-blue-500">
                    <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 20 20">
                      <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                    </svg>
                  </span>
                )}
                <span className="flex-1">{getTitle(entry.text)}</span>
              </h3>

              {/* Preview */}
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-3 line-clamp-2 leading-relaxed">
                {getPreview(entry.text)}
              </p>

              {/* Metadata chips and date */}
              <div className="flex gap-2 flex-wrap items-center">
                {entry.createdAt && (
                  <span className="inline-flex items-center gap-1 text-xs text-gray-500 dark:text-gray-500 font-medium">
                    <svg className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
                    </svg>
                    {formatDate(entry.createdAt)}
                  </span>
                )}

                {entry.metadata && Object.keys(entry.metadata).length > 0 && (
                  <>
                    {Object.entries(entry.metadata).slice(0, 3).map(([key, value]) => (
                      <span
                        key={key}
                        className="inline-flex items-center px-2.5 py-1 rounded-lg text-xs font-semibold bg-gradient-to-r from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-600 text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-500 shadow-sm"
                      >
                        {key}: {String(value)}
                      </span>
                    ))}
                    {Object.keys(entry.metadata).length > 3 && (
                      <span className="inline-flex items-center px-2.5 py-1 rounded-lg text-xs font-semibold bg-gradient-to-r from-orange-100 to-orange-200 dark:from-orange-900/30 dark:to-orange-800/30 text-orange-700 dark:text-orange-300 border border-orange-300 dark:border-orange-600 shadow-sm">
                        +{Object.keys(entry.metadata).length - 3} more
                      </span>
                    )}
                  </>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
