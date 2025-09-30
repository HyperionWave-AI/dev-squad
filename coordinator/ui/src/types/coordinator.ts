// TypeScript types matching the coordinator MCP server

export interface HumanTask {
  id: string;
  title: string;
  description: string;
  prompt: string;
  status: TaskStatus;
  priority: Priority;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
  createdBy: string;
  tags: string[];
}

export interface TodoItem {
  id: string;
  description: string;
  status: TodoStatus;
  createdAt: string;
  completedAt?: string;
  notes?: string;
}

export interface AgentTask {
  id: string;
  humanTaskId: string;
  agentName: string;
  role: string;
  todos: TodoItem[];
  status: TaskStatus;
  createdAt: string;
  updatedAt: string;
  notes?: string;
}

export interface AgentRole {
  id: string;
  agentName: string;
  role: string;
  squad: string;
  capabilities: string[];
  domain: string[];
  mcpTools: string[];
  createdAt: string;
  updatedAt: string;
}

export interface KnowledgeEntry {
  id: string;
  collection: string;
  text: string;
  embedding?: number[];
  metadata: Record<string, any>;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  tags: string[];
}


export type TaskStatus = 'pending' | 'in_progress' | 'completed' | 'blocked';
export type TodoStatus = 'pending' | 'in_progress' | 'completed';
export type Priority = 'low' | 'medium' | 'high' | 'urgent';

export interface TaskWithChildren extends HumanTask {
  agentTasks?: AgentTask[];
}