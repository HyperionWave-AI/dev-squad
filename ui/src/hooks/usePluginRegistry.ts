/**
 * usePluginRegistry Hook
 *
 * Provides plugin registry management for React components.
 * Handles plugin initialization, lifecycle, and mode toggling.
 */

import { useEffect, useRef, useCallback } from 'react';
import { getPluginRegistry } from '@/services/pluginRegistry';
import { errorPreventionPlugin } from '@/plugins/errorPreventionPlugin';
import { complexityAnalysisPlugin } from '@/plugins/complexityAnalysisPlugin';
import type { IPluginRegistry } from '@/types/plugin';

export interface UsePluginRegistryOptions {
  autoInitialize?: boolean;
  autoRegisterBuiltins?: boolean;
}

export function usePluginRegistry(options: UsePluginRegistryOptions = {}) {
  const {
    autoInitialize = true,
    autoRegisterBuiltins = true,
  } = options;

  const registryRef = useRef<IPluginRegistry | null>(null);
  const initializedRef = useRef(false);

  // Initialize registry
  useEffect(() => {
    if (initializedRef.current) return;

    const registry = getPluginRegistry();
    registryRef.current = registry;

    // Register built-in plugins
    if (autoRegisterBuiltins) {
      registry.register(errorPreventionPlugin);
      registry.register(complexityAnalysisPlugin);

      // Disable by default - users must explicitly enable
      registry.setPluginEnabled('error-prevention', false);
      registry.setPluginEnabled('complexity-analysis', false);

      console.log('[usePluginRegistry] Built-in plugins registered');
    }

    // Initialize all plugins
    if (autoInitialize) {
      registry.initializeAll().catch((error) => {
        console.error('[usePluginRegistry] Failed to initialize plugins:', error);
      });
    }

    initializedRef.current = true;

    // Cleanup on unmount
    return () => {
      if (autoInitialize) {
        registry.destroyAll().catch((error) => {
          console.error('[usePluginRegistry] Failed to destroy plugins:', error);
        });
      }
    };
  }, [autoInitialize, autoRegisterBuiltins]);

  // Toggle error prevention mode
  const toggleErrorPrevention = useCallback((enabled: boolean) => {
    if (!registryRef.current) return;

    const plugin = registryRef.current.getPlugin('error-prevention');
    if (plugin && 'setEnabled' in plugin) {
      (plugin as any).setEnabled(enabled);
      registryRef.current.setPluginEnabled('error-prevention', enabled);
    }
  }, []);

  // Toggle complexity analysis mode
  const toggleComplexityAnalysis = useCallback((enabled: boolean) => {
    if (!registryRef.current) return;

    const plugin = registryRef.current.getPlugin('complexity-analysis');
    if (plugin && 'setEnabled' in plugin) {
      (plugin as any).setEnabled(enabled);
      registryRef.current.setPluginEnabled('complexity-analysis', enabled);
    }
  }, []);

  // Get error prevention status
  const isErrorPreventionEnabled = useCallback((): boolean => {
    if (!registryRef.current) return false;
    return registryRef.current.isPluginEnabled('error-prevention');
  }, []);

  // Get complexity analysis status
  const isComplexityAnalysisEnabled = useCallback((): boolean => {
    if (!registryRef.current) return false;
    return registryRef.current.isPluginEnabled('complexity-analysis');
  }, []);

  // Get registry instance
  const getRegistry = useCallback((): IPluginRegistry | null => {
    return registryRef.current;
  }, []);

  return {
    registry: registryRef.current,
    toggleErrorPrevention,
    toggleComplexityAnalysis,
    isErrorPreventionEnabled,
    isComplexityAnalysisEnabled,
    getRegistry,
  };
}
