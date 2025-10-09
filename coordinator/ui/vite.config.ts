import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api/mcp': {
        target: 'http://localhost:8095',
        changeOrigin: true
      },
      '/bridge-health': {
        target: 'http://localhost:8095',
        changeOrigin: true,
        rewrite: (path) => '/health'
      }
    }
  }
})
