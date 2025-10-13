import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import { defineConfig, globalIgnores } from 'eslint/config'

export default defineConfig([
  globalIgnores(['dist']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs['recommended-latest'],
      reactRefresh.configs.vite,
    ],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
    },
    rules: {
      // Prevent ALL direct MCP tool usage in UI components
      // ARCHITECTURE RULE: UI → REST API → MCP (never UI → MCP directly)
      'no-restricted-imports': ['error', {
        paths: [
          // Block mcpClient (coordinator operations)
          {
            name: './services/mcpClient',
            message: 'ARCHITECTURE VIOLATION: Direct MCP calls are absolutely prohibited! Use ./services/restClient instead.',
          }, {
            name: '../services/mcpClient',
            message: 'ARCHITECTURE VIOLATION: Direct MCP calls are absolutely prohibited! Use ../services/restClient instead.',
          }, {
            name: '../../services/mcpClient',
            message: 'ARCHITECTURE VIOLATION: Direct MCP calls are absolutely prohibited! Use ../../services/restClient instead.',
          },
          // Block codeClient (code indexing operations)
          {
            name: './services/codeClient',
            message: 'ARCHITECTURE VIOLATION: Direct MCP calls are absolutely prohibited! Use ./services/restCodeClient instead.',
          }, {
            name: '../services/codeClient',
            message: 'ARCHITECTURE VIOLATION: Direct MCP calls are absolutely prohibited! Use ../services/restCodeClient instead.',
          }, {
            name: '../../services/codeClient',
            message: 'ARCHITECTURE VIOLATION: Direct MCP calls are absolutely prohibited! Use ../../services/restCodeClient instead.',
          },
        ],
        patterns: [
          {
            group: ['**/services/mcpClient'],
            message: 'ARCHITECTURE VIOLATION: Direct MCP calls are absolutely prohibited! Use restClient.ts instead.',
          },
          {
            group: ['**/services/codeClient'],
            message: 'ARCHITECTURE VIOLATION: Direct MCP calls are absolutely prohibited! Use restCodeClient.ts instead.',
          },
        ],
      }],
    },
  },
  // Exception: Allow mcpClient.ts and codeClient.ts themselves to exist (deprecated, will be removed)
  {
    files: ['**/services/mcpClient.ts', '**/services/codeClient.ts'],
    rules: {
      'no-restricted-imports': 'off',
    },
  },
])
