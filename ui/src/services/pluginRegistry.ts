/**
 * Plugin Registry Implementation
 *
 * Manages plugin registration, lifecycle, and hook execution with priority-based ordering.
 * Provides a centralized way to extend chat system functionality.
 */

import type { ChatPlugin, IPluginRegistry, PluginExecutionContext } from '@/types/plugin';
import { PluginError } from '@/types/plugin';

export class PluginRegistry implements IPluginRegistry {
  private plugins: Map<string, ChatPlugin> = new Map();
  private hooks: Map<string, ChatPlugin[]> = new Map();
  private enabledPlugins: Set<string> = new Set();
  private context: PluginExecutionContext = {
    timestamp: new Date(),
  };

  constructor() {
    console.log('[PluginRegistry] Initialized');
  }

  /**
   * Register a plugin
   */
  register(plugin: ChatPlugin): void {
    if (this.plugins.has(plugin.name)) {
      console.warn(`[PluginRegistry] Plugin "${plugin.name}" already registered, replacing`);
    }

    this.plugins.set(plugin.name, plugin);
    this.enabledPlugins.add(plugin.name);

    // Register hooks
    this.registerHooks(plugin);

    console.log(
      `[PluginRegistry] ✅ Registered plugin: "${plugin.name}" v${plugin.version} (priority: ${plugin.priority})`
    );
  }

  /**
   * Unregister a plugin
   */
  async unregister(name: string): Promise<void> {
    const plugin = this.plugins.get(name);
    if (!plugin) {
      console.warn(`[PluginRegistry] Plugin "${name}" not found`);
      return;
    }

    // Call destroy hook
    if (plugin.onDestroy) {
      try {
        await plugin.onDestroy();
      } catch (error) {
        console.error(`[PluginRegistry] Error destroying plugin "${name}":`, error);
      }
    }

    // Remove from registry
    this.plugins.delete(name);
    this.enabledPlugins.delete(name);

    // Remove from all hooks
    for (const hookPlugins of this.hooks.values()) {
      const index = hookPlugins.findIndex((p) => p.name === name);
      if (index !== -1) {
        hookPlugins.splice(index, 1);
      }
    }

    console.log(`[PluginRegistry] ✅ Unregistered plugin: "${name}"`);
  }

  /**
   * Get a specific plugin
   */
  getPlugin(name: string): ChatPlugin | undefined {
    return this.plugins.get(name);
  }

  /**
   * Get all plugins for a specific hook
   */
  getPlugins(hook: keyof ChatPlugin): ChatPlugin[] {
    return this.hooks.get(hook as string) || [];
  }

  /**
   * Get all registered plugins
   */
  getAllPlugins(): ChatPlugin[] {
    return Array.from(this.plugins.values());
  }

  /**
   * Check if a plugin is enabled
   */
  isPluginEnabled(name: string): boolean {
    return this.enabledPlugins.has(name);
  }

  /**
   * Enable/disable a plugin
   */
  setPluginEnabled(name: string, enabled: boolean): void {
    if (!this.plugins.has(name)) {
      console.warn(`[PluginRegistry] Plugin "${name}" not found`);
      return;
    }

    if (enabled) {
      this.enabledPlugins.add(name);
      console.log(`[PluginRegistry] ✅ Enabled plugin: "${name}"`);
    } else {
      this.enabledPlugins.delete(name);
      console.log(`[PluginRegistry] ⏸️  Disabled plugin: "${name}"`);
    }
  }

  /**
   * Execute a hook across all registered plugins
   * Plugins are executed in priority order (highest first)
   * Data is passed through each plugin sequentially
   */
  async executeHook<T>(hook: keyof ChatPlugin, data: T): Promise<T> {
    const hookName = hook as string;
    const plugins = this.hooks.get(hookName) || [];

    if (plugins.length === 0) {
      return data;
    }

    let result = data;

    for (const plugin of plugins) {
      // Skip disabled plugins
      if (!this.enabledPlugins.has(plugin.name)) {
        continue;
      }

      const hookFn = plugin[hook] as Function | undefined;
      if (!hookFn) {
        continue;
      }

      try {
        // Update context timestamp
        this.context.timestamp = new Date();

        // Execute hook with plugin context
        const pluginResult = await hookFn.call(plugin, result, this.context);

        // Validate result type matches input type
        if (pluginResult !== undefined && pluginResult !== null) {
          result = pluginResult;
        }

        console.debug(
          `[PluginRegistry] ✅ Hook "${hookName}" executed by plugin "${plugin.name}"`
        );
      } catch (error) {
        const pluginError = new PluginError(
          plugin.name,
          hookName,
          error instanceof Error ? error.message : String(error),
          error instanceof Error ? error : undefined
        );

        console.error(`[PluginRegistry] ❌ ${pluginError.message}`);

        // Continue with other plugins instead of failing completely
        // This ensures one plugin error doesn't break the entire pipeline
      }
    }

    return result;
  }

