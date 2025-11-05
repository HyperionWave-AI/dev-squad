import React, { useState, useEffect, lazy, Suspense } from 'react';
import { X, Copy, Download, FileCode, AlertCircle } from 'lucide-react';
import { Button } from '../atoms/Button';
import { Badge } from '../atoms/Badge';
import { codeIndexService } from '../../services/codeIndexService';

// Lazy load syntax highlighter
const SyntaxHighlighter = lazy(() =>
  import('react-syntax-highlighter').then(module => ({
    default: module.Prism
  }))
);

import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

interface FileInspectorProps {
  fileId: string;
  onClose: () => void;
}

interface FileData {
  path: string;
  content: string;
  language: string;
  size: number;
  lines: number;
}

export const FileInspector: React.FC<FileInspectorProps> = ({ fileId, onClose }) => {
  const [fileData, setFileData] = useState<FileData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadFileContent();
  }, [fileId]);

  const loadFileContent = async () => {
    setLoading(true);
    setError(null);

    try {
      // Note: API doesn't have a dedicated getFile endpoint yet
      // For now, we'll show a placeholder message
      // In a real implementation, this would fetch the file via the API
      const data = {
        path: fileId,
        content: '// File content loading not yet implemented in API\n// This would show the full file content when the API endpoint is ready',
      };

      if (!data) {
        throw new Error('File not found');
      }

      // Detect language from file extension
      const extension = data.path.split('.').pop()?.toLowerCase() || 'text';
      const languageMap: Record<string, string> = {
        'ts': 'typescript',
        'tsx': 'tsx',
        'js': 'javascript',
        'jsx': 'jsx',
        'go': 'go',
        'py': 'python',
        'rb': 'ruby',
        'java': 'java',
        'cpp': 'cpp',
        'c': 'c',
        'rs': 'rust',
        'html': 'html',
        'css': 'css',
        'json': 'json',
        'md': 'markdown',
      };

      setFileData({
        path: data.path || fileId,
        content: data.content || 'No content available',
        language: languageMap[extension] || 'text',
        size: new Blob([data.content || '']).size,
        lines: (data.content || '').split('\n').length,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load file');
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = () => {
    if (fileData) {
      navigator.clipboard.writeText(fileData.content);
    }
  };

  const handleDownload = () => {
    if (!fileData) return;

    const blob = new Blob([fileData.content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = fileData.path.split('/').pop() || 'file.txt';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  return (
    <div className="fixed inset-0 z-50 flex">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Drawer */}
      <div className="relative ml-auto w-full max-w-4xl bg-white dark:bg-gray-800 shadow-2xl flex flex-col animate-slide-in-right">
        {/* Header */}
        <div className="flex-shrink-0 border-b border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 sticky top-0 z-10">
          <div className="p-4 flex items-start justify-between gap-4">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-2">
                <FileCode className="h-5 w-5 text-gray-500 flex-shrink-0" />
                <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 truncate">
                  {fileData?.path || 'Loading...'}
                </h2>
              </div>
              {fileData && (
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant="secondary" className="text-xs">
                    {fileData.language.toUpperCase()}
                  </Badge>
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    {formatFileSize(fileData.size)}
                  </span>
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    {fileData.lines.toLocaleString()} lines
                  </span>
                </div>
              )}
            </div>

            <div className="flex items-center gap-2 flex-shrink-0">
              {fileData && (
                <>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleCopy}
                    title="Copy content"
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleDownload}
                    title="Download file"
                  >
                    <Download className="h-4 w-4" />
                  </Button>
                </>
              )}
              <Button
                variant="ghost"
                size="icon"
                onClick={onClose}
                title="Close"
              >
                <X className="h-5 w-5" />
              </Button>
            </div>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto bg-gray-50 dark:bg-gray-900">
          {loading ? (
            <div className="flex items-center justify-center h-full">
              <div className="text-center">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500 mx-auto mb-4" />
                <p className="text-gray-600 dark:text-gray-400">Loading file...</p>
              </div>
            </div>
          ) : error ? (
            <div className="flex items-center justify-center h-full p-8">
              <div className="text-center max-w-md">
                <AlertCircle className="h-12 w-12 text-red-500 mx-auto mb-4" />
                <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
                  Failed to Load File
                </h3>
                <p className="text-gray-600 dark:text-gray-400 mb-4">{error}</p>
                <Button variant="outline" onClick={loadFileContent}>
                  Try Again
                </Button>
              </div>
            </div>
          ) : fileData ? (
            <Suspense
              fallback={
                <div className="p-8 text-center text-gray-500 dark:text-gray-400">
                  Loading syntax highlighting...
                </div>
              }
            >
              <SyntaxHighlighter
                language={fileData.language}
                style={vscDarkPlus}
                customStyle={{
                  margin: 0,
                  padding: '1.5rem',
                  fontSize: '0.875rem',
                  borderRadius: 0,
                  height: '100%',
                }}
                showLineNumbers
                wrapLines
                lineNumberStyle={{
                  minWidth: '3em',
                  paddingRight: '1em',
                  color: '#858585',
                  textAlign: 'right',
                }}
              >
                {fileData.content}
              </SyntaxHighlighter>
            </Suspense>
          ) : null}
        </div>

        {/* Footer Info */}
        {fileData && (
          <div className="flex-shrink-0 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 px-4 py-2">
            <p className="text-xs text-gray-500 dark:text-gray-400 text-center">
              Press <kbd className="px-1.5 py-0.5 text-xs font-semibold text-gray-800 bg-gray-100 border border-gray-200 rounded dark:bg-gray-700 dark:text-gray-100 dark:border-gray-600">Esc</kbd> to close
            </p>
          </div>
        )}
      </div>
    </div>
  );
};
