import type { SearchResult, IndexStatus, SearchRequest } from '@/types/codeIndex';
import { fetchWithAuth } from './restClient';

const API_BASE = '';

export interface AddFolderConfig {
  folderPath: string;
  includePatterns?: string[];
  excludePatterns?: string[];
  chunkSize?: string; // T-shirt sizes: 's', 'm', 'l', 'xl'
}

export interface FileDetails {
  id: string;
  folderPath: string;
  relativePath: string;
  language: string;
  size: number;
  lineCount: number;
  chunkCount: number;
  indexedAt: string;
}

export interface FileChunkDetails {
  chunkNum: number;
  content: string;
  startLine: number;
  endLine: number;
  chunkType: 'ast' | 'line-based';
  nodeType?: string;
  nodeName?: string;
  signature?: string;
}

export const codeIndexService = {
  async getStatus(): Promise<IndexStatus> {
    return fetchWithAuth(`${API_BASE}/api/v1/code-index/status`);
  },

  async search(request: SearchRequest): Promise<{ results: SearchResult[] }> {
    return fetchWithAuth(`${API_BASE}/api/v1/code-index/search`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
  },

  async triggerScan(folderPath?: string): Promise<{ message: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/code-index/scan`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ folderPath }),
    });
  },

  async addFolder(config: AddFolderConfig): Promise<{ success: boolean; configId: string }> {
    const result = await fetchWithAuth<{
      success: boolean;
      message: string;
      folder: { id: string; path: string; description: string };
    }>(`${API_BASE}/api/v1/code-index/add-folder`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        folderPath: config.folderPath,
        includePatterns: config.includePatterns,
        excludePatterns: config.excludePatterns,
        chunkSize: config.chunkSize,
      }),
    });

    return { success: result.success, configId: result.folder.id };
  },

  async removeFolder(configId: string): Promise<{ success: boolean }> {
    const result = await fetchWithAuth<{
      success: boolean;
      message: string;
      filesRemoved: number;
    }>(`${API_BASE}/api/v1/code-index/remove-folder/${encodeURIComponent(configId)}`, {
      method: 'DELETE',
    });

    return { success: result.success };
  },

  async enableWatcher(): Promise<{ success: boolean; message: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/code-index/enable-watcher`, {
      method: 'POST',
    });
  },

  async disableWatcher(): Promise<{ success: boolean; message: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/code-index/disable-watcher`, {
      method: 'POST',
    });
  },

  async reindexAll(): Promise<{
    success: boolean;
    message: string;
    foldersReindexed: number;
    totalFilesIndexed: number;
  }> {
    return fetchWithAuth(`${API_BASE}/api/v1/code-index/reindex-all`, {
      method: 'POST',
    });
  },

  async getFile(fileId: string): Promise<FileDetails> {
    const result = await fetchWithAuth<{
      file: FileDetails;
    }>(`${API_BASE}/api/v1/code-index/file/${encodeURIComponent(fileId)}`);

    return result.file;
  },

  async getFileChunks(fileId: string): Promise<FileChunkDetails[]> {
    const result = await fetchWithAuth<{
      chunks: FileChunkDetails[];
      count: number;
    }>(`${API_BASE}/api/v1/code-index/file/${encodeURIComponent(fileId)}/chunks`);

    return result.chunks || [];
  },
};
