/**
 * Knowledge Base Service
 *
 * Handles API calls for knowledge base operations including:
 * - Browsing collections and entries
 * - Querying knowledge base
 * - Creating, updating, and deleting entries
 */

import type { SyncReport, ExportReport } from '@/types/knowledge';

export interface KnowledgeEntry {
  id: string;
  collection: string;
  text: string;
  metadata?: Record<string, any>;
  createdAt?: string;
}

export interface CollectionInfo {
  id: string;
  name: string;
  category: string;
  count: number;
  description?: string;
  tags?: string[];
}

export interface GetCollectionsResponse {
  collections: CollectionInfo[];
}

export interface GetEntriesResponse {
  entries: KnowledgeEntry[];
}

export interface UpdateEntryRequest {
  text: string;
  metadata?: Record<string, any>;
}

export interface UpdateEntryResponse {
  entry: KnowledgeEntry;
}

export interface UpdateCollectionMetadataRequest {
  description: string;
  tags: string[];
  category: string;
}

export interface RenameCollectionRequest {
  newName: string;
}

export interface RenameCollectionResponse {
  message: string;
  oldName: string;
  newName: string;
  entriesUpdated: number;
}

export interface ReviewResult {
  success: boolean;
  entryId: string;
  scores: {
    alignment: number;
    freshness: number;
    verbosity: number;
    uniqueness: number;
    health: number;
  };
  verification: {
    totalReferences: number;
    validReferences: number;
    brokenReferences: Array<{type: string; value: string; error: string}>;
  };
  actions: Array<{type: string; description: string; applied: boolean}>;
}

export interface CollectionReviewResult {
  success: boolean;
  collection: string;
  summary: {
    totalEntries: number;
    entriesReviewed: number;
    averageHealth: number;
    lowScoreCount: number;
  };
  entries: Array<{
    entryId: string;
    healthScore: number;
    issues: string[];
  }>;
}

export interface CompactionResult {
  success: boolean;
  original: {wordCount: number; text: string};
  compacted: {wordCount: number; text: string};
  compressionRatio: number;
  preserved: {filePaths: number; functionNames: number};
}

class KnowledgeService {
  /**
   * Get all collections with metadata and entry counts
   */
  async getCollections(): Promise<GetCollectionsResponse> {
    const response = await fetch('/api/v1/knowledge/collections', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch collections');
    }

    return response.json();
  }

  /**
   * Get entries from a specific collection
   */
  async getEntries(collection: string, limit: number = 50): Promise<GetEntriesResponse> {
    const params = new URLSearchParams({
      collection,
      limit: limit.toString(),
    });

    const response = await fetch(`/api/v1/knowledge/browse?${params}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to fetch entries');
    }

    return response.json();
  }

  /**
   * Update an existing knowledge entry
   */
  async updateEntry(id: string, data: UpdateEntryRequest): Promise<UpdateEntryResponse> {
    const response = await fetch(`/api/v1/knowledge/entries/${encodeURIComponent(id)}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || error.details || 'Failed to update entry');
    }

