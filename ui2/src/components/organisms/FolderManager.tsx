import React, { useState } from 'react';
import { Folder, Plus, Trash2, Power, PowerOff, RefreshCw, X, AlertTriangle } from 'lucide-react';
import { Button } from '../atoms/Button';
import { Input } from '../atoms/Input';
import { Badge } from '../atoms/Badge';
import type { IndexedFolder, IndexStatus, FolderConfig } from '../../types/codeSearch';
import { codeIndexService } from '../../services/codeIndexService';

interface FolderManagerProps {
  folders: IndexedFolder[];
  status: IndexStatus;
  onAdd: (config: FolderConfig) => void;
  onRemove: (folderId: string) => void;
  onWatcherToggle: (folderId: string, enabled: boolean) => void;
  onReindex: () => void;
  onRefreshStatus: () => void;
}

const DEFAULT_FILE_PATTERNS = ['.ts', '.tsx', '.js', '.jsx', '.go', '.py'];
const DEFAULT_EXCLUDE_PATTERNS = ['node_modules', 'dist', 'build', '.git'];

const CHUNK_SIZES = [
  { value: 's', label: 'S', description: '50 lines' },
  { value: 'm', label: 'M', description: '100 lines' },
  { value: 'l', label: 'L', description: '200 lines' },
  { value: 'xl', label: 'XL', description: '400 lines' },
];

