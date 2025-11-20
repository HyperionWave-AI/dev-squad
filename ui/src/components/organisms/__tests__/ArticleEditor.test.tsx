/**
 * Unit tests for ArticleEditor component
 * Tests rendering, interactions, validation, and state management
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ArticleEditor } from '../ArticleEditor';
import type { KnowledgeEntry } from '@/types/knowledge';
import { knowledgeApi } from '@/services/knowledgeApi';

// Mock the knowledgeApi
vi.mock('@/services/knowledgeApi', () => ({
  knowledgeApi: {
    compactText: vi.fn(),
    compactEntry: vi.fn(),
  },
}));

describe('ArticleEditor', () => {
  const mockOnSave = vi.fn();
  const mockOnCancel = vi.fn();
  const mockCollections = [
    { name: 'collection-1', count: 5 },
    { name: 'collection-2', count: 3 },
  ];

  const mockExistingEntry: KnowledgeEntry = {
    id: 'collection-1:entry-123',
    text: 'Existing entry content',
    metadata: { title: 'Test Entry', tags: ['test'] },
    embedding: [],
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  };

  const mockNewEntry: KnowledgeEntry = {
    id: 'new-temp-123',
    text: '',
    metadata: {},
    embedding: [],
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockOnSave.mockResolvedValue(undefined);
  });

  describe('Rendering', () => {
    it('should render editor with existing entry data', () => {
      render(
        <ArticleEditor
          entry={mockExistingEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      expect(screen.getByText('Edit Entry')).toBeInTheDocument();
      expect(screen.getByDisplayValue('Existing entry content')).toBeInTheDocument();
      expect(screen.getByText(/Collection: collection-1/)).toBeInTheDocument();
    });

    it('should render empty editor for new entry', () => {
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      expect(screen.getByText('Create New Entry')).toBeInTheDocument();
      const textarea = screen.getByPlaceholderText('Enter markdown content...');
      expect(textarea).toHaveValue('');
    });

    it('should display collection selector for new entries', () => {
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      expect(screen.getByText('Collection')).toBeInTheDocument();
      const select = screen.getByRole('combobox');
      expect(select).toBeInTheDocument();
      expect(screen.getByText('collection-1 (5 entries)')).toBeInTheDocument();
      expect(screen.getByText('collection-2 (3 entries)')).toBeInTheDocument();
    });

    it('should not display collection selector for existing entries', () => {
      render(
        <ArticleEditor
          entry={mockExistingEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
      expect(screen.getByText(/Collection: collection-1/)).toBeInTheDocument();
    });

    it('should show save and cancel buttons', () => {
      render(
        <ArticleEditor
          entry={mockExistingEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument();
    });

    it('should display token counter', () => {
      render(
        <ArticleEditor
          entry={mockExistingEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      expect(screen.getByText(/~\d+ \/ 1000 tokens/)).toBeInTheDocument();
    });

    it('should show no collections message when collections array is empty', () => {
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={[]}
        />
      );

      expect(screen.getByText('No collections available')).toBeInTheDocument();
    });
  });

  describe('Interactions', () => {
    it('should allow editing text in textarea', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter markdown content...');
      await user.type(textarea, 'New content');

      expect(textarea).toHaveValue('New content');
    });

    it('should allow editing metadata JSON field', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const metadataTextarea = screen.getByPlaceholderText('{"key": "value"}') as HTMLTextAreaElement;
      await user.clear(metadataTextarea);
      // Use paste instead of type to avoid issues with special characters
      await user.click(metadataTextarea);
      await user.paste('{"title": "My Article"}');

      expect(metadataTextarea).toHaveValue('{"title": "My Article"}');
    });

    it('should allow selecting collection for new entry', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const select = screen.getByRole('combobox');
      await user.selectOptions(select, 'collection-2');

      expect(select).toHaveValue('collection-2');
    });

    it('should call onSave with updated data when save clicked', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter markdown content...');
      await user.type(textarea, 'Test content');

      const saveButton = screen.getByRole('button', { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(mockOnSave).toHaveBeenCalledWith(
          'Test content',
          {},
          'collection-1'
        );
      });
    });

    it('should call onCancel when cancel button clicked', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockExistingEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      await user.click(cancelButton);

      expect(mockOnCancel).toHaveBeenCalledTimes(1);
    });

    it('should switch between edit and preview tabs', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockExistingEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      // Verify Edit tab is active initially
      const editTab = screen.getByRole('tab', { name: /edit/i });
      expect(editTab).toHaveAttribute('aria-selected', 'true');

      const previewTab = screen.getByRole('tab', { name: /preview/i });
      await user.click(previewTab);

      // Preview tab should now be active
      expect(previewTab).toHaveAttribute('aria-selected', 'true');
      expect(editTab).toHaveAttribute('aria-selected', 'false');
    });
  });

  describe('Validation', () => {
    it('should display validation error for empty text', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const saveButton = screen.getByRole('button', { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(screen.getByText('Content cannot be empty')).toBeInTheDocument();
      });
      expect(mockOnSave).not.toHaveBeenCalled();
    });

    it('should display validation error for missing collection on new entry', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={[]}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter markdown content...');
      await user.type(textarea, 'Some content');

      const saveButton = screen.getByRole('button', { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(screen.getByText('Please select a collection')).toBeInTheDocument();
      });
      expect(mockOnSave).not.toHaveBeenCalled();
    });

    it('should validate metadata JSON syntax', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const metadataTextarea = screen.getByPlaceholderText('{"key": "value"}') as HTMLTextAreaElement;
      await user.clear(metadataTextarea);
      await user.click(metadataTextarea);
      await user.paste('{invalid json}');

      await waitFor(() => {
        expect(screen.getByText('Invalid JSON syntax')).toBeInTheDocument();
      });
    });

    it('should validate metadata is an object not an array', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const metadataTextarea = screen.getByPlaceholderText('{"key": "value"}') as HTMLTextAreaElement;
      await user.clear(metadataTextarea);
      await user.click(metadataTextarea);
      await user.paste('["not", "an", "object"]');

      await waitFor(() => {
        expect(screen.getByText('Metadata must be a JSON object')).toBeInTheDocument();
      });
    });

    it('should disable save button when metadata has errors', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter markdown content...');
      await user.type(textarea, 'Valid content');

      const metadataTextarea = screen.getByPlaceholderText('{"key": "value"}') as HTMLTextAreaElement;
      await user.clear(metadataTextarea);
      await user.click(metadataTextarea);
      await user.paste('{invalid}');

      await waitFor(() => {
        const saveButton = screen.getByRole('button', { name: /save/i });
        expect(saveButton).toBeDisabled();
      });
    });

    it('should show warning when content exceeds token limit', async () => {
      const user = userEvent.setup();
      const longContent = 'a'.repeat(5000); // ~1250 tokens
      const longEntry = { ...mockNewEntry, text: longContent };

      render(
        <ArticleEditor
          entry={longEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      // Check for token count warning in the badge
      expect(screen.getByText(/\(over limit!\)/i)).toBeInTheDocument();
      expect(screen.getByText(/Compact with AI/i)).toBeInTheDocument();
    });

    it('should disable save button when over token limit', async () => {
      const longContent = 'a'.repeat(5000); // ~1250 tokens
      const longEntry = { ...mockNewEntry, text: longContent };

      render(
        <ArticleEditor
          entry={longEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const saveButton = screen.getByRole('button', { name: /over limit/i });
      expect(saveButton).toBeDisabled();
    });
  });

  describe('Error Handling', () => {
    it('should display save failure error message', async () => {
      const user = userEvent.setup();
      mockOnSave.mockRejectedValueOnce(new Error('Save failed'));

      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter markdown content...');
      await user.type(textarea, 'Test content');

      const saveButton = screen.getByRole('button', { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(screen.getByText('Save failed')).toBeInTheDocument();
      });
    });

    it('should handle onSave rejection gracefully', async () => {
      const user = userEvent.setup();
      mockOnSave.mockRejectedValueOnce(new Error('Network error'));

      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter markdown content...');
      await user.type(textarea, 'Test content');

      const saveButton = screen.getByRole('button', { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(screen.getByText('Network error')).toBeInTheDocument();
      });

      // Should remain in editor mode, not close
      expect(screen.getByText('Create New Entry')).toBeInTheDocument();
    });

    it('should allow dismissing error messages', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const saveButton = screen.getByRole('button', { name: /save/i });
      await user.click(saveButton);

      await waitFor(() => {
        expect(screen.getByText('Content cannot be empty')).toBeInTheDocument();
      });

      // Find and click the X button to dismiss error
      const dismissButtons = screen.getAllByRole('button');
      const dismissButton = dismissButtons.find(btn => btn.querySelector('svg'));
      if (dismissButton) {
        await user.click(dismissButton);
        await waitFor(() => {
          expect(screen.queryByText('Content cannot be empty')).not.toBeInTheDocument();
        });
      }
    });
  });

  describe('State Management', () => {
    it('should preserve text changes when switching tabs', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter markdown content...');
      await user.type(textarea, 'Test content');

      const previewTab = screen.getByRole('tab', { name: /preview/i });
      await user.click(previewTab);

      const editTab = screen.getByRole('tab', { name: /edit/i });
      await user.click(editTab);

      expect(screen.getByPlaceholderText('Enter markdown content...')).toHaveValue('Test content');
    });

    it('should preserve changes when collection changed', async () => {
      const user = userEvent.setup();
      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter markdown content...');
      await user.type(textarea, 'Test content');

      const select = screen.getByRole('combobox');
      await user.selectOptions(select, 'collection-2');

      expect(textarea).toHaveValue('Test content');
    });

    it('should show saving state on save button during save', async () => {
      const user = userEvent.setup();
      let resolveSave: () => void;
      const savePromise = new Promise<void>((resolve) => {
        resolveSave = resolve;
      });
      mockOnSave.mockReturnValue(savePromise);

      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter markdown content...');
      await user.type(textarea, 'Test content');

      const saveButton = screen.getByRole('button', { name: /save/i });
      await user.click(saveButton);

      expect(screen.getByRole('button', { name: /saving/i })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /saving/i })).toBeDisabled();

      resolveSave!();
    });

    it('should disable cancel button during save', async () => {
      const user = userEvent.setup();
      let resolveSave: () => void;
      const savePromise = new Promise<void>((resolve) => {
        resolveSave = resolve;
      });
      mockOnSave.mockReturnValue(savePromise);

      render(
        <ArticleEditor
          entry={mockNewEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const textarea = screen.getByPlaceholderText('Enter markdown content...');
      await user.type(textarea, 'Test content');

      const saveButton = screen.getByRole('button', { name: /save/i });
      await user.click(saveButton);

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      expect(cancelButton).toBeDisabled();

      resolveSave!();
    });
  });

  describe('AI Compacting', () => {
    it('should show compact button when over token limit', () => {
      const longContent = 'a'.repeat(5000); // ~1250 tokens
      const longEntry = { ...mockNewEntry, text: longContent };

      render(
        <ArticleEditor
          entry={longEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      expect(screen.getByText(/Compact with AI/i)).toBeInTheDocument();
    });

    it('should call compactText API for new entries', async () => {
      const user = userEvent.setup();
      const longContent = 'a'.repeat(5000);
      const longEntry = { ...mockNewEntry, text: longContent };

      vi.mocked(knowledgeApi.compactText).mockResolvedValue({
        success: true,
        original: { text: longContent, wordCount: 1250, charCount: 5000, tokenCount: 1250 },
        compacted: { text: 'Compacted content', wordCount: 100, charCount: 400, tokenCount: 100 },
        compressionRatio: 0.08,
        preserved: { filePaths: 0, functionNames: 0, codeBlocks: 0 },
      });

      render(
        <ArticleEditor
          entry={longEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const compactButton = screen.getByText(/Compact with AI/i);
      await user.click(compactButton);

      await waitFor(() => {
        expect(knowledgeApi.compactText).toHaveBeenCalledWith(longContent, 750, false);
      });
    });

    it('should update text with compacted content on success', async () => {
      const user = userEvent.setup();
      const longContent = 'a'.repeat(5000);
      const longEntry = { ...mockNewEntry, text: longContent };

      vi.mocked(knowledgeApi.compactText).mockResolvedValue({
        success: true,
        original: { text: longContent, wordCount: 1250, charCount: 5000, tokenCount: 1250 },
        compacted: { text: 'Compacted content', wordCount: 100, charCount: 400, tokenCount: 100 },
        compressionRatio: 0.08,
        preserved: { filePaths: 2, functionNames: 3, codeBlocks: 1 },
      });

      render(
        <ArticleEditor
          entry={longEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const compactButton = screen.getByText(/Compact with AI/i);
      await user.click(compactButton);

      await waitFor(() => {
        const textarea = screen.getByPlaceholderText('Enter markdown content...');
        expect(textarea).toHaveValue('Compacted content');
        expect(screen.getByText(/Compacted successfully/i)).toBeInTheDocument();
        expect(screen.getByText(/Reduced by 92%/i)).toBeInTheDocument();
      });
    });

    it('should show error message if compacting fails', async () => {
      const user = userEvent.setup();
      const longContent = 'a'.repeat(5000);
      const longEntry = { ...mockNewEntry, text: longContent };

      vi.mocked(knowledgeApi.compactText).mockRejectedValue(new Error('Compact failed'));

      render(
        <ArticleEditor
          entry={longEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const compactButton = screen.getByText(/Compact with AI/i);
      await user.click(compactButton);

      await waitFor(() => {
        expect(screen.getByText('Compact failed')).toBeInTheDocument();
      });
    });

    it('should call compactEntry API for existing entries', async () => {
      const user = userEvent.setup();
      const longContent = 'a'.repeat(5000);
      const longEntry = { ...mockExistingEntry, text: longContent };

      vi.mocked(knowledgeApi.compactEntry).mockResolvedValue({
        success: true,
        original: { text: longContent, wordCount: 1250, charCount: 5000, tokenCount: 1250 },
        compacted: { text: 'Compacted content', wordCount: 100, charCount: 400, tokenCount: 100 },
        compressionRatio: 0.08,
        preserved: { filePaths: 0, functionNames: 0, codeBlocks: 0 },
      });

      render(
        <ArticleEditor
          entry={longEntry}
          onSave={mockOnSave}
          onCancel={mockOnCancel}
          collections={mockCollections}
        />
      );

      const compactButton = screen.getByText(/Compact with AI/i);
      await user.click(compactButton);

      await waitFor(() => {
        expect(knowledgeApi.compactEntry).toHaveBeenCalledWith(mockExistingEntry.id, 750, false);
      });
    });
  });
});
