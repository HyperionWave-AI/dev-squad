// Code Index Types
export interface FolderConfig {
  configId?: string;
  folderPath: string;
  filePatterns: string[];
  excludePatterns: string[];
  enabled: boolean;
  fileCount?: number;
}

export interface CodeFile {
  filePath: string;
  fileName: string;
  fileType: string;
  lines: number;
  language: string;
}

export interface SearchResult {
  fileId: string;
  filePath: string;
  relativePath: string;
  language: string;
  chunkNum?: number;
  startLine?: number;
  endLine?: number;
  content: string;
  score: number;
  folderId: string;
  folderPath: string;
  fullFileRetrieved?: boolean;
  // Legacy fields for backward compatibility
  fileName?: string;
  excerpt?: string;
  lines?: number;
}

export interface IndexStatus {
  totalFolders: number;
  totalFiles: number;
  totalSize: number;
  watcherStatus: 'running' | 'stopped';
  folders: Array<{
    configId: string;
    folderPath: string;
    fileCount: number;
    enabled: boolean;
  }>;
}

export interface SearchOptions {
  fileTypes?: string[];
  minScore?: number;
  limit?: number;
  folderPath?: string;  // Optional: filter results to specific folder
  retrieve?: 'chunk' | 'full';  // Optional: content retrieval mode
}
