export interface MCPServer {
  name: string;
  url: string;
  description: string;
  headers?: Record<string, string>;
  toolCount?: number;
  status: 'active' | 'error' | 'unknown';
  lastDiscovery?: string;
}

export interface MCPTool {
  name: string;
  description: string;
  parameters?: Record<string, any>;
}
