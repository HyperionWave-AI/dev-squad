import React, { useState, useEffect } from 'react';
import { knowledgeApi } from '../../services/knowledgeApi';
import type { KnowledgeCollection } from '../../types/knowledge';

interface KnowledgeCreateProps {
  collections: KnowledgeCollection[];
  onSuccess?: () => void;
}

interface MetadataEntry {
  key: string;
  value: string;
}

export const KnowledgeCreate: React.FC<KnowledgeCreateProps> = ({ collections, onSuccess }) => {
  const [selectedCollection, setSelectedCollection] = useState<string>('');
  const [text, setText] = useState<string>('');
  const [metadata, setMetadata] = useState<MetadataEntry[]>([{ key: '', value: '' }]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<boolean>(false);

  // Keyboard shortcut handler (Ctrl+Enter to submit)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault();
        handleSubmit(e as any);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [selectedCollection, text, metadata]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validation
    if (!selectedCollection) {
      setError('Please select a collection');
      return;
    }

    if (!text.trim()) {
      setError('Please enter knowledge text');
      return;
    }

    // Build metadata object (exclude empty entries)
    const metadataObj: Record<string, any> = {};
    metadata.forEach(({ key, value }) => {
      if (key.trim() && value.trim()) {
        metadataObj[key.trim()] = value.trim();
      }
    });

    setLoading(true);
    setError(null);
    setSuccess(false);

    try {
      await knowledgeApi.createKnowledge({
        collection: selectedCollection,
        text: text.trim(),
        metadata: Object.keys(metadataObj).length > 0 ? metadataObj : undefined
      });

      // Success: reset form and show confirmation
      setSuccess(true);
      setSelectedCollection('');
      setText('');
      setMetadata([{ key: '', value: '' }]);

      if (onSuccess) {
        onSuccess();
      }

      // Clear success message after 3 seconds
      setTimeout(() => setSuccess(false), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create knowledge entry');
    } finally {
      setLoading(false);
    }
  };

  const addMetadataEntry = () => {
    setMetadata([...metadata, { key: '', value: '' }]);
  };

  const removeMetadataEntry = (index: number) => {
    setMetadata(metadata.filter((_, i) => i !== index));
  };

  const updateMetadataEntry = (index: number, field: 'key' | 'value', newValue: string) => {
    const updated = [...metadata];
    updated[index][field] = newValue;
    setMetadata(updated);
  };

  const characterCount = text.length;
  const maxCharacters = 10000;

  return (
    <div className="p-4 border-2 rounded-lg bg-white shadow-sm">
      <h2 className="text-xl font-bold mb-4">Create Knowledge Entry</h2>

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Collection Select */}
        <div>
          <label htmlFor="collection" className="block text-sm font-semibold mb-1">
            Collection <span className="text-red-600">*</span>
          </label>
          <select
            id="collection"
            value={selectedCollection}
            onChange={(e) => setSelectedCollection(e.target.value)}
            className="w-full p-2 border-2 border-gray-300 rounded focus:border-blue-500 focus:outline-none"
            required
          >
            <option value="">Select a collection...</option>
            {collections
              .sort((a, b) => a.category.localeCompare(b.category))
              .map((col) => (
                <option key={col.name} value={col.name}>
                  {col.category} / {col.name}
                </option>
              ))}
          </select>
        </div>

        {/* Text Input */}
        <div>
          <label htmlFor="text" className="block text-sm font-semibold mb-1">
            Knowledge Text <span className="text-red-600">*</span>
          </label>
          <textarea
            id="text"
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="Enter detailed knowledge, patterns, or documentation..."
            className="w-full p-2 border-2 border-gray-300 rounded focus:border-blue-500 focus:outline-none resize-y"
            rows={6}
            maxLength={maxCharacters}
            required
          />
          <p className="text-xs text-gray-600 mt-1">
            {characterCount} / {maxCharacters} characters
          </p>
        </div>

        {/* Metadata Editor */}
        <div>
          <label className="block text-sm font-semibold mb-2">
            Metadata (optional)
          </label>
          <div className="space-y-2">
            {metadata.map((entry, index) => (
              <div key={index} className="flex gap-2">
                <input
                  type="text"
                  value={entry.key}
                  onChange={(e) => updateMetadataEntry(index, 'key', e.target.value)}
                  placeholder="Key"
                  className="flex-1 p-2 border-2 border-gray-300 rounded focus:border-blue-500 focus:outline-none"
                />
                <input
                  type="text"
                  value={entry.value}
                  onChange={(e) => updateMetadataEntry(index, 'value', e.target.value)}
                  placeholder="Value"
                  className="flex-1 p-2 border-2 border-gray-300 rounded focus:border-blue-500 focus:outline-none"
                />
                {metadata.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeMetadataEntry(index)}
                    className="px-3 py-2 bg-red-100 text-red-700 rounded font-semibold hover:bg-red-200 transition-colors"
                    aria-label="Remove metadata entry"
                  >
                    Remove
                  </button>
                )}
              </div>
            ))}
          </div>
          <button
            type="button"
            onClick={addMetadataEntry}
            className="mt-2 px-3 py-1 bg-gray-100 text-gray-700 rounded text-sm font-semibold hover:bg-gray-200 transition-colors"
          >
            + Add Metadata
          </button>
        </div>

        {/* Submit Button */}
        <div className="flex gap-2 pt-2">
          <button
            type="submit"
            disabled={loading}
            className="px-6 py-2 bg-green-600 text-white rounded font-semibold hover:bg-green-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? 'Creating...' : 'Create Knowledge'}
          </button>
          <p className="text-xs text-gray-500 self-center">
            Press Ctrl+Enter to submit
          </p>
        </div>
      </form>

      {/* Success Message */}
      {success && (
        <div className="mt-4 p-4 border-2 border-green-300 bg-green-50 rounded-lg">
          <p className="text-green-800 font-semibold">✓ Knowledge entry created successfully!</p>
        </div>
      )}

      {/* Error Display */}
      {error && (
        <div className="mt-4 p-4 border-2 border-red-300 bg-red-50 rounded-lg">
          <p className="text-red-800 font-semibold">Error: {error}</p>
        </div>
      )}
    </div>
  );
};
