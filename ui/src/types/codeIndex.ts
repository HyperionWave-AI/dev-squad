export interface SearchResult {
  fileId: string;
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
  retrieve?: 'chunk-s' | 'chunk-m' | 'chunk-l' | 'chunk-xl' | 'full';
  fileTypes?: string[];
}

export interface SearchOptions {
  fileTypes?: string[];
  minRelevanceScore?: number;
  minScore?: number;
  maxResults?: number;
  limit?: number;
  folderPath?: string;
  retrieve?: 'chunk-s' | 'chunk-m' | 'chunk-l' | 'chunk-xl' | 'full';
}

export interface FolderInfo {
  configId?: string;
  folderPath: string;
  fileCount?: number;
  enabled?: boolean;
}

export interface IndexStatus {
  indexed?: boolean;
  folders: FolderInfo[];
  totalFiles: number;
  totalFolders?: number;
  totalSize?: number;
  fileCount?: number; // Legacy field, use totalFiles instead
  lastScan?: string;
  lastScanTime?: string;
  watcherStatus?: 'running' | 'stopped';
}
