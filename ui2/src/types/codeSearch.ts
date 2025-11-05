// Type definitions for code search functionality

export interface CodeResult {
  id: string;
  filePath: string;
  fileName: string;
  language: string;
  content: string;
  lineStart: number;
  lineEnd: number;
  relevanceScore: number;
}

export interface IndexedFolder {
  id: string;
  path: string;
  fileCount: number;
  enabled: boolean;
}

export interface IndexStatus {
  totalFiles: number;
  totalFolders: number;
  lastScanTime?: string;
  isRunning: boolean;
}

export interface FolderConfig {
  path: string;
  filePatterns: string[];
  chunkSize: number;
  excludePatterns: string[];
}

export interface SearchOptions {
  fileTypes?: string[];
  minRelevanceScore?: number;
  maxResults?: number;
}
