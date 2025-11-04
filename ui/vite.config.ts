import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // Always use /ui/ base path since UI is always served through Go proxy at /ui/ route
  base: '/ui/',
  server: {
    proxy: {
      '/api/mcp': {
        // In Docker, use service name. Outside Docker, use localhost:4097
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:4097',
        changeOrigin: true
        // Don't rewrite - bridge expects /api/mcp prefix
      },
      '/api/v1': {
        // Proxy all v1 API calls to the coordinator
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:4097',
        changeOrigin: true
        // Don't rewrite - coordinator expects /api/v1 prefix
      },
      '/api/knowledge': {
        // Proxy knowledge API calls to the same MCP bridge (coordinator MCP server handles these)
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:4097',
        changeOrigin: true
        // Don't rewrite - coordinator expects /api/knowledge prefix
      },
      '/api/tasks': {
        // Proxy task board API calls to the MCP bridge
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:4097',
        changeOrigin: true
        // Don't rewrite - bridge expects /api/tasks prefix
      },
      '/api/agent-tasks': {
        // Proxy agent tasks API calls to the MCP bridge
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:4097',
        changeOrigin: true
        // Don't rewrite - bridge expects /api/agent-tasks prefix
      },
      '/bridge-health': {
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:4097',
        changeOrigin: true,
        rewrite: () => '/health'
      }
    }
  }
})
