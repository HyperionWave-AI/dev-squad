import React, { useState } from 'react';
import * as Tabs from '@radix-ui/react-tabs';
import { Save, X } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { KnowledgeEntry } from '@/types/knowledge';

interface ArticleEditorProps {
  entry: KnowledgeEntry;
  onSave: (text: string, metadata: Record<string, any>, collection?: string) => Promise<void>;
  onCancel: () => void;
  collections?: Array<{ name: string; count: number }>;
  selectedCollection?: string | null;
}

export const ArticleEditor: React.FC<ArticleEditorProps> = ({
  entry,
  onSave,
  onCancel,
  collections = [],
  selectedCollection = null,
}) => {
  const isNewEntry = entry.id.startsWith('new-');
  const [text, setText] = useState(entry.text);
  const [metadataJson, setMetadataJson] = useState(
    JSON.stringify(entry.metadata || {}, null, 2)
  );
  const [targetCollection, setTargetCollection] = useState<string>(
    selectedCollection || (collections.length > 0 ? collections[0].name : '')
  );
  const [activeTab, setActiveTab] = useState('edit');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [metadataError, setMetadataError] = useState<string | null>(null);

  // Token estimation (1 token ≈ 4 characters)
  const estimatedTokens = Math.ceil(text.length / 4);
  const maxTokens = 1000;
  const isOverLimit = estimatedTokens > maxTokens;

  const validateMetadata = (json: string): Record<string, any> | null => {
    try {
      const parsed = JSON.parse(json);
      if (typeof parsed !== 'object' || Array.isArray(parsed)) {
        setMetadataError('Metadata must be a JSON object');
        return null;
      }
      setMetadataError(null);
      return parsed;
    } catch (e) {
      setMetadataError('Invalid JSON syntax');
      return null;
    }
  };

  const handleMetadataChange = (value: string) => {
    setMetadataJson(value);
    validateMetadata(value);
  };

  const handleSave = async () => {
    if (!text.trim()) {
      setError('Content cannot be empty');
      return;
    }

    if (isOverLimit) {
      setError(`Entry is too large: ~${estimatedTokens} tokens (max: ${maxTokens} tokens). Please split into multiple focused entries or reduce content.`);
      return;
    }

    if (isNewEntry && !targetCollection) {
      setError('Please select a collection');
      return;
    }

    const metadata = validateMetadata(metadataJson);
    if (metadata === null) {
      return;
    }

    setSaving(true);
    setError(null);

    try {
      await onSave(text, metadata, isNewEntry ? targetCollection : undefined);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save entry');
      setSaving(false);
    }
  };

  return (
    <div className="h-full overflow-auto p-6">
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm p-6">
        {/* Header */}
        <div className="mb-4">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">
            {isNewEntry ? 'Create New Entry' : 'Edit Entry'}
          </h2>
          {isNewEntry ? (
            <div className="mt-3">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Collection
              </label>
              <select
                value={targetCollection}
                onChange={(e) => setTargetCollection(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400"
              >
                {collections.length === 0 ? (
                  <option value="">No collections available</option>
                ) : (
                  collections.map((c) => (
                    <option key={c.name} value={c.name}>
                      {c.name} ({c.count} entries)
                    </option>
                  ))
                )}
              </select>
            </div>
          ) : (
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Collection: {entry.id.split(':')[0] || 'Unknown'}
            </p>
          )}
        </div>

        <div className="border-t border-gray-200 dark:border-gray-700 my-4" />

        {/* Error display */}
        {error && (
          <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-start justify-between">
            <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
            <button
              onClick={() => setError(null)}
              className="text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-200"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        )}

        {/* Content editor with preview using Radix Tabs */}
        <div className="mb-6">
          <div className="flex justify-between items-center mb-3">
            <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">
              Content
            </h3>
            <div className={`text-xs font-medium px-3 py-1 rounded-full ${
              isOverLimit
                ? 'bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 border border-red-300 dark:border-red-700'
                : estimatedTokens > 800
                ? 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300 border border-yellow-300 dark:border-yellow-700'
                : 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 border border-blue-300 dark:border-blue-700'
            }`}>
              {isOverLimit ? '⚠️ ' : ''}
              ~{estimatedTokens} / {maxTokens} tokens
              {isOverLimit && ' (over limit!)'}
            </div>
          </div>

          <Tabs.Root value={activeTab} onValueChange={setActiveTab}>
            <Tabs.List className="flex border-b border-gray-200 dark:border-gray-700 mb-4">
              <Tabs.Trigger
                value="edit"
                className="px-4 py-2 text-sm font-medium text-gray-600 dark:text-gray-400 border-b-2 border-transparent data-[state=active]:text-blue-600 data-[state=active]:dark:text-blue-400 data-[state=active]:border-blue-600 data-[state=active]:dark:border-blue-400 hover:text-gray-900 dark:hover:text-gray-200 transition-colors"
              >
                Edit
              </Tabs.Trigger>
              <Tabs.Trigger
                value="preview"
                className="px-4 py-2 text-sm font-medium text-gray-600 dark:text-gray-400 border-b-2 border-transparent data-[state=active]:text-blue-600 data-[state=active]:dark:text-blue-400 data-[state=active]:border-blue-600 data-[state=active]:dark:border-blue-400 hover:text-gray-900 dark:hover:text-gray-200 transition-colors"
              >
                Preview
              </Tabs.Trigger>
            </Tabs.List>

            <Tabs.Content value="edit">
              <textarea
                value={text}
                onChange={(e) => setText(e.target.value)}
                placeholder="Enter markdown content..."
                rows={20}
                className="w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent resize-none"
              />
            </Tabs.Content>

            <Tabs.Content value="preview">
              <div className="border border-gray-300 dark:border-gray-600 rounded-lg p-4 min-h-[400px] max-h-[600px] overflow-auto bg-gray-50 dark:bg-gray-900">
                <div className="prose prose-sm dark:prose-invert max-w-none">
                  <div className="
                    [&_h1]:text-3xl [&_h1]:font-semibold [&_h1]:mb-4 [&_h1]:mt-6
                    [&_h2]:text-2xl [&_h2]:font-semibold [&_h2]:mb-3 [&_h2]:mt-5
                    [&_h3]:text-xl [&_h3]:font-semibold [&_h3]:mb-2 [&_h3]:mt-4
                    [&_h4]:text-lg [&_h4]:font-semibold [&_h4]:mb-2 [&_h4]:mt-3
                    [&_p]:mb-4 [&_p]:leading-relaxed
                    [&_code]:bg-gray-100 [&_code]:dark:bg-gray-800 [&_code]:text-red-600 [&_code]:dark:text-red-400 [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:rounded [&_code]:text-sm [&_code]:font-mono
                    [&_pre]:bg-gray-100 [&_pre]:dark:bg-gray-800 [&_pre]:p-4 [&_pre]:rounded-lg [&_pre]:overflow-auto [&_pre]:mb-4
                    [&_pre_code]:bg-transparent [&_pre_code]:text-gray-900 [&_pre_code]:dark:text-gray-100 [&_pre_code]:p-0
                    [&_ul]:mb-4 [&_ul]:pl-6 [&_ol]:mb-4 [&_ol]:pl-6
                    [&_li]:mb-1
                    [&_blockquote]:border-l-4 [&_blockquote]:border-blue-500 [&_blockquote]:pl-4 [&_blockquote]:ml-0 [&_blockquote]:italic [&_blockquote]:text-gray-600 [&_blockquote]:dark:text-gray-400
                    [&_table]:border-collapse [&_table]:w-full [&_table]:mb-4
                    [&_th]:border [&_th]:border-gray-300 [&_th]:dark:border-gray-600 [&_th]:p-2 [&_th]:text-left [&_th]:bg-gray-100 [&_th]:dark:bg-gray-700 [&_th]:font-semibold
                    [&_td]:border [&_td]:border-gray-300 [&_td]:dark:border-gray-600 [&_td]:p-2 [&_td]:text-left
                  ">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>
                      {text || '*No content to preview*'}
                    </ReactMarkdown>
                  </div>
                </div>
              </div>
            </Tabs.Content>
          </Tabs.Root>
        </div>

        {/* Metadata editor */}
        <div className="mb-6">
          <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">
            Metadata (JSON)
          </h3>
          <textarea
            value={metadataJson}
            onChange={(e) => handleMetadataChange(e.target.value)}
            placeholder='{"key": "value"}'
            rows={6}
            className={`w-full px-4 py-3 border rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent resize-none ${
              metadataError
                ? 'border-red-300 dark:border-red-600'
                : 'border-gray-300 dark:border-gray-600'
            }`}
          />
          {metadataError && (
            <p className="mt-2 text-sm text-red-600 dark:text-red-400">
              {metadataError}
            </p>
          )}
        </div>

        <div className="border-t border-gray-200 dark:border-gray-700 my-4" />

        {/* Action buttons */}
        <div className="flex justify-end gap-3">
          <button
            onClick={onCancel}
            disabled={saving}
            className="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={saving || !!metadataError || isOverLimit}
            className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Save className="h-4 w-4" />
            {saving ? 'Saving...' : isOverLimit ? 'Over Limit' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  );
};
