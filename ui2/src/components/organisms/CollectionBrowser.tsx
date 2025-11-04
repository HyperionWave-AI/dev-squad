import React, { useState } from 'react';
import * as Accordion from '@radix-ui/react-accordion';
import * as Dialog from '@radix-ui/react-dialog';
import { Plus, ChevronDown, Folder, Search, X, Settings, Trash2, ClipboardCheck, Loader2 } from 'lucide-react';
import { Button } from '@atoms/Button';
import { Badge } from '@atoms/Badge';
import { cn } from '@/utils';
import type { KnowledgeCollection, CollectionReviewResult } from '@/types/knowledge';
import { knowledgeService } from '@/services/knowledgeService';
import { CreateCollectionModal } from './CreateCollectionModal';
import { CollectionReviewDialog } from './CollectionReviewDialog';

export interface CollectionBrowserProps {
  collections: KnowledgeCollection[];
  selectedCollection: string | null;
  onSelectCollection: (name: string) => void;
  onCollectionCreated: () => void;
  loading?: boolean;
}

const categoryIcons: Record<string, string> = {
  Tech: '🔧',
  Task: '📋',
  UI: '🎨',
  Ops: '⚙️',
  Other: '📚',
};

const categoryColors: Record<string, string> = {
  Task: 'bg-gradient-to-r from-blue-500 to-blue-600 text-white shadow-md',
  Tech: 'bg-gradient-to-r from-purple-500 to-purple-600 text-white shadow-md',
  UI: 'bg-gradient-to-r from-green-500 to-green-600 text-white shadow-md',
  Ops: 'bg-gradient-to-r from-orange-500 to-orange-600 text-white shadow-md',
  Other: 'bg-gradient-to-r from-gray-500 to-gray-600 text-white shadow-md',
};

