/**
 * Tauri Bindings for Hyperion Coordinator MCP Tools
 *
 * Use these functions in your React app to call MCP tools directly
 * via Tauri commands (no HTTP needed).
 *
 * @example
 * ```typescript
 * import { invoke } from '@tauri-apps/api/core'
 * import { createHumanTask, listHumanTasks } from './tauri-bindings'
 *
 * // Create a task
 * const result = await createHumanTask('Build authentication system')
 *
 * // List all tasks
 * const tasks = await listHumanTasks()
 * ```
 */

import { invoke } from '@tauri-apps/api/core'

// ============================================================================
// MCP Tool Types
// ============================================================================

export interface HumanTask {
  id: string
  prompt: string
  status: 'pending' | 'in_progress' | 'completed' | 'blocked'
  createdAt: string
  updatedAt: string
  notes?: string
}

export interface AgentTask {
  id: string
  humanTaskId: string
  agentName: string
  role: string
  contextSummary?: string
  filesModified?: string[]
  todos?: TodoItem[]
  status: 'pending' | 'in_progress' | 'completed' | 'blocked'
  createdAt: string
  updatedAt: string
  notes?: string
}

export interface TodoItem {
  id: string
  description: string
  status: 'pending' | 'in_progress' | 'completed'
  filePath?: string
  functionName?: string
  contextHint?: string
  notes?: string
}

export interface KnowledgeResult {
  text: string
  score: number
  metadata?: Record<string, any>
}

// ============================================================================
// Generic MCP Tool Call
// ============================================================================

/**
 * Call any MCP tool by name with arguments
 *
 * @param name - MCP tool name (e.g., "coordinator_create_human_task")
 * @param arguments - Tool-specific arguments
 * @returns Tool response
 */
export async function callMcpTool(
  name: string,
  args: Record<string, any> = {}
): Promise<any> {
  return invoke('call_mcp_tool', { name, arguments: args })
}

// ============================================================================
// Human Task Operations
// ============================================================================

/**
 * Create a new human task
 *
 * @param prompt - Task description
 * @returns Created task
 */
export async function createHumanTask(prompt: string): Promise<HumanTask> {
  return invoke('create_human_task', { prompt })
}

/**
 * List all human tasks
 *
 * @returns Array of human tasks
 */
export async function listHumanTasks(): Promise<HumanTask[]> {
  const result = await invoke<{ tasks: HumanTask[] }>('list_human_tasks')
  return result.tasks || []
}

// ============================================================================
// Agent Task Operations
// ============================================================================

export interface CreateAgentTaskParams {
  humanTaskId: string
  agentName: string
  role: string
  contextSummary?: string
  filesModified?: string[]
  todos?: Array<{
    description: string
    filePath?: string
    functionName?: string
    contextHint?: string
  }>
}

/**
 * Create a new agent task
 *
 * @param params - Agent task parameters
 * @returns Created agent task
 */
export async function createAgentTask(
  params: CreateAgentTaskParams
): Promise<AgentTask> {
  return invoke('create_agent_task', {
    humanTaskId: params.humanTaskId,
    agentName: params.agentName,
    role: params.role,
    contextSummary: params.contextSummary,
    filesModified: params.filesModified,
    todos: params.todos,
  })
}

/**
 * List agent tasks
 *
 * @param agentName - Filter by agent name (optional)
 * @param humanTaskId - Filter by human task ID (optional)
 * @returns Array of agent tasks
 */
export async function listAgentTasks(
  agentName?: string,
  humanTaskId?: string
): Promise<AgentTask[]> {
  const result = await invoke<{ tasks: AgentTask[] }>('list_agent_tasks', {
    agentName: agentName || null,
    humanTaskId: humanTaskId || null,
  })
  return result.tasks || []
}

/**
 * Update task status
 *
 * @param taskId - Task ID (human or agent)
 * @param status - New status
 * @param notes - Optional notes
 * @returns Updated task
 */
export async function updateTaskStatus(
  taskId: string,
  status: 'pending' | 'in_progress' | 'completed' | 'blocked',
  notes?: string
): Promise<any> {
  return invoke('update_task_status', { taskId, status, notes: notes || null })
}

// ============================================================================
// Knowledge Operations
// ============================================================================

/**
 * Store knowledge in a collection
 *
 * @param collection - Collection name
 * @param text - Knowledge text
 * @param metadata - Optional metadata
 * @returns Operation result
 */
export async function upsertKnowledge(
  collection: string,
  text: string,
  metadata?: Record<string, any>
): Promise<any> {
  return invoke('upsert_knowledge', {
    collection,
    text,
    metadata: metadata || null,
  })
}

/**
 * Query knowledge from a collection
 *
 * @param collection - Collection name
 * @param query - Search query
 * @param limit - Max results (default: 5)
 * @returns Array of knowledge results
 */
export async function queryKnowledge(
  collection: string,
  query: string,
  limit: number = 5
): Promise<KnowledgeResult[]> {
  const result = await invoke<{ results: KnowledgeResult[] }>(
    'query_knowledge',
    { collection, query, limit }
  )
  return result.results || []
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Check if the Hyperion server is healthy
 *
 * @returns 'healthy' if server is running
 */
export async function checkServerHealth(): Promise<string> {
  return invoke('check_server_health')
}

/**
 * Get the server URL
 *
 * @returns Server URL (e.g., "http://localhost:7095/ui")
 */
export async function getServerUrl(): Promise<string> {
  return invoke('get_server_url')
}
