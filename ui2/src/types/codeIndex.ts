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
  retrieve?: 'chunk' | 'chunk-s' | 'chunk-m' | 'chunk-l' | 'chunk-xl' | 'full';
}

export interface IndexStatus {
  indexed: boolean;
  folders: string[];
  fileCount: number;
  lastScan?: string;
}