export function CollectionBrowser({
  collections,
  selectedCollection,
  onSelectCollection,
  onCollectionCreated,
  loading = false,
}: CollectionBrowserProps) {
  const [modalOpen, setModalOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [collectionToDelete, setCollectionToDelete] = useState<KnowledgeCollection | null>(null);
  const [reviewDialogOpen, setReviewDialogOpen] = useState(false);
  const [reviewResult, setReviewResult] = useState<CollectionReviewResult | null>(null);
  const [reviewingCollection, setReviewingCollection] = useState<string | null>(null);

  // Filter collections by search query
  const filteredCollections = collections.filter((col) =>
    col.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Group collections by category
  const groupedCollections = filteredCollections.reduce((acc, collection) => {
    const category = collection.category || 'Other';
    if (!acc[category]) {
      acc[category] = [];
    }
    acc[category].push(collection);
    return acc;
  }, {} as Record<string, KnowledgeCollection[]>);

  // Category display order matching old UI
  const categoryOrder = ['Task', 'Tech', 'UI', 'Ops', 'Other'];
  const categories = categoryOrder.filter(cat => groupedCollections[cat]);

  const handleCreateSuccess = () => {
    setModalOpen(false);
    onCollectionCreated();
  };

  const handleDeleteClick = (collection: KnowledgeCollection, e: React.MouseEvent) => {
    e.stopPropagation();
    setCollectionToDelete(collection);
    setDeleteDialogOpen(true);
  };

  const handleSettingsClick = (collection: KnowledgeCollection, e: React.MouseEvent) => {
    e.stopPropagation();
    // TODO: Implement settings dialog
    console.log('Settings for:', collection.name);
  };

  const handleReviewCollection = async (collection: KnowledgeCollection, e: React.MouseEvent) => {
    e.stopPropagation();
    setReviewingCollection(collection.name);
    try {
      const result = await knowledgeService.reviewCollection(collection.name, 70, 100);
      setReviewResult(result);
      setReviewDialogOpen(true);
    } catch (err) {
      console.error('Failed to review collection:', err);
    } finally {
      setReviewingCollection(null);
    }
  };

  const truncateText = (text: string, maxLength: number) => {
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
  };

  if (loading) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
        <div className="animate-pulse space-y-4">
          <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-3/4"></div>
          <div className="space-y-3">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="h-16 bg-gray-200 dark:bg-gray-700 rounded"></div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
      {/* Header */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 flex items-center gap-2">
            <Folder className="h-5 w-5" />
            Collections
          </h3>
          <Button
            variant="primary"
            size="sm"
            onClick={() => setModalOpen(true)}
            className="flex items-center gap-1"
          >
            <Plus className="h-4 w-4" />
            Create
          </Button>
        </div>
        <p className="text-sm text-gray-600 dark:text-gray-400 mb-3">
          {collections.length} collections
        </p>

        {/* Search field */}
        <div className="relative">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <Search className="h-4 w-4 text-gray-400 dark:text-gray-500" />
          </div>
          <input
            type="text"
            className="block w-full pl-10 pr-10 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            placeholder="Search collections..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              className="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>

      {/* Collections Accordion */}
      <div className="overflow-y-auto max-h-[calc(100vh-350px)]">
        {filteredCollections.length === 0 ? (
          <div className="text-center py-8 text-gray-500 dark:text-gray-400 px-4">
            <p className="text-sm">
              {searchQuery ? 'No collections found matching your search.' : 'No collections found.'}
            </p>
            {!searchQuery && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setModalOpen(true)}
                className="mt-4"
              >
                Create your first collection
              </Button>
            )}
          </div>
        ) : (
          <Accordion.Root type="multiple" defaultValue={categories}>
            {categories.map((category) => (
              <Accordion.Item
                key={category}
                value={category}
                className="border-b border-gray-200 dark:border-gray-700 last:border-b-0"
              >
                <Accordion.Header>
                  <Accordion.Trigger
                    className={cn(
                      'flex w-full items-center justify-between px-4 py-4',
                      'bg-gray-50 dark:bg-gray-900/50 hover:bg-gray-100 dark:hover:bg-gray-800',
                      'transition-all duration-200 text-left font-semibold text-sm',
                      'group hover:shadow-sm'
                    )}
                  >
                    <span className="flex items-center gap-3">
                      <span
                        className={cn(
                          'inline-flex items-center px-4 py-1.5 rounded-lg text-sm font-bold tracking-wide',
                          categoryColors[category] || categoryColors.Other
                        )}
                      >
                        <span className="mr-2 text-base">{categoryIcons[category] || categoryIcons.Other}</span>
                        {category}
                      </span>
                      <Badge variant="secondary" className="text-xs font-bold px-2.5 py-1">
                        {groupedCollections[category].length}
                      </Badge>
                    </span>
                    <ChevronDown
                      className={cn(
                        'h-5 w-5 text-gray-500 transition-transform duration-300',
                        'group-data-[state=open]:rotate-180'
                      )}
                    />
                  </Accordion.Trigger>
                </Accordion.Header>

                <Accordion.Content
                  className={cn(
                    'overflow-hidden bg-white dark:bg-gray-800',
                    'data-[state=open]:animate-accordion-down',
                    'data-[state=closed]:animate-accordion-up'
                  )}
                >
                  <div className="py-2 px-2">
                    {groupedCollections[category].map((collection) => (
                      <div
                        key={collection.name}
                        className={cn(
                          'group relative mb-2 rounded-xl border-2 transition-all duration-200',
                          'hover:shadow-lg hover:-translate-y-0.5',
                          selectedCollection === collection.name
                            ? 'bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 border-blue-500 shadow-md'
                            : 'bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 hover:border-blue-300 dark:hover:border-blue-600'
                        )}
                      >
                        <div className="flex items-start justify-between gap-2 p-4">
                          <button
                            onClick={() => onSelectCollection(collection.name)}
                            className="flex-1 min-w-0 text-left"
                          >
                            <div className="flex items-center gap-2 mb-2">
                              <p className="text-sm font-bold text-gray-900 dark:text-gray-100 truncate">
                                {collection.name}
                              </p>
                              <span className="shrink-0 px-2.5 py-1 bg-gradient-to-r from-blue-500 to-blue-600 text-white text-xs font-bold rounded-lg shadow-sm">
                                {collection.count}
                              </span>
                            </div>
                            {collection.description && (
                              <p className="text-xs text-gray-600 dark:text-gray-400 mb-2 leading-relaxed">
                                {truncateText(collection.description, 50)}
                              </p>
                            )}
                            {collection.tags && collection.tags.length > 0 && (
                              <div className="flex gap-1.5 flex-wrap">
                                {collection.tags.slice(0, 2).map((tag) => (
                                  <span
                                    key={tag}
                                    className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-600"
                                  >
                                    {tag}
                                  </span>
                                ))}
                                {collection.tags.length > 2 && (
                                  <span className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300">
                                    +{collection.tags.length - 2}
                                  </span>
                                )}
                              </div>
                            )}
                          </button>

                          {/* Action buttons - shown on hover */}
                          <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-all duration-200 shrink-0">
                            <button
                              onClick={(e) => handleReviewCollection(collection, e)}
                              disabled={reviewingCollection === collection.name}
                              className="p-2 rounded-lg hover:bg-cyan-100 dark:hover:bg-cyan-900/30 text-gray-400 hover:text-cyan-600 dark:hover:text-cyan-400 transition-all duration-200 hover:scale-110 disabled:opacity-50 disabled:cursor-not-allowed"
                              title="Review collection"
                            >
                              {reviewingCollection === collection.name ? (
                                <Loader2 className="h-4 w-4 animate-spin" />
                              ) : (
                                <ClipboardCheck className="h-4 w-4" />
                              )}
                            </button>
                            <button
                              onClick={(e) => handleDeleteClick(collection, e)}
                              className="p-2 rounded-lg hover:bg-red-100 dark:hover:bg-red-900/30 text-gray-400 hover:text-red-600 dark:hover:text-red-400 transition-all duration-200 hover:scale-110"
                              title="Delete collection"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                            <button
                              onClick={(e) => handleSettingsClick(collection, e)}
                              className="p-2 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-all duration-200 hover:scale-110"
                              title="Collection settings"
                            >
                              <Settings className="h-4 w-4" />
                            </button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </Accordion.Content>
              </Accordion.Item>
            ))}
          </Accordion.Root>
        )}
      </div>

      {/* Summary */}
      {collections.length > 0 && (
        <div className="p-3 bg-gray-50 dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700">
          <p className="text-xs text-gray-600 dark:text-gray-400">
            <strong>{collections.length}</strong> collection{collections.length !== 1 ? 's' : ''}{' '}
            · <strong>{collections.reduce((sum, c) => sum + c.count, 0)}</strong> total entries
          </p>
        </div>
      )}

      {/* Create Collection Modal */}
      <CreateCollectionModal
        open={modalOpen}
        onOpenChange={setModalOpen}
        onSuccess={handleCreateSuccess}
      />

      {/* Delete Confirmation Dialog */}
      <Dialog.Root open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 data-[state=open]:animate-fade-in z-50" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6 w-full max-w-md data-[state=open]:animate-scale-in z-50">
            <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
              Confirm Delete
            </Dialog.Title>
            <Dialog.Description className="text-sm text-gray-600 dark:text-gray-400 mb-6">
              Are you sure you want to delete the collection{' '}
              <strong className="text-gray-900 dark:text-gray-100">{collectionToDelete?.name}</strong>?
              {collectionToDelete && collectionToDelete.count > 0 && (
                <span className="block mt-2 text-red-600 dark:text-red-400 font-medium">
                  This will delete {collectionToDelete.count} {collectionToDelete.count === 1 ? 'entry' : 'entries'}. This action cannot be undone.
                </span>
              )}
            </Dialog.Description>
            <div className="flex justify-end gap-3">
              <Dialog.Close asChild>
                <button
                  className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
                >
                  Cancel
                </button>
              </Dialog.Close>
              <button
                onClick={() => {
                  // TODO: Implement actual delete
                  console.log('Delete collection:', collectionToDelete?.name);
                  setDeleteDialogOpen(false);
                  setCollectionToDelete(null);
                }}
                className="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-700 rounded-md transition-colors"
              >
                Delete
              </button>
            </div>
            <Dialog.Close asChild>
              <button
                className="absolute top-4 right-4 p-1 rounded-md text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
                aria-label="Close"
              >
                <X className="h-4 w-4" />
              </button>
            </Dialog.Close>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>

      {/* Collection Review Dialog */}
      <CollectionReviewDialog
        open={reviewDialogOpen}
        onClose={() => setReviewDialogOpen(false)}
        result={reviewResult}
      />
    </div>
  );
}
