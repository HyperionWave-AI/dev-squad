import React, { useState, useEffect } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import * as Select from '@radix-ui/react-select';
import { X, Check, ChevronDown, Settings } from 'lucide-react';
import { Button } from '@atoms/Button';
import { Input } from '@atoms/Input';
import { Label } from '@atoms/Label';
import { Textarea } from '@atoms/Textarea';
import { Badge } from '@atoms/Badge';
import { cn } from '@/utils';
import { knowledgeService } from '@/services/knowledgeService';
import type { KnowledgeCollection } from '@/types/knowledge';

export interface CollectionSettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  collection: KnowledgeCollection | null;
  onSuccess: () => void;
}

const CATEGORIES = ['Tech', 'Task', 'UI', 'Ops', 'Other'];

export function CollectionSettingsDialog({
  open,
  onOpenChange,
  collection,
  onSuccess,
}: CollectionSettingsDialogProps) {
  const [formData, setFormData] = useState({
    name: '',
    category: 'Other',
    description: '',
    tags: [] as string[],
  });
  const [tagInput, setTagInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Initialize form data when collection changes
  useEffect(() => {
    if (collection) {
      setFormData({
        name: collection.name,
        category: collection.category || 'Other',
        description: collection.description || '',
        tags: collection.tags || [],
      });
    }
  }, [collection]);

  const handleClose = () => {
    if (!loading) {
      setTagInput('');
      setError(null);
      onOpenChange(false);
    }
  };

  const handleAddTag = () => {
    const tag = tagInput.trim();
    if (tag && !formData.tags.includes(tag)) {
      setFormData({ ...formData, tags: [...formData.tags, tag] });
      setTagInput('');
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setFormData({
      ...formData,
      tags: formData.tags.filter((tag) => tag !== tagToRemove),
    });
  };

  const handleTagKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAddTag();
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!collection) return;

    if (!formData.name.trim()) {
      setError('Collection name is required');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      // If name changed, rename the collection first
      if (formData.name !== collection.name) {
        await knowledgeService.renameCollection(collection.name, formData.name);
      }

      // Update metadata (description, tags, category)
      await knowledgeService.updateCollectionMetadata(
        formData.name, // Use new name if renamed
        {
          description: formData.description,
          tags: formData.tags,
          category: formData.category,
        }
      );

      onSuccess();
      handleClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update collection');
    } finally {
      setLoading(false);
    }
  };

  if (!collection) return null;

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 z-50" />
        <Dialog.Content
          className={cn(
            'fixed left-[50%] top-[50%] z-50 translate-x-[-50%] translate-y-[-50%]',
            'w-full max-w-lg bg-white dark:bg-gray-800 rounded-lg shadow-lg',
            'data-[state=open]:animate-in data-[state=closed]:animate-out',
            'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
            'data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95',
            'data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%]',
            'data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%]'
          )}
        >
          {/* Header */}
          <div className="flex items-center justify-between border-b border-gray-200 dark:border-gray-700 px-6 py-4">
            <div className="flex items-center gap-2">
              <Settings className="h-5 w-5 text-gray-500 dark:text-gray-400" />
              <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                Collection Settings
              </Dialog.Title>
            </div>
            <Dialog.Close asChild>
              <button
                className="rounded-sm opacity-70 ring-offset-white transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-gray-400 focus:ring-offset-2 disabled:pointer-events-none"
                disabled={loading}
              >
                <X className="h-4 w-4" />
                <span className="sr-only">Close</span>
              </button>
            </Dialog.Close>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="px-6 py-4 space-y-4">
            {/* Error Alert */}
            {error && (
              <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
                <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
              </div>
            )}

            {/* Collection Name */}
            <div className="space-y-2">
              <Label htmlFor="collection-name">
                Collection Name <span className="text-red-500">*</span>
              </Label>
              <Input
                id="collection-name"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                placeholder="e.g., Project Documentation"
                required
                disabled={loading}
              />
            </div>

            {/* Category */}
            <div className="space-y-2">
              <Label htmlFor="category">Category</Label>
              <Select.Root
                value={formData.category}
                onValueChange={(value) => setFormData({ ...formData, category: value })}
                disabled={loading}
              >
                <Select.Trigger
                  id="category"
                  className={cn(
                    'flex h-10 w-full items-center justify-between rounded-md border border-gray-300 dark:border-gray-600',
                    'bg-white dark:bg-gray-800 px-3 py-2 text-sm',
                    'hover:bg-gray-50 dark:hover:bg-gray-700',
                    'focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2',
                    'disabled:cursor-not-allowed disabled:opacity-50'
                  )}
                >
                  <Select.Value />
                  <Select.Icon>
                    <ChevronDown className="h-4 w-4 opacity-50" />
                  </Select.Icon>
                </Select.Trigger>

                <Select.Portal>
                  <Select.Content
                    className={cn(
                      'relative z-50 min-w-[8rem] overflow-hidden rounded-md border',
                      'bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700',
                      'shadow-md animate-in fade-in-80'
                    )}
                    position="popper"
                    sideOffset={4}
                  >
                    <Select.Viewport className="p-1">
                      {CATEGORIES.map((cat) => (
                        <Select.Item
                          key={cat}
                          value={cat}
                          className={cn(
                            'relative flex w-full cursor-pointer select-none items-center',
                            'rounded-sm py-2 pl-8 pr-2 text-sm outline-none',
                            'hover:bg-gray-100 dark:hover:bg-gray-700',
                            'focus:bg-gray-100 dark:focus:bg-gray-700'
                          )}
                        >
                          <span className="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
                            <Select.ItemIndicator>
                              <Check className="h-4 w-4" />
                            </Select.ItemIndicator>
                          </span>
                          <Select.ItemText>{cat}</Select.ItemText>
                        </Select.Item>
                      ))}
                    </Select.Viewport>
                  </Select.Content>
                </Select.Portal>
              </Select.Root>
            </div>

            {/* Description */}
            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                placeholder="Describe what this collection contains..."
                rows={3}
                disabled={loading}
              />
            </div>

            {/* Tags */}
            <div className="space-y-2">
              <Label htmlFor="tags">Tags</Label>
              <div className="flex gap-2">
                <Input
                  id="tags"
                  value={tagInput}
                  onChange={(e) => setTagInput(e.target.value)}
                  onKeyDown={handleTagKeyDown}
                  placeholder="Press Enter to add"
                  disabled={loading}
                  className="flex-1"
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleAddTag}
                  disabled={loading || !tagInput.trim()}
                >
                  Add
                </Button>
              </div>

              {/* Display Tags */}
              {formData.tags.length > 0 && (
                <div className="flex flex-wrap gap-2 mt-2">
                  {formData.tags.map((tag) => (
                    <Badge
                      key={tag}
                      variant="secondary"
                      className="flex items-center gap-1 px-2 py-1"
                    >
                      {tag}
                      <button
                        type="button"
                        onClick={() => handleRemoveTag(tag)}
                        disabled={loading}
                        className="ml-1 hover:text-gray-700 dark:hover:text-gray-200"
                      >
                        <X className="h-3 w-3" />
                      </button>
                    </Badge>
                  ))}
                </div>
              )}
            </div>
          </form>

          {/* Footer */}
          <div className="flex justify-end gap-3 border-t border-gray-200 dark:border-gray-700 px-6 py-4">
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={loading}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              onClick={handleSubmit}
              disabled={loading || !formData.name.trim()}
              className="flex items-center gap-2"
            >
              {loading ? (
                <>
                  <span className="animate-spin">⏳</span>
                  Saving...
                </>
              ) : (
                <>
                  <Check className="h-4 w-4" />
                  Save Changes
                </>
              )}
            </Button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
