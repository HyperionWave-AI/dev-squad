import React, { useState, useEffect } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import * as Collapsible from '@radix-ui/react-collapsible';
import {
  Plug, Plus, RefreshCw, Trash2, X, ChevronDown, ChevronUp,
  Search, Filter, Server, Layers, MessageSquare, Code
} from 'lucide-react';
import { mcpService } from '@/services/mcpService';
import type { MCPServer, MCPServerDetails } from '@/types/mcp';
import { Button } from '@/components/atoms/Button';
import { Input } from '@/components/atoms/Input';
import { Badge } from '@/components/atoms/Badge';

export function MCPServersPage() {
  const [servers, setServers] = useState<MCPServer[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [expandedServer, setExpandedServer] = useState<string | null>(null);
  const [serverDetails, setServerDetails] = useState<Record<string, MCPServerDetails>>({});
  const [newServer, setNewServer] = useState({
    serverName: '',
    serverUrl: '',
    description: '',
    headers: {} as Record<string, string>
  });
  const [notification, setNotification] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  useEffect(() => {
    loadServers();
  }, []);

  const loadServers = async () => {
    try {
      setLoading(true);
      const { servers: data } = await mcpService.listServers();
      setServers(data);
    } catch (error) {
      showNotification('Failed to load servers', 'error');
      console.error('Failed to load servers:', error);
    } finally {
      setLoading(false);
    }
  };

  const showNotification = (message: string, type: 'success' | 'error') => {
    setNotification({ message, type });
    setTimeout(() => setNotification(null), 5000);
  };

  const handleAddServer = async () => {
    try {
      await mcpService.addServer(newServer);
      setShowAddDialog(false);
      setNewServer({ serverName: '', serverUrl: '', description: '', headers: {} });
      await loadServers();
      showNotification('MCP server added successfully', 'success');
    } catch (error) {
      showNotification('Failed to add server', 'error');
      console.error('Failed to add server:', error);
    }
  };

  const handleRemove = async (serverName: string) => {
    if (!confirm(`Remove server "${serverName}"? This will also remove all associated tools, resources, and prompts.`)) return;
    try {
      await mcpService.removeServer(serverName);
      await loadServers();
      showNotification(`Server "${serverName}" removed successfully`, 'success');
    } catch (error) {
      showNotification('Failed to remove server', 'error');
      console.error('Failed to remove server:', error);
    }
  };

  const handleRediscover = async (serverName: string) => {
    try {
      const result = await mcpService.rediscoverServer(serverName);
      await loadServers();
      showNotification(`Rediscovered ${result.toolCount} tools, ${result.resourceCount} resources, ${result.promptCount} prompts`, 'success');
    } catch (error) {
      showNotification('Failed to rediscover', 'error');
      console.error('Failed to rediscover:', error);
    }
  };

  const handleRediscoverAll = async () => {
    try {
      setLoading(true);
      const result = await mcpService.rediscoverAll();
      await loadServers();
      const successCount = result.results.filter(r => r.success).length;
      showNotification(`Rediscovered ${successCount}/${result.results.length} servers`, 'success');
    } catch (error) {
      showNotification('Failed to rediscover all', 'error');
      console.error('Failed to rediscover all:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleExpand = async (serverName: string) => {
    if (expandedServer === serverName) {
      setExpandedServer(null);
    } else {
      setExpandedServer(serverName);
      if (!serverDetails[serverName]) {
        try {
          const details = await mcpService.getServerDetails(serverName);
          setServerDetails(prev => ({ ...prev, [serverName]: details }));
        } catch (error) {
          showNotification('Failed to load server details', 'error');
          console.error('Failed to load server details:', error);
        }
      }
    }
  };

  const formatDate = (dateString: string): string => {
    try {
      return new Date(dateString).toLocaleString();
    } catch {
      return dateString;
    }
  };

  const filteredServers = servers.filter(server =>
    server.serverName.toLowerCase().includes(searchTerm.toLowerCase()) ||
    server.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
    server.serverUrl.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <div className="container mx-auto p-6 space-y-6 max-w-7xl">
        {/* Header - Glassmorphic Container */}
        <div className="backdrop-blur-xl bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-2xl p-6 shadow-xl sticky top-0 z-10">
          <div className="flex justify-between items-start">
            <div className="flex items-center gap-4">
              <div className="relative">
                <div className="absolute inset-0 bg-gradient-to-br from-orange-400 to-red-500 rounded-xl blur-xl opacity-40 animate-pulse"></div>
                <div className="relative p-3 bg-gradient-to-br from-orange-500 to-red-600 rounded-xl shadow-xl">
                  <Plug className="h-8 w-8 text-white" />
                </div>
              </div>
              <div>
                <h1 className="text-3xl font-bold bg-gradient-to-r from-orange-600 via-red-600 to-orange-600 bg-clip-text text-transparent">
                  MCP Servers
                </h1>
                <p className="text-gray-600 dark:text-gray-400 mt-1">
                  Manage external Model Context Protocol servers and tools
                </p>
              </div>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={handleRediscoverAll} disabled={loading}>
                <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
                Rediscover All
              </Button>
              <Dialog.Root open={showAddDialog} onOpenChange={setShowAddDialog}>
                <Dialog.Trigger asChild>
                  <Button>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Server
                  </Button>
                </Dialog.Trigger>
                <Dialog.Portal>
                  <Dialog.Overlay className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50" />
                  <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl p-6 w-full max-w-md shadow-2xl z-50">
                    <Dialog.Title className="text-xl font-bold mb-4 text-gray-900 dark:text-white">Add MCP Server</Dialog.Title>
                    <div className="space-y-4">
                      <div>
                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Server Name</label>
                        <Input
                          value={newServer.serverName}
                          onChange={(e) => setNewServer({ ...newServer, serverName: e.target.value })}
                          placeholder="my-mcp-server"
                        />
                      </div>
                      <div>
                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">URL</label>
                        <Input
                          value={newServer.serverUrl}
                          onChange={(e) => setNewServer({ ...newServer, serverUrl: e.target.value })}
                          placeholder="http://localhost:3000/mcp"
                        />
                      </div>
                      <div>
                        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">Description</label>
                        <Input
                          value={newServer.description}
                          onChange={(e) => setNewServer({ ...newServer, description: e.target.value })}
                          placeholder="My custom MCP server"
                        />
                      </div>
                      <div className="flex gap-2 justify-end pt-2">
                        <Dialog.Close asChild>
                          <Button variant="outline">Cancel</Button>
                        </Dialog.Close>
                        <Button onClick={handleAddServer}>Add Server</Button>
                      </div>
                    </div>
                  </Dialog.Content>
                </Dialog.Portal>
              </Dialog.Root>
            </div>
          </div>

          {/* Search and Stats */}
          <div className="flex items-center gap-4 mt-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
              <Input
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Search servers by name, URL, or description..."
                className="pl-10"
              />
            </div>
            <Badge variant="default" className="px-4 py-2 bg-gradient-to-r from-blue-500 to-blue-600">
              <Server className="h-4 w-4 mr-2" />
              {filteredServers.length} server{filteredServers.length !== 1 ? 's' : ''}
            </Badge>
          </div>
        </div>

        {/* Notification Toast */}
        {notification && (
          <div className={`fixed top-4 right-4 z-50 backdrop-blur-xl rounded-xl p-4 shadow-2xl border ${
            notification.type === 'success'
              ? 'bg-green-500/90 border-green-400 text-white'
              : 'bg-red-500/90 border-red-400 text-white'
          }`}>
            {notification.message}
          </div>
        )}

        {/* Content */}
        {loading && servers.length === 0 ? (
          <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-xl p-12 shadow-lg">
            <div className="text-center text-gray-600 dark:text-gray-400 flex items-center justify-center gap-3">
              <RefreshCw className="h-5 w-5 animate-spin" />
              Loading servers...
            </div>
          </div>
        ) : filteredServers.length === 0 ? (
          <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-xl p-12 shadow-lg">
            <div className="text-center">
              <Plug className="h-16 w-16 mx-auto mb-4 opacity-30 text-gray-400 dark:text-gray-500" />
              <p className="text-gray-600 dark:text-gray-400 text-lg font-medium">
                {searchTerm ? 'No servers match your search' : 'No MCP servers registered yet'}
              </p>
              <p className="text-gray-500 dark:text-gray-500 text-sm mt-2">
                {searchTerm ? 'Try a different search term' : 'Click "Add Server" to get started'}
              </p>
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            {/* Table Header */}
            <div className="backdrop-blur-md bg-white/50 dark:bg-gray-800/50 border border-white/30 dark:border-gray-700/30 rounded-xl px-6 py-3 shadow-lg">
              <div className="grid grid-cols-12 gap-4 text-sm font-semibold text-gray-700 dark:text-gray-300">
                <div className="col-span-2">Server Name</div>
                <div className="col-span-3">URL</div>
                <div className="col-span-2">Description</div>
                <div className="col-span-1 text-center">Tools</div>
                <div className="col-span-1 text-center">Resources</div>
                <div className="col-span-1 text-center">Prompts</div>
                <div className="col-span-2 text-right">Actions</div>
              </div>
            </div>

            {/* Server Rows */}
            {filteredServers.map((server) => (
              <Collapsible.Root
                key={server.serverName}
                open={expandedServer === server.serverName}
                onOpenChange={() => handleToggleExpand(server.serverName)}
              >
                <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-xl shadow-lg hover:shadow-xl transition-all duration-200">
                  {/* Server Row */}
                  <div className="px-6 py-4">
                    <div className="grid grid-cols-12 gap-4 items-center">
                      <div className="col-span-2">
                        <div className="font-semibold text-gray-900 dark:text-white">{server.serverName}</div>
                        <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                          {formatDate(server.updatedAt)}
                        </div>
                      </div>
                      <div className="col-span-3">
                        <code className="text-xs text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-900/50 px-2 py-1 rounded">
                          {server.serverUrl}
                        </code>
                      </div>
                      <div className="col-span-2 text-sm text-gray-600 dark:text-gray-400 truncate">
                        {server.description || '-'}
                      </div>
                      <div className="col-span-1 text-center">
                        <Badge variant={server.toolCount > 0 ? 'success' : 'default'} className="text-xs">
                          <Code className="h-3 w-3 mr-1" />
                          {server.toolCount}
                        </Badge>
                      </div>
                      <div className="col-span-1 text-center">
                        <Badge variant={server.resourceCount > 0 ? 'success' : 'default'} className="text-xs">
                          <Layers className="h-3 w-3 mr-1" />
                          {server.resourceCount}
                        </Badge>
                      </div>
                      <div className="col-span-1 text-center">
                        <Badge variant={server.promptCount > 0 ? 'success' : 'default'} className="text-xs">
                          <MessageSquare className="h-3 w-3 mr-1" />
                          {server.promptCount}
                        </Badge>
                      </div>
                      <div className="col-span-2 flex gap-2 justify-end">
                        <Collapsible.Trigger asChild>
                          <Button size="sm" variant="outline">
                            {expandedServer === server.serverName ? (
                              <><ChevronUp className="h-4 w-4 mr-1" /> Hide</>
                            ) : (
                              <><ChevronDown className="h-4 w-4 mr-1" /> Details</>
                            )}
                          </Button>
                        </Collapsible.Trigger>
                        <Button size="sm" variant="outline" onClick={() => handleRediscover(server.serverName)}>
                          <RefreshCw className="h-4 w-4" />
                        </Button>
                        <Button size="sm" variant="outline" onClick={() => handleRemove(server.serverName)}>
                          <Trash2 className="h-4 w-4 text-red-500" />
                        </Button>
                      </div>
                    </div>
                  </div>

                  {/* Expanded Details */}
                  <Collapsible.Content>
                    <div className="border-t border-gray-200 dark:border-gray-700 px-6 py-4 bg-gray-50/50 dark:bg-gray-900/50 rounded-b-xl">
                      {serverDetails[server.serverName] ? (
                        <div className="space-y-6">
                          {/* Tools */}
                          {serverDetails[server.serverName].tools && serverDetails[server.serverName].tools.length > 0 && (
                            <div>
                              <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                                <Code className="h-4 w-4" />
                                Tools ({serverDetails[server.serverName].tools.length})
                              </h3>
                              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                                {serverDetails[server.serverName].tools.map((tool) => (
                                  <div key={tool.name} className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-3">
                                    <div className="font-medium text-sm text-gray-900 dark:text-white">{tool.name}</div>
                                    <div className="text-xs text-gray-600 dark:text-gray-400 mt-1">{tool.description}</div>
                                    {tool.inputSchema && (
                                      <details className="mt-2">
                                        <summary className="text-xs text-blue-600 dark:text-blue-400 cursor-pointer hover:underline">
                                          View Schema
                                        </summary>
                                        <pre className="mt-2 text-xs bg-gray-100 dark:bg-gray-900 p-2 rounded overflow-x-auto">
                                          {JSON.stringify(tool.inputSchema, null, 2)}
                                        </pre>
                                      </details>
                                    )}
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}

                          {/* Resources */}
                          {serverDetails[server.serverName].resources && serverDetails[server.serverName].resources.length > 0 && (
                            <div>
                              <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                                <Layers className="h-4 w-4" />
                                Resources ({serverDetails[server.serverName].resources.length})
                              </h3>
                              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                                {serverDetails[server.serverName].resources.map((resource) => (
                                  <div key={resource.uri} className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-3">
                                    <div className="font-medium text-sm text-gray-900 dark:text-white">{resource.name}</div>
                                    <div className="text-xs text-gray-600 dark:text-gray-400 mt-1">{resource.description}</div>
                                    <div className="text-xs text-gray-500 dark:text-gray-500 mt-1">
                                      <span className="font-mono">{resource.uri}</span>
                                      {resource.mimeType && <span className="ml-2">({resource.mimeType})</span>}
                                    </div>
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}

                          {/* Prompts */}
                          {serverDetails[server.serverName].prompts && serverDetails[server.serverName].prompts.length > 0 && (
                            <div>
                              <h3 className="text-sm font-semibold text-gray-900 dark:text-white mb-3 flex items-center gap-2">
                                <MessageSquare className="h-4 w-4" />
                                Prompts ({serverDetails[server.serverName].prompts.length})
                              </h3>
                              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                                {serverDetails[server.serverName].prompts.map((prompt) => (
                                  <div key={prompt.name} className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-3">
                                    <div className="font-medium text-sm text-gray-900 dark:text-white">{prompt.name}</div>
                                    <div className="text-xs text-gray-600 dark:text-gray-400 mt-1">{prompt.description}</div>
                                    {prompt.arguments && prompt.arguments.length > 0 && (
                                      <details className="mt-2">
                                        <summary className="text-xs text-blue-600 dark:text-blue-400 cursor-pointer hover:underline">
                                          View Arguments
                                        </summary>
                                        <pre className="mt-2 text-xs bg-gray-100 dark:bg-gray-900 p-2 rounded overflow-x-auto">
                                          {JSON.stringify(prompt.arguments, null, 2)}
                                        </pre>
                                      </details>
                                    )}
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}

                          {/* Empty State */}
                          {(!serverDetails[server.serverName].tools || serverDetails[server.serverName].tools.length === 0) &&
                           (!serverDetails[server.serverName].resources || serverDetails[server.serverName].resources.length === 0) &&
                           (!serverDetails[server.serverName].prompts || serverDetails[server.serverName].prompts.length === 0) && (
                            <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                              No tools, resources, or prompts discovered for this server
                            </div>
                          )}
                        </div>
                      ) : (
                        <div className="text-center py-8 text-gray-500 dark:text-gray-400 flex items-center justify-center gap-3">
                          <RefreshCw className="h-5 w-5 animate-spin" />
                          Loading details...
                        </div>
                      )}
                    </div>
                  </Collapsible.Content>
                </div>
              </Collapsible.Root>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