  /**
   * Initialize all plugins
   */
  async initializeAll(): Promise<void> {
    console.log('[PluginRegistry] Initializing all plugins...');

    for (const plugin of this.plugins.values()) {
      if (!this.enabledPlugins.has(plugin.name)) {
        continue;
      }

      if (plugin.onInit) {
        try {
          await plugin.onInit();
          console.log(`[PluginRegistry] ✅ Initialized plugin: "${plugin.name}"`);
        } catch (error) {
          console.error(
            `[PluginRegistry] ❌ Failed to initialize plugin "${plugin.name}":`,
            error
          );
          // Disable plugin if initialization fails
          this.enabledPlugins.delete(plugin.name);
        }
      }
    }
  }

  /**
   * Destroy all plugins
   */
  async destroyAll(): Promise<void> {
    console.log('[PluginRegistry] Destroying all plugins...');

    for (const plugin of this.plugins.values()) {
      if (plugin.onDestroy) {
        try {
          await plugin.onDestroy();
          console.log(`[PluginRegistry] ✅ Destroyed plugin: "${plugin.name}"`);
        } catch (error) {
          console.error(
            `[PluginRegistry] ❌ Failed to destroy plugin "${plugin.name}":`,
            error
          );
        }
      }
    }
  }

  /**
   * Set execution context (e.g., session ID, user ID)
   */
  setContext(context: Partial<PluginExecutionContext>): void {
    this.context = {
      ...this.context,
      ...context,
      timestamp: new Date(),
    };
  }

  /**
   * Get current execution context
   */
  getContext(): PluginExecutionContext {
    return { ...this.context };
  }

  /**
   * Register hooks for a plugin
   */
  private registerHooks(plugin: ChatPlugin): void {
    const hookNames: (keyof ChatPlugin)[] = [
      'onInit',
      'onDestroy',
      'onMessageReceived',
      'onMessageSent',
      'onToolCall',
      'onToolResult',
      'onSessionCreated',
      'onSessionDeleted',
      'onSessionChanged',
      'onError',
      'onStreamStart',
      'onStreamEnd',
      'onStreamChunk',
    ];

    for (const hookName of hookNames) {
      if (plugin[hookName]) {
        const hookKey = hookName as string;

        if (!this.hooks.has(hookKey)) {
          this.hooks.set(hookKey, []);
        }

        this.hooks.get(hookKey)!.push(plugin);

        // Sort by priority (highest first)
        this.hooks.get(hookKey)!.sort((a, b) => b.priority - a.priority);
      }
    }
  }

  /**
   * Get plugin statistics
   */
  getStats(): {
    totalPlugins: number;
    enabledPlugins: number;
    hooks: Record<string, number>;
  } {
    const stats = {
      totalPlugins: this.plugins.size,
      enabledPlugins: this.enabledPlugins.size,
      hooks: {} as Record<string, number>,
    };

    for (const [hookName, plugins] of this.hooks.entries()) {
      stats.hooks[hookName] = plugins.length;
    }

    return stats;
  }
}

// Global plugin registry instance
let globalRegistry: PluginRegistry | null = null;

/**
 * Get or create the global plugin registry
 */
export function getPluginRegistry(): PluginRegistry {
  if (!globalRegistry) {
    globalRegistry = new PluginRegistry();
  }
  return globalRegistry;
}

/**
 * Reset the global plugin registry (useful for testing)
 */
export function resetPluginRegistry(): void {
  globalRegistry = null;
}
