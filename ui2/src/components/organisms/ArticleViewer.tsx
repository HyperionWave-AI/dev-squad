import React, { useState } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import * as Accordion from '@radix-ui/react-accordion';
import { Edit, Trash2, X, ChevronDown, ClipboardCheck, Archive, Loader2 } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { KnowledgeEntry, ReviewResult, CompactionResult } from '@/types/knowledge';
import { knowledgeService } from '@/services/knowledgeService';
import { ReviewResultDialog } from './ReviewResultDialog';
import { CompactionDialog } from './CompactionDialog';

interface ArticleViewerProps {
  entry: KnowledgeEntry | null;
  onEdit: () => void;
  onDelete: () => void;
}

export const ArticleViewer: React.FC<ArticleViewerProps> = ({
  entry,
  onEdit,
  onDelete,
}) => {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [reviewing, setReviewing] = useState(false);
  const [reviewResult, setReviewResult] = useState<ReviewResult | null>(null);
  const [reviewDialogOpen, setReviewDialogOpen] = useState(false);
  const [compacting, setCompacting] = useState(false);
  const [compactionResult, setCompactionResult] = useState<CompactionResult | null>(null);
  const [compactionDialogOpen, setCompactionDialogOpen] = useState(false);

  // Mock data test handlers (DEV ONLY - for debugging)
  const handleReviewMock = () => {
    console.log('🧪 [MOCK REVIEW] Testing dialog with mock data');
    const mockResult: ReviewResult = {
      success: true,
      entryId: entry?.id || 'mock-id',
      scores: {
        alignment: 85,
        freshness: 72,
        verbosity: 90,
        uniqueness: 88,
        health: 84
      },
      verification: {
        totalReferences: 5,
        validReferences: 4,
        brokenReferences: [
          { type: 'file', value: '/path/to/missing.ts', error: 'File not found' }
        ]
      },
      actions: [
        { type: 'update', description: 'Update outdated dependency reference', applied: false },
        { type: 'cleanup', description: 'Remove redundant section', applied: false }
      ]
    };
    setReviewResult(mockResult);
    setReviewDialogOpen(true);
    console.log('✅ [MOCK REVIEW] Dialog should now be open with mock data');
  };

  const handleCompactMock = () => {
    console.log('🧪 [MOCK COMPACT] Testing dialog with mock data');
    const mockResult: CompactionResult = {
      success: true,
      original: {
        wordCount: 800,
        text: 'This is a very long original text with lots of details and explanations that could be condensed...'
      },
      compacted: {
        wordCount: 400,
        text: 'This is a concise version of the text with key points preserved...'
      },
      compressionRatio: 0.5,
      preserved: {
        filePaths: 12,
        functionNames: 8
      }
    };
    setCompactionResult(mockResult);
    setCompactionDialogOpen(true);
    console.log('✅ [MOCK COMPACT] Dialog should now be open with mock data');
  };

  if (!entry) {
    return (
      <div className="flex items-center justify-center h-full p-8">
        <p className="text-gray-500 dark:text-gray-400">
          Select an entry to view
        </p>
      </div>
    );
  }

  const handleDeleteClick = () => {
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = () => {
    setDeleteDialogOpen(false);
    onDelete();
  };

  const handleDeleteCancel = () => {
    setDeleteDialogOpen(false);
  };

  const handleReview = async () => {
    if (!entry) return;

    console.log('🔍 [REVIEW] Starting review for entry:', entry.id);
    setReviewing(true);
    try {
      console.log('🔍 [REVIEW] Calling API: POST /api/v1/knowledge/entries/{id}/review');
      const result = await knowledgeService.reviewEntry(entry.id, 'full', false);
      console.log('✅ [REVIEW] Review result:', result);
      setReviewResult(result);
      setReviewDialogOpen(true);
      console.log('✅ [REVIEW] Dialog should now be open');
    } catch (err) {
      console.error('❌ [REVIEW] Failed to review entry:', err);
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      alert(`Review failed: ${errorMessage}\n\nCheck browser console for details.`);
    } finally {
      setReviewing(false);
    }
  };

  const handleCompact = async () => {
    if (!entry) return;

    console.log('📦 [COMPACT] Starting compaction for entry:', entry.id);
    setCompacting(true);
    try {
      console.log('📦 [COMPACT] Calling API: POST /api/v1/knowledge/entries/{id}/compact');
      const result = await knowledgeService.compactEntry(entry.id, 500, true);
      console.log('✅ [COMPACT] Compaction result:', result);
      setCompactionResult(result);
      setCompactionDialogOpen(true);
      console.log('✅ [COMPACT] Dialog should now be open');
    } catch (err) {
      console.error('❌ [COMPACT] Failed to compact entry:', err);
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      alert(`Compaction failed: ${errorMessage}\n\nCheck browser console for details.`);
    } finally {
      setCompacting(false);
    }
  };

  const handleApplyCompaction = async () => {
    if (!entry) return;

    console.log('💾 [COMPACT APPLY] Applying compaction for entry:', entry.id);
    try {
      await knowledgeService.compactEntry(entry.id, 500, false);
      console.log('✅ [COMPACT APPLY] Compaction applied successfully');
      setCompactionDialogOpen(false);
      // Refresh the entry by triggering onEdit or similar
      window.location.reload(); // Temporary: ideally trigger a refresh callback
    } catch (err) {
      console.error('❌ [COMPACT APPLY] Failed to apply compaction:', err);
      const errorMessage = err instanceof Error ? err.message : 'Unknown error occurred';
      alert(`Failed to apply compaction: ${errorMessage}\n\nCheck browser console for details.`);
    }
  };

  const formatDate = (dateString?: string): string => {
    if (!dateString) return '';
    try {
      return new Date(dateString).toLocaleString();
    } catch {
      return '';
    }
  };

  return (
    <div className="h-full overflow-auto p-6">
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm p-6">
        {/* Header with actions */}
        <div className="flex justify-between items-center mb-4">
          <div className="flex gap-3 items-center">
            <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-200">
              {entry.id.split(':')[0] || 'Entry'}
            </span>
            {entry.createdAt && (
              <span className="text-xs text-gray-500 dark:text-gray-400">
                Created: {formatDate(entry.createdAt)}
              </span>
            )}
          </div>
          <div className="flex gap-2">
            {/* DEV: Mock test buttons */}
            <button
              onClick={handleReviewMock}
              className="inline-flex items-center justify-center px-2 py-1 rounded-md text-xs font-medium bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 hover:bg-yellow-200 dark:hover:bg-yellow-900/50 transition-colors"
              title="Test Review Dialog (Mock Data)"
            >
              🧪 Review
            </button>
            <button
              onClick={handleCompactMock}
              className="inline-flex items-center justify-center px-2 py-1 rounded-md text-xs font-medium bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 hover:bg-yellow-200 dark:hover:bg-yellow-900/50 transition-colors"
              title="Test Compact Dialog (Mock Data)"
            >
              🧪 Compact
            </button>
            {/* Real API buttons */}
            <button
              onClick={handleReview}
              disabled={reviewing || compacting}
              className="inline-flex items-center justify-center p-2 rounded-md text-cyan-600 dark:text-cyan-400 hover:bg-cyan-50 dark:hover:bg-cyan-900/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              title="Review Entry"
            >
              {reviewing ? (
                <Loader2 className="h-5 w-5 animate-spin" />
              ) : (
                <ClipboardCheck className="h-5 w-5" />
              )}
            </button>
            <button
              onClick={handleCompact}
              disabled={reviewing || compacting}
              className="inline-flex items-center justify-center p-2 rounded-md text-purple-600 dark:text-purple-400 hover:bg-purple-50 dark:hover:bg-purple-900/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              title="Compact Entry"
            >
              {compacting ? (
                <Loader2 className="h-5 w-5 animate-spin" />
              ) : (
                <Archive className="h-5 w-5" />
              )}
            </button>
            <button
              onClick={onEdit}
              className="inline-flex items-center justify-center p-2 rounded-md text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors"
              title="Edit"
            >
              <Edit className="h-5 w-5" />
            </button>
            <button
              onClick={handleDeleteClick}
              className="inline-flex items-center justify-center p-2 rounded-md text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
              title="Delete"
            >
              <Trash2 className="h-5 w-5" />
            </button>
          </div>
        </div>

        <div className="border-t border-gray-200 dark:border-gray-700 my-4" />

        {/* Markdown content */}
        <div className="prose prose-sm dark:prose-invert max-w-none">
          <div className="
            [&_h1]:text-3xl [&_h1]:font-semibold [&_h1]:mb-4 [&_h1]:mt-6
            [&_h2]:text-2xl [&_h2]:font-semibold [&_h2]:mb-3 [&_h2]:mt-5
            [&_h3]:text-xl [&_h3]:font-semibold [&_h3]:mb-2 [&_h3]:mt-4
            [&_h4]:text-lg [&_h4]:font-semibold [&_h4]:mb-2 [&_h4]:mt-3
            [&_p]:mb-4 [&_p]:leading-relaxed
            [&_code]:bg-gray-100 [&_code]:dark:bg-gray-900 [&_code]:text-red-600 [&_code]:dark:text-red-400 [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:rounded [&_code]:text-sm [&_code]:font-mono
            [&_pre]:bg-gray-100 [&_pre]:dark:bg-gray-900 [&_pre]:p-4 [&_pre]:rounded-lg [&_pre]:overflow-auto [&_pre]:mb-4
            [&_pre_code]:bg-transparent [&_pre_code]:text-gray-900 [&_pre_code]:dark:text-gray-100 [&_pre_code]:p-0
            [&_ul]:mb-4 [&_ul]:pl-6 [&_ol]:mb-4 [&_ol]:pl-6
            [&_li]:mb-1
            [&_blockquote]:border-l-4 [&_blockquote]:border-blue-500 [&_blockquote]:pl-4 [&_blockquote]:ml-0 [&_blockquote]:italic [&_blockquote]:text-gray-600 [&_blockquote]:dark:text-gray-400
            [&_table]:border-collapse [&_table]:w-full [&_table]:mb-4
            [&_th]:border [&_th]:border-gray-300 [&_th]:dark:border-gray-600 [&_th]:p-2 [&_th]:text-left [&_th]:bg-gray-100 [&_th]:dark:bg-gray-800 [&_th]:font-semibold
            [&_td]:border [&_td]:border-gray-300 [&_td]:dark:border-gray-600 [&_td]:p-2 [&_td]:text-left
          ">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
              {entry.text}
            </ReactMarkdown>
          </div>
        </div>

        {/* Metadata section with Radix Accordion */}
        {entry.metadata && Object.keys(entry.metadata).length > 0 && (
          <Accordion.Root type="single" collapsible className="mt-6">
            <Accordion.Item value="metadata" className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
              <Accordion.Header>
                <Accordion.Trigger className="flex items-center justify-between w-full px-4 py-3 text-left bg-gray-50 dark:bg-gray-800/50 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors group">
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                    Metadata
                  </span>
                  <ChevronDown className="h-4 w-4 text-gray-500 dark:text-gray-400 transition-transform duration-200 group-data-[state=open]:rotate-180" />
                </Accordion.Trigger>
              </Accordion.Header>
              <Accordion.Content className="px-4 py-3 bg-white dark:bg-gray-800 data-[state=open]:animate-accordion-down data-[state=closed]:animate-accordion-up">
                <div className="flex gap-2 flex-wrap">
                  {Object.entries(entry.metadata).map(([key, value]) => (
                    <span
                      key={key}
                      className="inline-flex items-center px-2.5 py-1 rounded text-xs font-medium bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-600"
                    >
                      {key}: {JSON.stringify(value)}
                    </span>
                  ))}
                </div>
              </Accordion.Content>
            </Accordion.Item>
          </Accordion.Root>
        )}
      </div>

      {/* Delete confirmation dialog with Radix Dialog */}
      <Dialog.Root open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/50 data-[state=open]:animate-fade-in" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6 w-full max-w-md data-[state=open]:animate-scale-in">
            <Dialog.Title className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
              Confirm Delete
            </Dialog.Title>
            <Dialog.Description className="text-sm text-gray-600 dark:text-gray-400 mb-6">
              Are you sure you want to delete this entry? This action cannot be undone.
            </Dialog.Description>
            <div className="flex justify-end gap-3">
              <Dialog.Close asChild>
                <button
                  onClick={handleDeleteCancel}
                  className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors"
                >
                  Cancel
                </button>
              </Dialog.Close>
              <button
                onClick={handleDeleteConfirm}
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

      {/* Review Result Dialog */}
      <ReviewResultDialog
        open={reviewDialogOpen}
        onClose={() => setReviewDialogOpen(false)}
        result={reviewResult}
      />

      {/* Compaction Dialog */}
      <CompactionDialog
        open={compactionDialogOpen}
        onClose={() => setCompactionDialogOpen(false)}
        result={compactionResult}
        onApprove={handleApplyCompaction}
        loading={compacting}
      />
    </div>
  );
};
