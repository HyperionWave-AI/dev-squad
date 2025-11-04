import type { MCPServer, MCPTool } from '@/types/mcp';
import { fetchWithAuth } from './restClient';

const API_BASE = '';

export const mcpService = {
  async listServers(): Promise<{ servers: MCPServer[] }> {
    return fetchWithAuth(`${API_BASE}/api/v1/mcp/servers`);
  },

  async addServer(server: Omit<MCPServer, 'toolCount' | 'status' | 'lastDiscovery'>): Promise<{ message: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/mcp/servers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(server),
    });
  },

  async removeServer(serverName: string): Promise<{ message: string }> {
    return fetchWithAuth(`${API_BASE}/api/v1/mcp/servers/${serverName}`, {
      method: 'DELETE',
    });
  },

  async rediscoverServer(serverName: string): Promise<{ tools: MCPTool[] }> {
    return fetchWithAuth(`${API_BASE}/api/v1/mcp/servers/${serverName}/rediscover`, {
      method: 'POST',
    });
  },
};
