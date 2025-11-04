import React, { useState, useEffect } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import { Plug, Plus, RefreshCw, Trash2, X } from 'lucide-react';
import { mcpService } from '@/services/mcpService';
import type { MCPServer } from '@/types/mcp';
import { Button } from '@/components/atoms/Button';
import { Input } from '@/components/atoms/Input';
import { Badge } from '@/components/atoms/Badge';

export function MCPServersPage() {
  const [servers, setServers] = useState<MCPServer[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [newServer, setNewServer] = useState({ name: '', url: '', description: '' });

  useEffect(() => {
    loadServers();
  }, []);

  const loadServers = async () => {
    try {
      setLoading(true);
      const { servers: data } = await mcpService.listServers();
      setServers(data);
    } catch (error) {
      console.error('Failed to load servers:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAddServer = async () => {
    try {
      await mcpService.addServer(newServer);
      setShowAddDialog(false);
      setNewServer({ name: '', url: '', description: '' });
      await loadServers();
    } catch (error) {
      console.error('Failed to add server:', error);
    }
  };

  const handleRemove = async (serverName: string) => {
    if (!confirm(`Remove server "${serverName}"?`)) return;
    try {
      await mcpService.removeServer(serverName);
      await loadServers();
    } catch (error) {
      console.error('Failed to remove server:', error);
    }
  };

  const handleRediscover = async (serverName: string) => {
    try {
      setLoading(true);
      await mcpService.rediscoverServer(serverName);
      await loadServers();
    } catch (error) {
      console.error('Failed to rediscover:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <div className="container mx-auto p-6 space-y-6 max-w-7xl">
        {/* Header - Glassmorphic Container */}
        <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 shadow-lg">
          <div className="flex justify-between items-start">
            <div className="flex items-center gap-3">
              <div className="relative">
                <div className="absolute inset-0 bg-gradient-to-br from-orange-400 to-red-500 rounded-xl blur-lg opacity-30 animate-pulse"></div>
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
            <Dialog.Root open={showAddDialog} onOpenChange={setShowAddDialog}>
              <Dialog.Trigger asChild>
                <Button>
                  <Plus className="h-4 w-4 mr-2" />
                  Add Server
                </Button>
              </Dialog.Trigger>
              <Dialog.Portal>
                <Dialog.Overlay className="fixed inset-0 bg-black/50" />
                <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-background border border-border rounded-lg p-6 w-full max-w-md shadow-lg">
                  <Dialog.Title className="text-xl font-bold mb-4">Add MCP Server</Dialog.Title>
                  <div className="space-y-4">
                    <div>
                      <label className="text-sm font-medium">Server Name</label>
                      <Input
                        value={newServer.name}
                        onChange={(e) => setNewServer({ ...newServer, name: e.target.value })}
                        placeholder="my-mcp-server"
                      />
                    </div>
                    <div>
                      <label className="text-sm font-medium">URL</label>
                      <Input
                        value={newServer.url}
                        onChange={(e) => setNewServer({ ...newServer, url: e.target.value })}
                        placeholder="http://localhost:3000/mcp"
                      />
                    </div>
                    <div>
                      <label className="text-sm font-medium">Description</label>
                      <Input
                        value={newServer.description}
                        onChange={(e) => setNewServer({ ...newServer, description: e.target.value })}
                        placeholder="My custom MCP server"
                      />
                    </div>
                    <div className="flex gap-2 justify-end">
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

        {/* Content */}
        {loading && servers.length === 0 ? (
          <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-12 shadow-lg">
            <div className="text-center text-gray-600 dark:text-gray-400">Loading servers...</div>
          </div>
        ) : servers.length === 0 ? (
          <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-12 shadow-lg">
            <div className="text-center">
              <Plug className="h-12 w-12 mx-auto mb-4 opacity-50 text-gray-400 dark:text-gray-500" />
              <p className="text-gray-600 dark:text-gray-400">No MCP servers registered yet</p>
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {servers.map((server) => (
              <div key={server.name} className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 shadow-lg space-y-3">
                <div className="flex justify-between items-start">
                  <div>
                    <h3 className="font-semibold text-lg text-gray-900 dark:text-gray-100">{server.name}</h3>
                    <p className="text-sm text-gray-600 dark:text-gray-400">{server.description}</p>
                  </div>
                  <Badge variant={server.status === 'active' ? 'success' : 'default'}>
                    {server.status}
                  </Badge>
                </div>
                <div className="text-sm text-gray-700 dark:text-gray-300">
                  <span className="text-gray-600 dark:text-gray-400">URL:</span> {server.url}
                </div>
                {server.toolCount !== undefined && (
                  <div className="text-sm text-gray-700 dark:text-gray-300">
                    <span className="text-gray-600 dark:text-gray-400">Tools:</span> {server.toolCount}
                  </div>
                )}
                <div className="flex gap-2">
                  <Button size="sm" variant="outline" onClick={() => handleRediscover(server.name)}>
                    <RefreshCw className="h-4 w-4 mr-1" />
                    Rediscover
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => handleRemove(server.name)}>
                    <Trash2 className="h-4 w-4 mr-1" />
                    Remove
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
