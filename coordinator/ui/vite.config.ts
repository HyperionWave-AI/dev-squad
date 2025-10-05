import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api/mcp': {
        // Use port 8095 for HTTP bridge (local dev & Docker)
        target: process.env.MCP_BRIDGE_URL || 'http://localhost:8095',
        changeOrigin: true
        // Don't rewrite - bridge expects /api/mcp prefix
      },
      '/bridge-health': {
        target: process.env.MCP_BRIDGE_URL || 'http://localhost:8095',
        changeOrigin: true,
        rewrite: () => '/health'
      }
    }
  }
})
