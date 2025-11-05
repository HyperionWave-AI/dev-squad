import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // Always use /ui/ base path since UI is always served through Go proxy at /ui/ route
  base: '/ui/',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@atoms': path.resolve(__dirname, './src/components/atoms'),
      '@molecules': path.resolve(__dirname, './src/components/molecules'),
      '@organisms': path.resolve(__dirname, './src/components/organisms'),
      '@templates': path.resolve(__dirname, './src/components/templates'),
      '@pages': path.resolve(__dirname, './src/pages'),
      '@hooks': path.resolve(__dirname, './src/hooks'),
      '@services': path.resolve(__dirname, './src/services'),
      '@types': path.resolve(__dirname, './src/types'),
      '@utils': path.resolve(__dirname, './src/utils'),
    },
  },
  server: {
    port: 4588,
    proxy: {
      '/api/mcp': {
        // Point to hyper backend on port 7878 (configured via .env.hyper)
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:7878',
        changeOrigin: true
        // Don't rewrite - bridge expects /api/mcp prefix
      },
      '/api/v1': {
        // Proxy all v1 API calls to the coordinator
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:7878',
        changeOrigin: true
        // Don't rewrite - coordinator expects /api/v1 prefix
      },
      '/api/knowledge': {
        // Proxy knowledge API calls to the same MCP bridge (coordinator MCP server handles these)
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:7878',
        changeOrigin: true
        // Don't rewrite - coordinator expects /api/knowledge prefix
      },
      '/api/tasks': {
        // Proxy task board API calls to the MCP bridge
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:7878',
        changeOrigin: true
        // Don't rewrite - bridge expects /api/tasks prefix
      },
      '/api/agent-tasks': {
        // Proxy agent tasks API calls to the MCP bridge
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:7878',
        changeOrigin: true
        // Don't rewrite - bridge expects /api/agent-tasks prefix
      },
      '/bridge-health': {
        target: process.env.VITE_MCP_BRIDGE_URL || 'http://localhost:7878',
        changeOrigin: true,
        rewrite: () => '/health'
      }
    }
  }
})
