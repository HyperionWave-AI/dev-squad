import React, { useState, useEffect } from 'react';
import { CodeSearchForm } from '../components/organisms/CodeSearchForm';
import { CodeResultsList } from '../components/organisms/CodeResultsList';
import { FolderManager } from '../components/organisms/FolderManager';
import { IndexStatusDisplay } from '../components/organisms/IndexStatusDisplay';
import { FileInspector } from '../components/organisms/FileInspector';
import { PageHeader } from '../components/organisms/PageHeader';
import { ErrorBoundary } from '../components/organisms/ErrorBoundary';
import { codeIndexService } from '../services/codeIndexService';
import type { CodeResult, IndexedFolder, IndexStatus, FolderConfig, SearchOptions } from '../types/codeSearch';
import { Code2 } from 'lucide-react';

export const CodeSearchPage: React.FC = () => {
  // Search state
  const [results, setResults] = useState<CodeResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedFileId, setSelectedFileId] = useState<string | null>(null);

  // Index management state
  const [folders, setFolders] = useState<IndexedFolder[]>([]);
  const [status, setStatus] = useState<IndexStatus>({
    totalFiles: 0,
    totalFolders: 0,
    isRunning: false,
  });

  // Load initial status on mount
  useEffect(() => {
    loadStatus();
  }, []);

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd+K or Ctrl+K to focus search
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        const searchInput = document.querySelector('input[type="text"]') as HTMLInputElement;
        searchInput?.focus();
      }

      // Esc to close file inspector
      if (e.key === 'Escape' && selectedFileId) {
        setSelectedFileId(null);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [selectedFileId]);

  const loadStatus = async () => {
    try {
      const statusData = await codeIndexService.getStatus();
      setStatus({
        totalFiles: statusData.totalFiles || 0,
        totalFolders: statusData.folders?.length || 0,
        lastScanTime: statusData.lastScanTime,
        isRunning: statusData.watcherStatus === 'running',
      });

      // Convert folders array to IndexedFolder format
      if (statusData.folders) {
        const foldersData: IndexedFolder[] = statusData.folders.map((folder, index) => ({
          id: folder.configId || `folder-${index}`,
          path: folder.folderPath,
          fileCount: folder.fileCount || 0,
          enabled: folder.enabled ?? true,
        }));
        setFolders(foldersData);
      }
    } catch (error) {
      console.error('Failed to load status:', error);
    }
  };

  const handleSearch = async (query: string, options: SearchOptions) => {
    setLoading(true);
    try {
      const response = await codeIndexService.search({
        query,
        limit: options.maxResults,
        minScore: options.minRelevanceScore,
        folderPath: options.folderPath,
        retrieve: options.retrieve,
        fileTypes: options.fileTypes,
      });

      // Transform results to match our interface
      const transformedResults: CodeResult[] = response.results.map((result) => {
        const fileExtension = result.filePath.split('.').pop()?.toLowerCase() || '';
        const languageMap: Record<string, string> = {
          'ts': 'typescript',
          'tsx': 'tsx',
          'js': 'javascript',
          'jsx': 'jsx',
          'go': 'go',
          'py': 'python',
        };

        return {
          id: result.fileId,
          filePath: result.filePath,
          fileName: result.filePath.split('/').pop() || '',
          language: languageMap[fileExtension] || 'text',
          content: result.content,
          lineStart: result.lineStart,
          lineEnd: result.lineEnd,
          relevanceScore: result.score,
        };
      });

      setResults(transformedResults);
    } catch (error) {
      console.error('Search failed:', error);
      alert('Search failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleInspect = (fileId: string) => {
    setSelectedFileId(fileId);
  };

  const handleAddFolder = async (config: FolderConfig) => {
    try {
      // Add folder with full configuration
      const result = await codeIndexService.addFolder({
        folderPath: config.path,
        includePatterns: config.filePatterns,
        excludePatterns: config.excludePatterns,
        chunkSize: config.chunkSize,
      });

      if (result.success) {
        // Trigger initial scan for the new folder
        await codeIndexService.triggerScan(config.path);
        await loadStatus();
        alert('Folder added successfully!');
      }
    } catch (error) {
      console.error('Failed to add folder:', error);
      alert(`Failed to add folder: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleRemoveFolder = async (folderId: string) => {
    if (!confirm('Are you sure you want to remove this folder?')) return;

    try {
      const result = await codeIndexService.removeFolder(folderId);

      if (result.success) {
        await loadStatus();
        alert('Folder removed successfully!');
      }
    } catch (error) {
      console.error('Failed to remove folder:', error);
      alert(`Failed to remove folder: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleWatcherToggle = async (folderId: string, enabled: boolean) => {
    try {
      const result = await codeIndexService.toggleFolderWatcher(folderId, enabled);

      if (result.success) {
        await loadStatus();
      } else {
        alert(result.message || 'Failed to toggle watcher');
      }
    } catch (error) {
      console.error('Failed to toggle watcher:', error);
      alert(`Failed to toggle watcher: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  };

  const handleReindex = async () => {
    if (!confirm('This will reindex all folders. Continue?')) return;

    setStatus(prev => ({ ...prev, isRunning: true }));

    try {
      const result = await codeIndexService.reindexAll();

      if (result.success) {
        alert(`Reindexed ${result.foldersReindexed} folders (${result.totalFilesIndexed} files)`);
        await loadStatus();
      }
    } catch (error) {
      console.error('Failed to reindex:', error);
      alert(`Failed to reindex: ${error instanceof Error ? error.message : 'Unknown error'}`);
    } finally {
      setStatus(prev => ({ ...prev, isRunning: false }));
    }
  };

  const handleRefreshStatus = () => {
    loadStatus();
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950 p-6">
      <div className="max-w-7xl mx-auto space-y-6">
        {/* Page Header */}
        <PageHeader
          title="Code Search"
          description="Semantic search across your indexed codebase"
          icon={<Code2 className="h-8 w-8" />}
          gradientFrom="#3b82f6"
          gradientTo="#1d4ed8"
        />

        {/* Main Grid Layout */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Left Column: Search & Results (2/3) */}
          <div className="lg:col-span-2 space-y-6">
            <CodeSearchForm
              onSearch={handleSearch}
              loading={loading}
            />

            <CodeResultsList
              results={results}
              loading={loading}
              onInspect={handleInspect}
            />
          </div>

          {/* Right Column: Config & Status (1/3) */}
          <div className="space-y-6">
            <FolderManager
              folders={folders}
              status={status}
              onAdd={handleAddFolder}
              onRemove={handleRemoveFolder}
              onWatcherToggle={handleWatcherToggle}
              onReindex={handleReindex}
              onRefreshStatus={handleRefreshStatus}
            />

            <IndexStatusDisplay
              status={status}
              onRefresh={handleRefreshStatus}
            />
          </div>
        </div>

        {/* File Inspector Drawer */}
        {selectedFileId && (
          <FileInspector
            fileId={selectedFileId}
            onClose={() => setSelectedFileId(null)}
          />
        )}

        {/* Keyboard Shortcuts Info */}
        <div className="mt-8 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
          <p className="text-sm text-blue-800 dark:text-blue-200">
            <strong>Keyboard Shortcuts:</strong> Cmd+K to focus search • Esc to close inspector
          </p>
        </div>
      </div>
    </div>
  );
};

// Wrap with ErrorBoundary for production error handling
// Keep named export for testing purposes
export default () => (
  <ErrorBoundary>
    <CodeSearchPage />
  </ErrorBoundary>
);
