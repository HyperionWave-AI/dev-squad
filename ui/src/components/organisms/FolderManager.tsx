import React, { useState } from 'react';
import { Folder, Plus, Trash2, Power, PowerOff, RefreshCw, X } from 'lucide-react';
import { Button } from '../atoms/Button';
import { Input } from '../atoms/Input';
import { Badge } from '../atoms/Badge';
import type { IndexedFolder, IndexStatus, FolderConfig } from '../../types/codeSearch';

interface FolderManagerProps {
  folders: IndexedFolder[];
  status: IndexStatus;
  onAdd: (config: FolderConfig) => void;
  onRemove: (folderId: string) => void;
  onWatcherToggle: (folderId: string, enabled: boolean) => void;
  onReindex: () => void;
}

const DEFAULT_FILE_PATTERNS = ['.ts', '.tsx', '.js', '.jsx', '.go', '.py'];
const DEFAULT_EXCLUDE_PATTERNS = ['node_modules', 'dist', 'build', '.git'];

export const FolderManager: React.FC<FolderManagerProps> = ({
  folders,
  status,
  onAdd,
  onRemove,
  onWatcherToggle,
  onReindex,
}) => {
  const [showAddModal, setShowAddModal] = useState(false);
  const [newFolderPath, setNewFolderPath] = useState('');
  const [selectedPatterns, setSelectedPatterns] = useState<string[]>(DEFAULT_FILE_PATTERNS);
  const [chunkSize, setChunkSize] = useState(200);
  const [excludePatterns, setExcludePatterns] = useState<string[]>(DEFAULT_EXCLUDE_PATTERNS);
  const [customPattern, setCustomPattern] = useState('');
  const [customExclude, setCustomExclude] = useState('');

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
    setChunkSize(200);
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

        {/* Reindex All Button */}
        {folders.length > 0 && (
          <div className="p-4 border-t border-gray-200 dark:border-gray-700">
            <Button
              variant="outline"
              className="w-full"
              onClick={onReindex}
              disabled={status.isRunning}
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${status.isRunning ? 'animate-spin' : ''}`} />
              {status.isRunning ? 'Reindexing...' : 'Reindex All'}
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
                  Chunk Size: {chunkSize} lines
                </label>
                <input
                  type="range"
                  min="50"
                  max="500"
                  step="50"
                  value={chunkSize}
                  onChange={(e) => setChunkSize(parseInt(e.target.value))}
                  className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700 accent-primary-500"
                />
                <div className="flex justify-between text-xs text-gray-500 dark:text-gray-400">
                  <span>50</span>
                  <span>500</span>
                </div>
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
    </>
  );
};
