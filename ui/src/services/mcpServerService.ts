/**
 * MCP Server Service
 *
 * Handles API calls for managing external MCP servers in the Hyperion system.
 * MCP servers provide additional tools and capabilities that can be integrated
 * into the Hyperion platform.
 */

export interface MCPServer {
  serverName: string;
  serverUrl: string;
  description: string;
  headers?: Record<string, string>;
  toolCount: number;
  resourceCount?: number;
  promptCount?: number;
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
  headers?: Record<string, string>;
}

export interface MCPServerOperationResponse {
  success: boolean;
  message: string;
}

export interface UpdateMCPServerRequest {
  serverUrl: string;
  description?: string;
  headers?: Record<string, string>;
}

export interface MCPTool {
  name: string;
  description: string;
  inputSchema?: Record<string, unknown>;
}

export interface MCPResource {
  name: string;
  uri: string;
  description: string;
  mimeType?: string;
}

export interface MCPPromptArgument {
  name: string;
  description?: string;
  required?: boolean;
}

export interface MCPPrompt {
  name: string;
  description: string;
  arguments?: MCPPromptArgument[];
}

export interface MCPServerDetails extends MCPServer {
  tools: MCPTool[];
  resources: MCPResource[];
  prompts: MCPPrompt[];
}

export interface GetServerDetailsResponse {
  success: boolean;
  server: MCPServerDetails;
}

export interface RediscoverAllServersResponse {
  success: boolean;
  totalServers: number;
  successCount: number;
  failureCount: number;
  totalTools: number;
  totalResources: number;
  totalPrompts: number;
  errors: Array<{ serverName: string; error: string }>;
  message: string;
}

class MCPServerService {
  /**
   * List all registered MCP servers
   */
  async listMCPServers(): Promise<ListMCPServersResponse> {
    const response = await fetch('/api/v1/mcp/servers', {
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
    const response = await fetch('/api/v1/mcp/servers', {
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
    const response = await fetch(`/api/v1/mcp/servers/${encodeURIComponent(serverName)}`, {
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
      `/api/v1/mcp/servers/${encodeURIComponent(serverName)}/rediscover`,
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

  /**
   * Rediscover tools from all registered MCP servers
   * This refreshes the tool lists from all servers and returns a summary
   */
  async rediscoverAllServers(): Promise<RediscoverAllServersResponse> {
    const response = await fetch('/api/v1/mcp/servers/rediscover-all', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || error.details || 'Failed to rediscover all MCP servers');
    }

    return response.json();
  }

  /**
   * Update an existing MCP server
   * Updates server URL, description, and headers
   */
  async updateMCPServer(
    serverName: string,
    request: UpdateMCPServerRequest
  ): Promise<MCPServerOperationResponse> {
    const response = await fetch(`/api/v1/mcp/servers/${encodeURIComponent(serverName)}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || error.details || 'Failed to update MCP server');
    }

    return response.json();
  }

  /**
   * Get server details including discovered tools
   */
  async getServerDetails(serverName: string): Promise<GetServerDetailsResponse> {
    const response = await fetch(`/api/v1/mcp/servers/${encodeURIComponent(serverName)}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || error.details || 'Failed to get server details');
    }

    return response.json();
  }
}

export const mcpServerService = new MCPServerService();
