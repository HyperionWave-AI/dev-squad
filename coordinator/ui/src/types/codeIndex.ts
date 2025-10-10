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
  filePath: string;
  fileName: string;
  score: number;
  excerpt: string;
  lines: number;
  language: string;
}

export interface IndexStatus {
  totalFolders: number;
  totalFiles: number;
  totalSize: number;
  watcherStatus: 'running' | 'stopped';
  folders: Array<{
    folderPath: string;
    fileCount: number;
    enabled: boolean;
  }>;
}

export interface SearchOptions {
  fileTypes?: string[];
  minScore?: number;
  limit?: number;
}
