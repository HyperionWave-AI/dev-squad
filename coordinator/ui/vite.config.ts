import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api/mcp': {
        // In Docker, use service name. Outside Docker, use localhost:7095
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:7095',
        changeOrigin: true
        // Don't rewrite - bridge expects /api/mcp prefix
      },
      '/api/knowledge': {
        // Proxy knowledge API calls to the same MCP bridge (coordinator MCP server handles these)
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:7095',
        changeOrigin: true
        // Don't rewrite - coordinator expects /api/knowledge prefix
      },
      '/bridge-health': {
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:7095',
        changeOrigin: true,
        rewrite: () => '/health'
      }
    }
  }
})
