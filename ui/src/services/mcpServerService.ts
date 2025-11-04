/**
 * MCP Server Service
 *
 * Handles API calls for managing external MCP servers in the Hyperion system.
 * MCP servers provide additional tools and capabilities that can be integrated
 * into the Hyperion platform.
 */

// Use relative URL to leverage Vite's proxy configuration
// Proxy forwards /api/v1 to http://localhost:7095 (see vite.config.ts)
const API_BASE_URL = import.meta.env.VITE_API_URL || '';

export interface MCPServer {
  serverName: string;
  serverUrl: string;
  description: string;
  toolCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface ListMCPServersResponse {
  servers: MCPServer[];
  total: number;
}

export interface AddMCPServerRequest {
  serverName: string;
  serverUrl: string;
  description?: string;
}

export interface MCPServerOperationResponse {
  success: boolean;
  message: string;
}

class MCPServerService {
  /**
   * List all registered MCP servers
   */
  async listMCPServers(): Promise<ListMCPServersResponse> {
    const response = await fetch(`${API_BASE_URL}/api/v1/mcp/servers`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to list MCP servers');
    }

    return response.json();
  }

  /**
   * Add a new MCP server to the registry
   */
  async addMCPServer(request: AddMCPServerRequest): Promise<MCPServerOperationResponse> {
    const response = await fetch(`${API_BASE_URL}/api/v1/mcp/servers`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || error.details || 'Failed to add MCP server');
    }

    return response.json();
  }

  /**
   * Remove an MCP server from the registry
   * This also removes all tools associated with the server
   */
  async removeMCPServer(serverName: string): Promise<MCPServerOperationResponse> {
    const response = await fetch(`${API_BASE_URL}/api/v1/mcp/servers/${encodeURIComponent(serverName)}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || error.details || 'Failed to remove MCP server');
    }

    return response.json();
  }

  /**
   * Rediscover tools from an existing MCP server
   * This refreshes the tool list from the server
   */
  async rediscoverMCPServer(serverName: string): Promise<MCPServerOperationResponse> {
    const response = await fetch(
      `${API_BASE_URL}/api/v1/mcp/servers/${encodeURIComponent(serverName)}/rediscover`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      }
    );

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || error.details || 'Failed to rediscover MCP server tools');
    }

    return response.json();
  }
}

export const mcpServerService = new MCPServerService();
