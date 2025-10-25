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
    <div className="w-full max-w-4xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <h2 className="text-2xl font-semibold text-gray-900 mb-2">
          Create Knowledge Entry
        </h2>
        <p className="text-sm text-gray-600">
          Add new knowledge to your collection with proper metadata and categorization.
        </p>
      </div>

      {/* Main Form Card */}
      <div className="bg-white border border-gray-200 rounded-lg shadow-sm">
        <form onSubmit={handleSubmit} className="p-6 space-y-6" noValidate>
          {/* Collection Select */}
          <div className="space-y-2">
            <label htmlFor="collection" className="block text-sm font-medium text-gray-700">
              Collection <span className="text-red-500">*</span>
            </label>
            <select
              id="collection"
              value={selectedCollection}
              onChange={(e) => setSelectedCollection(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors"
              required
              aria-describedby="collection-help"
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
            <p id="collection-help" className="text-xs text-gray-500">
              Choose the collection where this knowledge will be stored
            </p>
          </div>

          {/* Text Input */}
          <div className="space-y-2">
            <label htmlFor="text" className="block text-sm font-medium text-gray-700">
              Knowledge Text <span className="text-red-500">*</span>
            </label>
            <textarea
              id="text"
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Enter detailed knowledge, patterns, or documentation..."
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors resize-vertical"
              rows={8}
              maxLength={maxCharacters}
              required
              aria-describedby="text-help text-count"
            />
            <div className="flex justify-between items-center">
              <p id="text-help" className="text-xs text-gray-500">
                Provide comprehensive information that will be useful for future reference
              </p>
              <p id="text-count" className="text-xs text-gray-500">
                {characterCount} / {maxCharacters} characters
              </p>
            </div>
          </div>

          {/* Metadata Editor */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <label className="block text-sm font-medium text-gray-700">
                Metadata (optional)
              </label>
              <button
                type="button"
                onClick={addMetadataEntry}
                className="inline-flex items-center px-3 py-1.5 text-xs font-medium text-blue-600 bg-blue-50 border border-blue-200 rounded-md hover:bg-blue-100 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-colors"
              >
                + Add Metadata
              </button>
            </div>
            
            <div className="space-y-3">
              {metadata.map((entry, index) => (
                <div key={index} className="flex gap-3 items-start">
                  <div className="flex-1">
                    <input
                      type="text"
                      value={entry.key}
                      onChange={(e) => updateMetadataEntry(index, 'key', e.target.value)}
                      placeholder="Key"
                      className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors"
                      aria-label={`Metadata key ${index + 1}`}
                    />
                  </div>
                  <div className="flex-1">
                    <input
                      type="text"
                      value={entry.value}
                      onChange={(e) => updateMetadataEntry(index, 'value', e.target.value)}
                      placeholder="Value"
                      className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-colors"
                      aria-label={`Metadata value ${index + 1}`}
                    />
                  </div>
                  {metadata.length > 1 && (
                    <button
                      type="button"
                      onClick={() => removeMetadataEntry(index)}
                      className="px-3 py-2 text-sm font-medium text-red-600 bg-red-50 border border-red-200 rounded-md hover:bg-red-100 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 transition-colors"
                      aria-label={`Remove metadata entry ${index + 1}`}
                    >
                      Remove
                    </button>
                  )}
                </div>
              ))}
            </div>
            
            <p className="text-xs text-gray-500">
              Add key-value pairs to provide additional context and searchable attributes
            </p>
          </div>

          {/* Submit Section */}
          <div className="pt-4 border-t border-gray-200">
            <div className="flex items-center justify-between">
              <p className="text-xs text-gray-500">
                Press <kbd className="px-1.5 py-0.5 text-xs font-mono bg-gray-100 border border-gray-300 rounded">Ctrl+Enter</kbd> to submit
              </p>
              <button
                type="submit"
                disabled={loading || !selectedCollection || !text.trim()}
                className="inline-flex items-center px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                {loading ? (
                  <>
                    <svg className="w-4 h-4 mr-2 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    Creating...
                  </>
                ) : (
                  'Create Knowledge'
                )}
              </button>
            </div>
          </div>
        </form>
      </div>

      {/* Success Message */}
      {success && (
        <div className="mt-4 p-4 bg-green-50 border border-green-200 rounded-md" role="alert">
          <div className="flex items-center">
            <svg className="w-5 h-5 text-green-400 mr-2" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
            </svg>
            <p className="text-sm font-medium text-green-800">
              Knowledge entry created successfully!
            </p>
          </div>
        </div>
      )}

      {/* Error Display */}
      {error && (
        <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-md" role="alert">
          <div className="flex items-center">
            <svg className="w-5 h-5 text-red-400 mr-2" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
            </svg>
            <p className="text-sm font-medium text-red-800">
              Error: {error}
            </p>
          </div>
        </div>
      )}
    </div>
  );
};