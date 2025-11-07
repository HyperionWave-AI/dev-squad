import { useState, useEffect } from 'react';
import { Library, Plus } from 'lucide-react';
import {
  CollectionBrowser,
  ArticleList,
  ArticleViewer,
  ArticleEditor,
  PageHeader,
} from '@/components/organisms';
import { Badge } from '@/components/atoms/Badge';
import { ResyncButton } from '@/components/ResyncButton';
import { ResyncProgressDialog } from '@/components/ResyncProgressDialog';
import { knowledgeApi } from '@/services/knowledgeApi';
import { knowledgeService } from '@/services/knowledgeService';
import type { KnowledgeCollection, KnowledgeEntry } from '@/types/knowledge';

export function KnowledgeBasePage() {
  const [collections, setCollections] = useState<KnowledgeCollection[]>([]);
  const [selectedCollection, setSelectedCollection] = useState<string | null>(null);
  const [entries, setEntries] = useState<KnowledgeEntry[]>([]);
  const [selectedEntry, setSelectedEntry] = useState<KnowledgeEntry | null>(null);
  const [isEditing, setIsEditing] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [loading, setLoading] = useState(false);
  const [collectionsLoading, setCollectionsLoading] = useState(true);
  const [entriesLoading, setEntriesLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Universal search state
  const [universalSearchQuery, setUniversalSearchQuery] = useState<string>('');
  const [isUniversalSearchMode, setIsUniversalSearchMode] = useState(false);
  const [searchCollections, setSearchCollections] = useState<string[]>([]);

  // Resync state
  const [showResyncProgress, setShowResyncProgress] = useState(false);

  // Load collections on mount
  useEffect(() => {
    loadCollections();
  }, []);

  // Load entries when collection is selected
  useEffect(() => {
    if (selectedCollection) {
      loadEntries(selectedCollection);
    } else {
      setEntries([]);
      setSelectedEntry(null);
    }
  }, [selectedCollection]);

  const loadCollections = async () => {
    setCollectionsLoading(true);
    setError(null);
    try {
      const response = await knowledgeApi.listCollections();
      setCollections(response.collections || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load collections');
      setCollections([]);
    } finally {
      setCollectionsLoading(false);
    }
  };

  const loadEntries = async (collection: string) => {
    setEntriesLoading(true);
    setError(null);
    try {
      const response = await knowledgeApi.getEntries(collection, 100);
      // Transform entries to match KnowledgeEntry type
      const transformedEntries: KnowledgeEntry[] = (response.entries || []).map((entry: any) => ({
        id: entry.id || entry._id || Math.random().toString(36).substr(2, 9),
        text: entry.text || entry.information || '',
        metadata: entry.metadata || {},
        createdAt: entry.createdAt || entry.created_at,
      }));
      setEntries(transformedEntries);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load entries');
      setEntries([]);
    } finally {
      setEntriesLoading(false);
    }
  };

  const handleCollectionSelect = (collectionName: string) => {
    setSelectedCollection(collectionName);
    setSelectedEntry(null);
    setIsEditing(false);
    // Clear universal search when switching to collection browse
    setIsUniversalSearchMode(false);
    setUniversalSearchQuery('');
  };

  const handleSelectEntry = (entry: KnowledgeEntry) => {
    setSelectedEntry(entry);
    setIsEditing(false);
  };

  const handleEdit = () => {
    setIsEditing(true);
  };

  const handleCancelEdit = () => {
    if (isCreating) {
      // If canceling a new entry creation, clear selection
      setSelectedEntry(null);
      setIsCreating(false);
    }
    setIsEditing(false);
  };

  const handleCreateNew = () => {
    // Create a temporary new entry
    const newEntry: KnowledgeEntry = {
      id: 'new-' + Date.now(),
      text: '',
      metadata: {},
      createdAt: new Date().toISOString()
    };
    setSelectedEntry(newEntry);
    setIsCreating(true);
    setIsEditing(true);
  };

  const handleSave = async (text: string, metadata: Record<string, any>, collectionName?: string) => {
    if (!selectedEntry) return;

    const isNewEntry = selectedEntry.id.startsWith('new-');

    try {
      if (isNewEntry) {
        // Creating a new entry
        const targetCollection = collectionName || selectedCollection;
        if (!targetCollection) {
          throw new Error('No collection selected');
        }

        const response = await knowledgeApi.createKnowledge({
          collection: targetCollection,
          text,
          metadata
        });

        // Create new entry from response
        const newEntry: KnowledgeEntry = {
          id: response.id || `${targetCollection}:${Date.now()}`,
          text: text,
          metadata: metadata,
          createdAt: response.createdAt || new Date().toISOString(),
          collection: response.collection || targetCollection,
        };

        // Add new entry to the list
        setEntries([newEntry, ...entries]);
        setSelectedEntry(newEntry);
        setIsCreating(false);
        setIsEditing(false);
        setSuccessMessage('Entry created successfully');
        setError(null);

        // Reload collections to update counts
        loadCollections();
      } else {
        // Updating existing entry
        const response = await knowledgeApi.updateEntry(selectedEntry.id, {
          text,
          metadata,
        });

        // Update the entry in the list
        const updatedEntry = {
          id: response.entry.id || selectedEntry.id,
          text: response.entry.text || text,
          metadata: response.entry.metadata || metadata,
          createdAt: response.entry.createdAt || selectedEntry.createdAt,
        };

        setEntries(entries.map(e =>
          e.id === selectedEntry.id ? updatedEntry : e
        ));

        // Update selected entry
        setSelectedEntry(updatedEntry);

        setIsEditing(false);
        setSuccessMessage('Entry updated successfully');
        setError(null);
      }
    } catch (err) {
      throw err; // Let the editor handle the error
    }
  };

  const handleDelete = async () => {
    if (!selectedEntry) return;

    try {
      await knowledgeApi.deleteEntry(selectedEntry.id);

      // Remove entry from list
      setEntries(entries.filter(e => e.id !== selectedEntry.id));

      // Clear selection
      setSelectedEntry(null);
      setIsEditing(false);

      setSuccessMessage('Entry deleted successfully');
      setError(null);

      // Reload collections to update counts
      loadCollections();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete entry');
    }
  };

  const handleCollectionCreated = () => {
    loadCollections();
  };

  // Universal search handler
  const handleUniversalSearch = async () => {
    if (!universalSearchQuery.trim()) {
      // Clear search mode
      setIsUniversalSearchMode(false);
      setEntries([]);
      setSearchCollections([]);
      setSelectedEntry(null);
      return;
    }

    setLoading(true);
    setIsUniversalSearchMode(true);
    setSelectedCollection(null); // Clear collection selection in universal search mode

    try {
      const { entries: searchResults, collectionsWithData } = await knowledgeService.universalSearch(
        universalSearchQuery,
        100
      );
      setEntries(searchResults);
      setSearchCollections(collectionsWithData);
      setSelectedEntry(null);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Universal search failed');
      setEntries([]);
      setSearchCollections([]);
    } finally {
      setLoading(false);
    }
  };

  // Handle Enter key in search box
  const handleSearchKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleUniversalSearch();
    }
  };

  // Clear universal search
  const handleClearUniversalSearch = () => {
    setUniversalSearchQuery('');
    setIsUniversalSearchMode(false);
    setEntries([]);
    setSearchCollections([]);
    setSelectedEntry(null);
  };

  // Handle resync started
  const handleResyncStarted = () => {
    setShowResyncProgress(true);
  };

  // Handle resync dialog close (optionally reload collections)
  const handleResyncClose = () => {
    setShowResyncProgress(false);
    // Reload collections to get updated counts
    loadCollections();
  };

  // Auto-hide success message after 3 seconds
  useEffect(() => {
    if (successMessage) {
      const timer = setTimeout(() => setSuccessMessage(null), 3000);
      return () => clearTimeout(timer);
    }
  }, [successMessage]);

  // Calculate total entries
  const totalEntries = collections.reduce((sum, c) => sum + c.count, 0);

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950 p-6">
      <div className="max-w-7xl mx-auto space-y-6">
      {/* Header with Stats */}
      <div className="space-y-4">
        <PageHeader
          title="Knowledge Base"
          description="Explore, search, and manage your knowledge collections"
          icon={<Library className="h-8 w-8" />}
          gradientFrom="#3b82f6"
          gradientTo="#8b5cf6"
        />
        <div className="flex gap-4 justify-between items-center">
          <ResyncButton onResyncStarted={handleResyncStarted} />
          <div className="flex gap-4">
            <Badge variant="default" className="px-4 py-2 bg-gradient-to-r from-blue-500 to-blue-600">
              {collections.length} Collections
            </Badge>
            <Badge variant="default" className="px-4 py-2 bg-gradient-to-r from-purple-500 to-purple-600">
              {totalEntries} Entries
            </Badge>
          </div>
        </div>
      </div>

      {/* Error Alert */}
      {error && (
        <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-start justify-between">
          <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
          <button
            onClick={() => setError(null)}
            className="text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-200"
          >
            ×
          </button>
        </div>
      )}

      {/* Success Alert */}
      {successMessage && (
        <div className="mb-6 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg flex items-start justify-between">
          <p className="text-sm text-green-800 dark:text-green-200">{successMessage}</p>
          <button
            onClick={() => setSuccessMessage(null)}
            className="text-green-600 dark:text-green-400 hover:text-green-800 dark:hover:text-green-200"
          >
            ×
          </button>
        </div>
      )}

      {/* Universal Search Bar - Full Width at Top */}
      <div className="mb-6">
        <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-lg p-5 transition-all duration-300 hover:shadow-xl">
          <div className="relative">
            <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
              <svg
                className="h-6 w-6 text-blue-500 dark:text-blue-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2.5}
                  d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
                />
              </svg>
            </div>
            <input
              type="text"
              className="block w-full pl-12 pr-14 py-4 text-base border-2 border-gray-300 dark:border-gray-600 rounded-xl bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 placeholder-gray-500 dark:placeholder-gray-400 focus:outline-none focus:ring-4 focus:ring-blue-500/20 focus:border-blue-500 focus:shadow-lg transition-all duration-200 font-medium"
              placeholder="Search across all collections... ✨ Press Enter"
              value={universalSearchQuery}
              onChange={(e) => setUniversalSearchQuery(e.target.value)}
              onKeyPress={handleSearchKeyPress}
            />
            {universalSearchQuery && (
              <button
                onClick={handleClearUniversalSearch}
                className="absolute inset-y-0 right-0 pr-4 flex items-center text-gray-400 hover:text-red-500 dark:hover:text-red-400 transition-all duration-200 hover:scale-110"
              >
                <svg
                  className="h-6 w-6"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2.5}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            )}
          </div>
          {isUniversalSearchMode && (
            <div className="mt-4 flex items-center gap-2 px-2">
              <div className="flex items-center gap-2 text-sm">
                <span className="text-gray-600 dark:text-gray-400">Found</span>
                <span className="px-3 py-1 bg-gradient-to-r from-blue-500 to-blue-600 text-white font-bold rounded-lg shadow-md">
                  {entries.length}
                </span>
                <span className="text-gray-600 dark:text-gray-400">results across</span>
                <span className="px-3 py-1 bg-gradient-to-r from-purple-500 to-purple-600 text-white font-bold rounded-lg shadow-md">
                  {searchCollections.length}
                </span>
                <span className="text-gray-600 dark:text-gray-400">collections</span>
                {entries.length >= 100 && (
                  <span className="ml-2 px-2 py-1 bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300 text-xs font-semibold rounded-md">
                    Top 100
                  </span>
                )}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Three-Column Layout: 25% - 25% - 50% */}
      <div className="grid grid-cols-12 gap-4">
        {/* Left Column: Collections Browser (25%) */}
        <div className="col-span-3">
          <CollectionBrowser
            collections={collections}
            selectedCollection={selectedCollection}
            onSelectCollection={handleCollectionSelect}
            onCollectionCreated={handleCollectionCreated}
            loading={collectionsLoading}
          />
        </div>

        {/* Middle Column: Article List (25%) */}
        <div className="col-span-3">
          <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm min-h-[calc(100vh-200px)] h-full transition-all duration-200">
            {/* Create New Article Button - Show when collection is selected */}
            {selectedCollection && !isUniversalSearchMode && (
              <div className="p-4 border-b border-gray-200 dark:border-gray-700">
                <button
                  onClick={handleCreateNew}
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-3 text-sm font-medium text-white bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 rounded-lg transition-all duration-200 shadow-md hover:shadow-lg transform hover:scale-[1.02]"
                >
                  <Plus className="h-5 w-5" />
                  New Article
                </button>
              </div>
            )}

            {!selectedCollection && !isUniversalSearchMode ? (
              <div className="flex flex-col items-center justify-center h-full text-center p-8">
                <div className="relative">
                  <div className="absolute inset-0 bg-blue-500/10 dark:bg-blue-400/10 rounded-full blur-2xl animate-pulse"></div>
                  <div className="relative rounded-full bg-gradient-to-br from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 p-10 mb-6 shadow-lg">
                    <svg
                      className="h-20 w-20 text-blue-500 dark:text-blue-400"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={1.5}
                        d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                      />
                    </svg>
                  </div>
                </div>
                <h3 className="text-xl font-bold text-gray-900 dark:text-gray-100 mb-3">
                  Select a Collection
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-2">
                  Choose a collection from the left
                </p>
                <p className="text-xs text-gray-500 dark:text-gray-500">
                  or use the universal search above ✨
                </p>
              </div>
            ) : entriesLoading || loading ? (
              <div className="flex flex-col items-center justify-center h-full gap-4">
                <div className="relative">
                  <div className="animate-spin rounded-full h-12 w-12 border-4 border-blue-200 dark:border-blue-900"></div>
                  <div className="animate-spin rounded-full h-12 w-12 border-4 border-blue-600 border-t-transparent absolute top-0 left-0"></div>
                </div>
                <p className="text-sm text-gray-600 dark:text-gray-400 font-medium">Loading entries...</p>
              </div>
            ) : (
              <ArticleList
                entries={entries}
                selectedEntryId={selectedEntry?.id || null}
                onSelectEntry={handleSelectEntry}
              />
            )}
          </div>
        </div>

        {/* Right Column: Article Viewer/Editor (50%) */}
        <div className="col-span-6">
          <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 min-h-[calc(100vh-200px)] h-full">
            {isEditing && selectedEntry ? (
              <ArticleEditor
                entry={selectedEntry}
                onSave={handleSave}
                onCancel={handleCancelEdit}
                collections={collections}
                selectedCollection={selectedCollection}
              />
            ) : (
              <ArticleViewer
                entry={selectedEntry}
                onEdit={handleEdit}
                onDelete={handleDelete}
              />
            )}
          </div>
        </div>
      </div>

      {/* Stats Footer */}
      {collections.length > 0 && (
        <div className="mt-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            {/* Total Collections Card */}
            <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-md p-5 hover:shadow-lg transition-all duration-200">
              <div className="flex items-center gap-3">
                <div className="p-3 bg-gradient-to-br from-blue-500 to-blue-600 rounded-lg shadow-sm">
                  <svg className="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                  </svg>
                </div>
                <div>
                  <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">Collections</p>
                  <p className="text-2xl font-bold text-blue-600 dark:text-blue-400">{collections.length}</p>
                </div>
              </div>
            </div>

            {/* Total Entries Card */}
            <div className="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-md p-5 hover:shadow-lg transition-all duration-200">
              <div className="flex items-center gap-3">
                <div className="p-3 bg-gradient-to-br from-green-500 to-green-600 rounded-lg shadow-sm">
                  <svg className="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                </div>
                <div>
                  <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">Total Entries</p>
                  <p className="text-2xl font-bold text-green-600 dark:text-green-400">{collections.reduce((sum, c) => sum + c.count, 0)}</p>
                </div>
              </div>
            </div>

            {/* Selected Collection Card */}
            {selectedCollection && (
              <div className="bg-white dark:bg-gray-800 rounded-xl border-2 border-purple-500 dark:border-purple-400 shadow-md p-5 hover:shadow-lg transition-all duration-200">
                <div className="flex items-center gap-3">
                  <div className="p-3 bg-gradient-to-br from-purple-500 to-purple-600 rounded-lg shadow-sm">
                    <svg className="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">Selected</p>
                    <p className="text-base font-bold text-purple-600 dark:text-purple-400 truncate">{selectedCollection}</p>
                    <p className="text-xs text-gray-500 dark:text-gray-400">{entries.length} entries</p>
                  </div>
                </div>
              </div>
            )}

            {/* Viewing Entry Card */}
            {selectedEntry && (
              <div className="bg-white dark:bg-gray-800 rounded-xl border-2 border-orange-500 dark:border-orange-400 shadow-md p-5 hover:shadow-lg transition-all duration-200">
                <div className="flex items-center gap-3">
                  <div className="p-3 bg-gradient-to-br from-orange-500 to-orange-600 rounded-lg shadow-sm">
                    <svg className="h-6 w-6 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                    </svg>
                  </div>
                  <div>
                    <p className="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide">Viewing</p>
                    <p className="text-2xl font-bold text-orange-600 dark:text-orange-400">
                      {entries.findIndex(e => e.id === selectedEntry.id) + 1} / {entries.length}
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Resync Progress Dialog */}
      <ResyncProgressDialog open={showResyncProgress} onClose={handleResyncClose} />
      </div>
    </div>
  );
}