    return response.json();
  }

  /**
   * Delete a knowledge entry
   */
  async deleteEntry(id: string): Promise<void> {
    const response = await fetch(`/api/v1/knowledge/entries/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok && response.status !== 204) {
      const error = await response.json();
      throw new Error(error.error || error.details || 'Failed to delete entry');
    }
  }

  /**
   * Query knowledge base (semantic search)
   */
  async queryKnowledge(collection: string, query: string, limit: number = 10): Promise<GetEntriesResponse> {
    const response = await fetch('/api/v1/knowledge/query', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        collection,
        query,
        limit,
      }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to query knowledge base');
    }

    return response.json();
  }

  /**
   * Update collection metadata (description, tags, category)
   */
  async updateCollectionMetadata(
    collectionName: string,
    data: UpdateCollectionMetadataRequest
  ): Promise<any> {
    const response = await fetch(`/api/v1/knowledge/collections/${encodeURIComponent(collectionName)}/metadata`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to update collection metadata');
    }

    return response.json();
  }

  /**
   * Rename a collection
   */
  async renameCollection(
    oldName: string,
    newName: string
  ): Promise<RenameCollectionResponse> {
    const response = await fetch(`/api/v1/knowledge/collections/${encodeURIComponent(oldName)}/rename`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ newName }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to rename collection');
    }

    return response.json();
  }

  /**
   * Delete an entire collection including all MongoDB and Qdrant data
   */
  async deleteCollection(id: string): Promise<{ message: string; collectionId: string; collectionName: string; entriesDeleted: number }> {
    const response = await fetch(`/api/v1/knowledge/collections/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to delete collection');
    }

    return response.json();
  }

  /**
   * Review a single knowledge entry
   */
  async reviewEntry(
    entryId: string,
    mode: string = 'full',
    dryRun: boolean = false
  ): Promise<ReviewResult> {
    const response = await fetch(`/api/v1/knowledge/entries/${encodeURIComponent(entryId)}/review`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ mode, dryRun }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to review entry');
    }

    return response.json();
  }

  /**
   * Review all entries in a collection
   */
  async reviewCollection(
    collection: string,
    minHealthScore: number = 70,
    limit: number = 100
  ): Promise<CollectionReviewResult> {
    const response = await fetch(`/api/v1/knowledge/collections/${encodeURIComponent(collection)}/review`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ minHealthScore, limit }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to review collection');
    }

    return response.json();
  }

  /**
   * Compact a knowledge entry (reduce verbosity)
   */
  async compactEntry(
    entryId: string,
    targetWordCount: number = 500,
    dryRun: boolean = true
  ): Promise<CompactionResult> {
    const response = await fetch(`/api/v1/knowledge/entries/${encodeURIComponent(entryId)}/compact`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ targetWordCount, dryRun }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to compact entry');
    }

    return response.json();
  }

  /**
   * Verify a knowledge article by creating a chat session
   */
  async verifyKnowledgeArticle(id: string): Promise<{ sessionId: string }> {
    const response = await fetch(`/api/v1/knowledge/${encodeURIComponent(id)}/verify`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to verify knowledge article');
    }

    return response.json();
  }

  /**
   * Universal search across all collections
   * Returns entries from all collections that match the query, limited to top 100 results
   */
  async universalSearch(query: string, limit: number = 100): Promise<{entries: KnowledgeEntry[], collectionsWithData: string[]}> {
    // Get all collections first
    const collectionsResponse = await this.getCollections();
    const allCollections = collectionsResponse.collections
      .filter(c => !c.name.startsWith('task:')) // Exclude task-specific collections
      .map(c => c.name);

    // Search each collection
    const searchPromises = allCollections.map(async (collection) => {
      try {
        const response = await this.queryKnowledge(collection, query, 20);
        return response.entries || [];
      } catch (err) {
        console.warn(`Failed to search collection ${collection}:`, err);
        return [];
      }
    });

    const results = await Promise.all(searchPromises);
    let allEntries = results.flat();

    // Sort by score if available (highest first)
    allEntries.sort((a, b) => {
      const scoreA = (a.metadata?.score || 0) as number;
      const scoreB = (b.metadata?.score || 0) as number;
      return scoreB - scoreA;
    });

    // Limit to top 100
    allEntries = allEntries.slice(0, limit);

    // Get unique collections that have data
    const collectionsWithData = [...new Set(allEntries.map(e => e.collection))];

    return {
      entries: allEntries,
      collectionsWithData
    };
  }

  /**
   * Generate a new blog entry from recent task progress
   * Calls the backend endpoint to create a progress article
   */
  async generateBlogEntry(): Promise<{ entry: KnowledgeEntry }> {
    const response = await fetch('/api/v1/blog/generate-entry', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to generate blog entry');
    }

    return response.json();
  }

  /**
   * Start resync to unified collection
   * Rebuilds the knowledge base from MongoDB to Qdrant unified collection
   */
  async startResync(): Promise<void> {
    const response = await fetch('/api/v1/knowledge/resync-to-unified', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to start resync');
    }
  }

  /**
   * Get resync status
   * Returns current progress of the resync operation
   */
  async getResyncStatus(): Promise<{
    inProgress: boolean;
    totalEntries: number;
    processedEntries: number;
    percentage: number;
    estimatedTimeRemaining?: string;
    errorMessage?: string;
    completedTime?: string;
  }> {
    const response = await fetch('/api/v1/knowledge/resync-status', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to get resync status');
    }

    return response.json();
  }

  /**
   * Sync markdown files from knowledge-base folder
   * Imports markdown files and creates/updates knowledge entries
   */
  async syncMarkdownKB(): Promise<SyncReport> {
    const response = await fetch('/api/v1/knowledge/sync-markdown-kb', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to sync markdown KB');
    }

    return response.json();
  }

  /**
   * Export knowledge entries to markdown files
   * Exports all or specified collections to a target directory
   */
  async exportToFiles(outputPath?: string, collections?: string[]): Promise<ExportReport> {
    const response = await fetch('/api/v1/knowledge/export', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        outputPath: outputPath || '.hyper/kb',
        collections: collections || [],
      }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to export knowledge to files');
    }

    return response.json();
  }
}

export const knowledgeService = new KnowledgeService();