export const FolderManager: React.FC<FolderManagerProps> = ({
  folders,
  status,
  onAdd,
  onRemove,
  onWatcherToggle,
  onReindex,
  onRefreshStatus,
}) => {
  const [showAddModal, setShowAddModal] = useState(false);
  const [newFolderPath, setNewFolderPath] = useState('');
  const [selectedPatterns, setSelectedPatterns] = useState<string[]>(DEFAULT_FILE_PATTERNS);
  const [chunkSize, setChunkSize] = useState('m');
  const [excludePatterns, setExcludePatterns] = useState<string[]>(DEFAULT_EXCLUDE_PATTERNS);
  const [customPattern, setCustomPattern] = useState('');
  const [customExclude, setCustomExclude] = useState('');
  const [showClearAllModal, setShowClearAllModal] = useState(false);
  const [confirmText, setConfirmText] = useState('');
  const [isClearing, setIsClearing] = useState(false);
  const [clearErrors, setClearErrors] = useState<string[]>([]);

  const handleAddFolder = () => {
    if (!newFolderPath.trim()) return;

    const config: FolderConfig = {
      path: newFolderPath.trim(),
      filePatterns: selectedPatterns,
      chunkSize,
      excludePatterns,
    };

    onAdd(config);
    setShowAddModal(false);
    setNewFolderPath('');
    setSelectedPatterns(DEFAULT_FILE_PATTERNS);
    setChunkSize('m');
    setExcludePatterns(DEFAULT_EXCLUDE_PATTERNS);
  };

  const addCustomPattern = () => {
    if (customPattern.trim() && !selectedPatterns.includes(customPattern.trim())) {
      setSelectedPatterns(prev => [...prev, customPattern.trim()]);
      setCustomPattern('');
    }
  };

  const removePattern = (pattern: string) => {
    setSelectedPatterns(prev => prev.filter(p => p !== pattern));
  };

  const addCustomExclude = () => {
    if (customExclude.trim() && !excludePatterns.includes(customExclude.trim())) {
      setExcludePatterns(prev => [...prev, customExclude.trim()]);
      setCustomExclude('');
    }
  };

  const removeExclude = (pattern: string) => {
    setExcludePatterns(prev => prev.filter(p => p !== pattern));
  };

  const handleClearAll = async () => {
    if (confirmText !== 'CLEAR ALL') return;

    setIsClearing(true);
    setClearErrors([]);

    try {
      const result = await codeIndexService.clearAllIndexData();

      if (result.success) {
        setShowClearAllModal(false);
        setConfirmText('');
        onRefreshStatus();
      } else if (result.errors && result.errors.length > 0) {
        setClearErrors(result.errors);
      }
    } catch (error) {
      setClearErrors([error instanceof Error ? error.message : 'Failed to clear index data']);
    } finally {
      setIsClearing(false);
    }
  };

  return (
    <>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md border border-gray-200 dark:border-gray-700">
        {/* Header */}
        <div className="p-4 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              Indexed Folders
            </h2>
            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowAddModal(true)}
            >
              <Plus className="h-4 w-4 mr-1" />
              Add
            </Button>
          </div>
        </div>

        {/* Folder List */}
        <div className="p-4 space-y-3">
          {folders.length === 0 ? (
            <div className="text-center py-8 text-gray-500 dark:text-gray-400">
              <Folder className="h-12 w-12 mx-auto mb-2 opacity-50" />
              <p className="text-sm">No folders indexed yet</p>
            </div>
          ) : (
            folders.map((folder) => (
              <div
                key={folder.id}
                className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-700"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <Folder className="h-4 w-4 text-gray-500 flex-shrink-0" />
                    <p className="text-sm font-mono text-gray-900 dark:text-gray-100 truncate">
                      {folder.path}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant={folder.enabled ? 'success' : 'secondary'} className="text-xs">
                      {folder.enabled ? 'Active' : 'Inactive'}
                    </Badge>
                    <span className="text-xs text-gray-500 dark:text-gray-400">
                      {folder.fileCount} files
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-1 ml-2">
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => onWatcherToggle(folder.id, !folder.enabled)}
                    title={folder.enabled ? 'Disable watcher' : 'Enable watcher'}
                  >
                    {folder.enabled ? (
                      <Power className="h-4 w-4 text-green-600" />
                    ) : (
                      <PowerOff className="h-4 w-4 text-gray-400" />
                    )}
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => onRemove(folder.id)}
                    title="Remove folder"
                  >
                    <Trash2 className="h-4 w-4 text-red-600" />
                  </Button>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Reindex All and Clear All Buttons */}
        {folders.length > 0 && (
          <div className="p-4 border-t border-gray-200 dark:border-gray-700 space-y-2">
            <Button
              variant="outline"
              className="w-full"
              onClick={onReindex}
              disabled={status.isRunning}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${status.isRunning ? 'animate-spin' : ''}`} />
              {status.isRunning ? 'Reindexing...' : 'Reindex All'}
            </Button>
            <Button
              variant="outline"
              className="w-full border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
              onClick={() => setShowClearAllModal(true)}
              disabled={status.isRunning}
            >
              <Trash2 className="h-4 w-4 mr-2" />
              Clear All Index Data
            </Button>
          </div>
        )}
      </div>

      {/* Add Folder Modal */}
      {showAddModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            {/* Modal Header */}
            <div className="sticky top-0 bg-white dark:bg-gray-800 p-6 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
              <h3 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                Add Folder to Index
              </h3>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setShowAddModal(false)}
              >
                <X className="h-5 w-5" />
              </Button>
            </div>

            {/* Modal Body */}
            <div className="p-6 space-y-6">
              {/* Folder Path */}
              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Folder Path *
                </label>
                <Input
                  type="text"
                  value={newFolderPath}
                  onChange={(e) => setNewFolderPath(e.target.value)}
                  placeholder="/path/to/your/code"
                />
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Absolute path to the folder you want to index
                </p>
              </div>

              {/* File Patterns */}
              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  File Patterns
                </label>
                <div className="flex flex-wrap gap-2 mb-2">
                  {selectedPatterns.map((pattern) => (
                    <Badge
                      key={pattern}
                      variant="primary"
                      className="cursor-pointer"
                      onClick={() => removePattern(pattern)}
                    >
                      {pattern}
                      <X className="h-3 w-3 ml-1" />
                    </Badge>
                  ))}
                </div>
                <div className="flex gap-2">
                  <Input
                    type="text"
                    value={customPattern}
                    onChange={(e) => setCustomPattern(e.target.value)}
                    placeholder="Add custom pattern (e.g., .rs)"
                    onKeyPress={(e) => e.key === 'Enter' && addCustomPattern()}
                  />
                  <Button variant="outline" onClick={addCustomPattern}>
                    Add
                  </Button>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Click a pattern to remove it
                </p>
              </div>

              {/* Chunk Size */}
              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Chunk Size
                </label>
                <div className="flex flex-wrap gap-2">
                  {CHUNK_SIZES.map((size) => (
                    <Badge
                      key={size.value}
                      variant={chunkSize === size.value ? 'primary' : 'secondary'}
                      className="cursor-pointer px-4 py-2"
                      onClick={() => setChunkSize(size.value)}
                    >
                      {size.label} - {size.description}
                    </Badge>
                  ))}
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Select the chunk size for code indexing
                </p>
              </div>

              {/* Exclude Patterns */}
              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Exclude Patterns
                </label>
                <div className="flex flex-wrap gap-2 mb-2">
                  {excludePatterns.map((pattern) => (
                    <Badge
                      key={pattern}
                      variant="secondary"
                      className="cursor-pointer"
                      onClick={() => removeExclude(pattern)}
                    >
                      {pattern}
                      <X className="h-3 w-3 ml-1" />
                    </Badge>
                  ))}
                </div>
                <div className="flex gap-2">
                  <Input
                    type="text"
                    value={customExclude}
                    onChange={(e) => setCustomExclude(e.target.value)}
                    placeholder="Add exclude pattern (e.g., __pycache__)"
                    onKeyPress={(e) => e.key === 'Enter' && addCustomExclude()}
                  />
                  <Button variant="outline" onClick={addCustomExclude}>
                    Add
                  </Button>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  Click a pattern to remove it
                </p>
              </div>
            </div>

            {/* Modal Footer */}
            <div className="sticky bottom-0 bg-white dark:bg-gray-800 p-6 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
              <Button
                variant="outline"
                onClick={() => setShowAddModal(false)}
              >
                Cancel
              </Button>
              <Button
                variant="primary"
                onClick={handleAddFolder}
                disabled={!newFolderPath.trim()}
              >
                Add Folder
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Clear All Confirmation Modal */}
      {showClearAllModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-lg w-full">
            {/* Modal Header */}
            <div className="p-6 border-b border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-red-100 dark:bg-red-900/30 rounded-full">
                  <AlertTriangle className="h-6 w-6 text-red-600 dark:text-red-400" />
                </div>
                <h3 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                  Clear All Index Data
                </h3>
              </div>
            </div>

            {/* Modal Body */}
            <div className="p-6 space-y-4">
              <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                <p className="text-sm text-red-800 dark:text-red-200 font-medium mb-2">
                  ⚠️ Warning: This is a destructive operation!
                </p>
                <p className="text-sm text-red-700 dark:text-red-300">
                  This will permanently delete ALL indexed data including:
                </p>
                <ul className="list-disc list-inside text-sm text-red-700 dark:text-red-300 mt-2 space-y-1">
                  <li>{folders.length} folder configuration{folders.length !== 1 ? 's' : ''}</li>
                  <li>{status.totalFiles || 0} indexed file{status.totalFiles !== 1 ? 's' : ''}</li>
                  <li>All file chunks and embeddings</li>
                  <li>All Qdrant vector collections</li>
                </ul>
              </div>

              {clearErrors.length > 0 && (
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                  <p className="text-sm font-medium text-red-800 dark:text-red-200 mb-2">
                    Errors occurred:
                  </p>
                  <ul className="list-disc list-inside text-sm text-red-700 dark:text-red-300 space-y-1">
                    {clearErrors.map((error, index) => (
                      <li key={index}>{error}</li>
                    ))}
                  </ul>
                </div>
              )}

              <div className="space-y-2">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Type <span className="font-mono font-bold">CLEAR ALL</span> to confirm:
                </label>
                <Input
                  type="text"
                  value={confirmText}
                  onChange={(e) => setConfirmText(e.target.value)}
                  placeholder="CLEAR ALL"
                  className="font-mono"
                  disabled={isClearing}
                  autoFocus
                />
              </div>
            </div>

            {/* Modal Footer */}
            <div className="p-6 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
              <Button
                variant="outline"
                onClick={() => {
                  setShowClearAllModal(false);
                  setConfirmText('');
                  setClearErrors([]);
                }}
                disabled={isClearing}
              >
                Cancel
              </Button>
              <Button
                variant="outline"
                className="border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
                onClick={handleClearAll}
                disabled={confirmText !== 'CLEAR ALL' || isClearing}
              >
                {isClearing ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-red-600 mr-2" />
                    Clearing...
                  </>
                ) : (
                  <>
                    <Trash2 className="h-4 w-4 mr-2" />
                    Clear All Data
                  </>
                )}
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
};
