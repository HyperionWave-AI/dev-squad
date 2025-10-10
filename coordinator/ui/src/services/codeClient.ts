// Code Index MCP Client
import { mcpClient } from './mcpClient';
import type {
  SearchResult,
  IndexStatus,
  SearchOptions,
} from '../types/codeIndex';

class CodeIndexClient {
  /**
   * Add a folder to the code index
   */
  async addFolder(params: {
    folderPath: string;
    filePatterns?: string[];
    excludePatterns?: string[];
  }): Promise<{ success: boolean; configId: string }> {
    try {
      const result = await (mcpClient as any).callTool('code_index_add_folder', {
        folderPath: params.folderPath,
        filePatterns: params.filePatterns || ['*.go', '*.ts', '*.tsx', '*.js', '*.py'],
        excludePatterns: params.excludePatterns || ['node_modules', 'dist', 'build', '.git'],
      });

      // Extract config ID from response
      let configId = 'unknown';
      if (result.content && result.content[0] && result.content[0].text) {
        const text = result.content[0].text;
        const match = text.match(/Config ID:\s*([a-f0-9-]+)/i);
        if (match) {
          configId = match[1];
        }
      }

      return {
        success: true,
        configId,
      };
    } catch (error) {
      console.error('Failed to add folder:', error);
      throw error;
    }
  }

  /**
   * Remove a folder from the code index
   */
  async removeFolder(configId: string): Promise<{ success: boolean }> {
    try {
      await (mcpClient as any).callTool('code_index_remove_folder', {
        configId,
      });

      return { success: true };
    } catch (error) {
      console.error('Failed to remove folder:', error);
      throw error;
    }
  }

  /**
   * Trigger a scan of indexed folders
   */
  async scan(configId?: string): Promise<{ success: boolean }> {
    try {
      const args: any = {};
      if (configId) {
        args.configId = configId;
      }

      await (mcpClient as any).callTool('code_index_scan', args);

      return { success: true };
    } catch (error) {
      console.error('Failed to scan folders:', error);
      throw error;
    }
  }

  /**
   * Search code semantically
   */
  async search(
    query: string,
    options?: SearchOptions
  ): Promise<SearchResult[]> {
    try {
      const args: any = {
        query,
      };

      if (options?.fileTypes && options.fileTypes.length > 0) {
        args.fileTypes = options.fileTypes;
      }

      if (options?.minScore !== undefined) {
        args.minScore = options.minScore;
      }

      if (options?.limit !== undefined) {
        args.limit = options.limit;
      }

      const result = await (mcpClient as any).callTool('code_search_semantic', args);

      // Parse results from response
      if (result.content && result.content[0] && result.content[0].text) {
        const text = result.content[0].text;
        try {
          // Try to parse as JSON array
          const results = JSON.parse(text) as SearchResult[];
          return results;
        } catch (parseError) {
          // If not JSON, try to extract from formatted text
          console.warn('Failed to parse search results as JSON:', parseError);
          return [];
        }
      }

      return [];
    } catch (error) {
      console.error('Failed to search code:', error);
      throw error;
    }
  }

  /**
   * Get index status
   */
  async getStatus(): Promise<IndexStatus> {
    try {
      const result = await (mcpClient as any).callTool('code_index_status', {});

      // Parse status from response
      if (result.content && result.content[0] && result.content[0].text) {
        const text = result.content[0].text;
        try {
          const status = JSON.parse(text) as IndexStatus;
          return status;
        } catch (parseError) {
          console.warn('Failed to parse status as JSON:', parseError);
          // Return default status
          return {
            totalFolders: 0,
            totalFiles: 0,
            totalSize: 0,
            watcherStatus: 'stopped',
            folders: [],
          };
        }
      }

      return {
        totalFolders: 0,
        totalFiles: 0,
        totalSize: 0,
        watcherStatus: 'stopped',
        folders: [],
      };
    } catch (error) {
      console.error('Failed to get status:', error);
      throw error;
    }
  }
}

export const codeClient = new CodeIndexClient();
