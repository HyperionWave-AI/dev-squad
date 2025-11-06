import React, { useState, useEffect, lazy, Suspense } from 'react';
import { X, Copy, Download, FileCode, AlertCircle } from 'lucide-react';
import { Button } from '../atoms/Button';
import { Badge } from '../atoms/Badge';
import { codeIndexService, type FileDetails, type FileChunkDetails } from '../../services/codeIndexService';
import { ChunkList } from './ChunkList';

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

export const FileInspector: React.FC<FileInspectorProps> = ({ fileId, onClose }) => {
  const [fileDetails, setFileDetails] = useState<FileDetails | null>(null);
  const [chunks, setChunks] = useState<FileChunkDetails[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'chunks' | 'full'>('chunks');

  useEffect(() => {
    loadFileContent();
  }, [fileId]);

  const loadFileContent = async () => {
    setLoading(true);
    setError(null);

    try {
      // Fetch file details and chunks in parallel
      const [details, chunksData] = await Promise.all([
        codeIndexService.getFile(fileId),
        codeIndexService.getFileChunks(fileId),
      ]);

      setFileDetails(details);
      setChunks(chunksData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load file');
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = () => {
    if (chunks.length > 0) {
      const allContent = chunks.map(c => c.content).join('\n\n');
      navigator.clipboard.writeText(allContent);
    }
  };

  const handleDownload = () => {
    if (!fileDetails || chunks.length === 0) return;

    const allContent = chunks.map(c => c.content).join('\n\n');
    const blob = new Blob([allContent], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = fileDetails.filePath.split('/').pop() || 'file.txt';
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
                  {fileDetails?.filePath || 'Loading...'}
                </h2>
              </div>
              {fileDetails && (
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant="secondary" className="text-xs">
                    {fileDetails.language.toUpperCase()}
                  </Badge>
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    {formatFileSize(fileDetails.size)}
                  </span>
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    {fileDetails.lines.toLocaleString()} lines
                  </span>
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    {fileDetails.chunkCount} chunk{fileDetails.chunkCount !== 1 ? 's' : ''}
                  </span>
                  <span className="text-xs text-gray-500 dark:text-gray-400">
                    Indexed: {new Date(fileDetails.indexed).toLocaleDateString()}
                  </span>
                </div>
              )}
            </div>

            <div className="flex items-center gap-2 flex-shrink-0">
              {fileDetails && (
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
          ) : fileDetails ? (
            <div className="p-6">
              <ChunkList chunks={chunks} language={fileDetails.language} />
            </div>
          ) : null}
        </div>

        {/* Footer Info */}
        {fileDetails && (
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
