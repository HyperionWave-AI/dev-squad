export interface MCPServer {
  serverName: string;
  serverUrl: string;
  description: string;
  headers?: Record<string, any>;
  toolCount: number;
  resourceCount: number;
  promptCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface MCPTool {
  name: string;
  description: string;
  inputSchema?: Record<string, any>;
  createdAt: string;
  updatedAt: string;
}

export interface MCPResource {
  uri: string;
  name: string;
  description: string;
  mimeType?: string;
  createdAt: string;
  updatedAt: string;
}

export interface MCPPrompt {
  name: string;
  description: string;
  arguments?: Array<Record<string, any>>;
  createdAt: string;
  updatedAt: string;
}

export interface MCPServerDetails extends MCPServer {
  tools: MCPTool[];
  resources: MCPResource[];
  prompts: MCPPrompt[];
}

export interface ListMCPServersResponse {
  servers: MCPServer[];
}
