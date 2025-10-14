// REST Code Index Client - NO MCP usage
// This replaces codeClient.ts to use REST API instead of direct MCP calls

import type {
  SearchResult,
  IndexStatus,
  SearchOptions,
} from '../types/codeIndex';

const BASE_URL = '/api/v1/code-index';

class RestCodeClient {
  private async fetchJSON<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${BASE_URL}${endpoint}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: response.statusText }));
      throw new Error(error.error || `HTTP ${response.status}: ${response.statusText}`);
    }

    return response.json();
  }

  /**
   * Add a folder to the code index
   */
  async addFolder(params: {
    folderPath: string;
    description?: string;
  }): Promise<{ success: boolean; configId: string }> {
    const result = await this.fetchJSON<{
      success: boolean;
      message: string;
      folder: { id: string; path: string; description: string };
    }>(
      '/add-folder',
      {
        method: 'POST',
        body: JSON.stringify({
          folderPath: params.folderPath,
          description: params.description,
        }),
      }
    );

    return { success: result.success, configId: result.folder.id };
  }

  /**
   * Remove a folder from the code index
   */
  async removeFolder(configId: string): Promise<{ success: boolean }> {
    const result = await this.fetchJSON<{
      success: boolean;
      message: string;
      filesRemoved: number;
    }>(
      `/remove-folder/${encodeURIComponent(configId)}`,
      {
        method: 'DELETE',
      }
    );

    return { success: result.success };
  }

  /**
   * Trigger a scan of indexed folders
   */
  async scan(folderPath?: string): Promise<{ success: boolean }> {
    if (!folderPath) {
      throw new Error('folderPath is required for scan operation');
    }

    const result = await this.fetchJSON<{
      success: boolean;
      filesIndexed: number;
      filesUpdated: number;
      filesSkipped: number;
      totalFiles: number;
    }>(
      '/scan',
      {
        method: 'POST',
        body: JSON.stringify({ folderPath }),
      }
    );

    return { success: result.success };
  }

  /**
   * Search code semantically
   */
  async search(
    query: string,
    options?: SearchOptions
  ): Promise<SearchResult[]> {
    const result = await this.fetchJSON<{
      success: boolean;
      query: string;
      retrieveMode: string;
      results: SearchResult[];
      count: number;
    }>(
      '/search',
      {
        method: 'POST',
        body: JSON.stringify({
          query,
          fileTypes: options?.fileTypes,
          minScore: options?.minScore,
          folderPath: options?.folderPath,
          limit: options?.limit,
          retrieve: options?.retrieve,
        }),
      }
    );

    return result.results || [];
  }

  /**
   * Get index status
   */
  async getStatus(): Promise<IndexStatus> {
    const status = await this.fetchJSON<IndexStatus>('/status');

    return status || {
      totalFolders: 0,
      totalFiles: 0,
      totalSize: 0,
      watcherStatus: 'stopped',
      folders: [],
    };
  }
}

export const restCodeClient = new RestCodeClient();
