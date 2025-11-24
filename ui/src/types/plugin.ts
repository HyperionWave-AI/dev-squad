/**
 * Plugin System Types
 *
 * Defines the plugin interface and related types for the chat system.
 * Plugins can hook into various lifecycle events and message processing pipelines.
 */

import type { ChatMessage, ToolCall, ToolResult } from '@/services/chatService';

/**
 * Plugin lifecycle and event hooks
 */
export interface ChatPlugin {
  // Plugin metadata
  name: string;
  version: string;
  description?: string;
  priority: number; // 0-100, higher = runs first

  // Lifecycle hooks
  onInit?(): Promise<void>;
  onDestroy?(): Promise<void>;

  // Message processing hooks
  onMessageReceived?(message: ChatMessage): Promise<ChatMessage>;
  onMessageSent?(content: string): Promise<string>;

  // Tool execution hooks
  onToolCall?(toolCall: ToolCall): Promise<ToolCall>;
  onToolResult?(result: ToolResult): Promise<ToolResult>;

  // Session lifecycle hooks
  onSessionCreated?(sessionId: string): Promise<void>;
  onSessionDeleted?(sessionId: string): Promise<void>;
  onSessionChanged?(sessionId: string): Promise<void>;

  // Error handling hooks
  onError?(error: Error): Promise<void>;

  // Streaming hooks
  onStreamStart?(sessionId: string): Promise<void>;
  onStreamEnd?(sessionId: string): Promise<void>;
  onStreamChunk?(sessionId: string, chunk: string): Promise<string>;
}

/**
 * Plugin registry interface
 */
export interface IPluginRegistry {
  register(plugin: ChatPlugin): void;
  unregister(name: string): void;
  getPlugin(name: string): ChatPlugin | undefined;
  getPlugins(hook: keyof ChatPlugin): ChatPlugin[];
  executeHook<T>(hook: keyof ChatPlugin, data: T): Promise<T>;
  getAllPlugins(): ChatPlugin[];
  isPluginEnabled(name: string): boolean;
  setPluginEnabled(name: string, enabled: boolean): void;
}

/**
 * Plugin configuration
 */
export interface PluginConfig {
  enabled: boolean;
  priority?: number;
  options?: Record<string, any>;
}

/**
 * Plugin metadata for discovery
 */
export interface PluginMetadata {
  name: string;
  version: string;
  description: string;
  author?: string;
  hooks: (keyof ChatPlugin)[];
  config?: PluginConfig;
}

/**
 * Plugin execution context
 */
export interface PluginExecutionContext {
  sessionId?: string;
  userId?: string;
  timestamp: Date;
  metadata?: Record<string, any>;
}

/**
 * Plugin error
 */
export class PluginError extends Error {
  pluginName: string;
  hook: string;
  originalError?: Error;

  constructor(
    pluginName: string,
    hook: string,
    message: string,
    originalError?: Error
  ) {
    super(`Plugin "${pluginName}" failed on hook "${hook}": ${message}`);
    this.name = 'PluginError';
    this.pluginName = pluginName;
    this.hook = hook;
    this.originalError = originalError;
  }
}
