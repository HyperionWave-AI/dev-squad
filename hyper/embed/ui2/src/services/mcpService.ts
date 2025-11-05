import type { MCPServer, MCPServerDetails, MCPTool, ListMCPServersResponse } from '@/types/mcp';
import { fetchWithAuth } from './restClient';

const API_BASE = '';

export const mcpService = {
  async listServers(): Promise<ListMCPServersResponse> {
    return fetchWithAuth(`${API_BASE}/api/v1/mcp/servers`);
  },

  async getServerDetails(serverName: string): Promise<MCPServerDetails> {
    return fetchWithAuth(`${API_BASE}/api/v1/mcp/servers/${serverName}`);
  },

  async addServer(params: {
    serverName: string;
    serverUrl: string;
    description?: string;
    headers?: Record<string, any>;
  }): Promise<{ message: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/mcp/servers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params),
    });
  },

  async removeServer(serverName: string): Promise<{ message: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/mcp/servers/${serverName}`, {
      method: 'DELETE',
    });
  },

  async rediscoverServer(serverName: string): Promise<{ message: string; toolCount: number; resourceCount: number; promptCount: number }> {
    return fetchWithAuth(`${API_BASE}/api/v1/mcp/servers/${serverName}/rediscover`, {
      method: 'POST',
    });
  },

  async rediscoverAll(): Promise<{ message: string; results: Array<{ serverName: string; success: boolean; error?: string }> }> {
    return fetchWithAuth(`${API_BASE}/api/v1/mcp/servers/rediscover-all`, {
      method: 'POST',
    });
  },
};
