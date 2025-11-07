export interface SearchResult {
  filePath: string;
  content: string;
  lineStart: number;
  lineEnd: number;
  score: number;
  language?: string;
}

export interface SearchRequest {
  query: string;
  folderPath?: string;
  limit?: number;
  minScore?: number;
  retrieve?: 'chunk' | 'full';
}

export interface SearchOptions {
  fileTypes?: string[];
  minRelevanceScore?: number;
  minScore?: number;
  maxResults?: number;
  limit?: number;
  folderPath?: string;
  retrieve?: 'chunk' | 'full';
}

export interface FolderInfo {
  configId?: string;
  folderPath: string;
  fileCount?: number;
  enabled?: boolean;
}

export interface IndexStatus {
  indexed: boolean;
  folders: FolderInfo[];
  totalFiles: number;
  fileCount?: number; // Legacy field, use totalFiles instead
  lastScan?: string;
}
