// Knowledge Base Type Definitions
// Enhanced with complete types from ./ui implementation

export interface KnowledgeEntry {
  id: string;
  collection?: string;
  text: string;
  score?: number;
  metadata?: Record<string, any>;
  createdAt?: string;
}

export interface KnowledgeCollection {
  id: string;
  name: string;
  count: number;
  category: string;
  description?: string;
  tags?: string[];
}

// Full Collection interface with metadata (matches backend schema)
export interface Collection {
  id: string;
  name: string;
  category: string;
  description: string;
  tags: string[];
  count: number;
  createdAt: string;
}

// CollectionInfo interface with category field
export interface CollectionInfo {
  id: string;
  name: string;
  category: string;
  count: number;
  description?: string;
  tags?: string[];
}

// Request/Response types for collection creation
export interface CreateCollectionRequest {
  name: string;
  category: string;
  description: string;
  tags: string[];
}

export interface CreateCollectionResponse {
  collection: Collection;
}

export interface SearchRequest {
  collection: string;
  query: string;
  limit?: number;
}

export interface SearchResponse {
  results: KnowledgeEntry[];
  total: number;
}

export interface CreateRequest {
  collection: string;
  text: string;
  metadata?: Record<string, any>;
}

export interface CreateResponse {
  id: string;
  collection: string;
  createdAt: string;
}

// Response types for service methods
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

// Review and Compaction types
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
