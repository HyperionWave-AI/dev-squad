/**
 * Mock Task Data for Kanban Board Testing
 *
 * Provides consistent test data for validating:
 * - Task rendering across different statuses
 * - Priority badge colors
 * - Drag-and-drop functionality
 * - Responsive behavior
 */

import type { HumanTask, AgentTask } from '../../src/types/coordinator';

export const mockHumanTasks: HumanTask[] = [
  {
    id: 'human-task-1',
    prompt: 'Implement user authentication system',
    status: 'in_progress',
    createdAt: '2025-09-30T10:00:00.000Z',
    updatedAt: '2025-09-30T11:30:00.000Z',
  },
  {
    id: 'human-task-2',
    prompt: 'Build Kanban board UI with MUI',
    status: 'completed',
    createdAt: '2025-09-30T09:00:00.000Z',
    updatedAt: '2025-09-30T15:00:00.000Z',
  },
  {
    id: 'human-task-3',
    prompt: 'Setup CI/CD pipeline',
    status: 'pending',
    createdAt: '2025-09-30T14:00:00.000Z',
    updatedAt: '2025-09-30T14:00:00.000Z',
  },
  {
    id: 'human-task-4',
    prompt: 'Fix database connection issues',
    status: 'blocked',
    createdAt: '2025-09-30T08:00:00.000Z',
    updatedAt: '2025-09-30T12:00:00.000Z',
  },
];

export const mockAgentTasks: AgentTask[] = [
  {
    id: 'agent-task-1',
    humanTaskId: 'human-task-1',
    agentName: 'Backend Services Specialist',
    role: 'Implement JWT authentication',
    status: 'in_progress',
    todos: [
      {
        id: 'todo-1-1',
        description: 'Setup JWT library',
        status: 'completed',
        createdAt: '2025-09-30T10:15:00.000Z',
      },
      {
        id: 'todo-1-2',
        description: 'Create auth middleware',
        status: 'in_progress',
        createdAt: '2025-09-30T10:30:00.000Z',
      },
      {
        id: 'todo-1-3',
        description: 'Add token validation',
        status: 'pending',
        createdAt: '2025-09-30T10:45:00.000Z',
      },
    ],
    createdAt: '2025-09-30T10:10:00.000Z',
    updatedAt: '2025-09-30T11:30:00.000Z',
  },
  {
    id: 'agent-task-2',
    humanTaskId: 'human-task-2',
    agentName: 'ui-dev',
    role: 'Build Kanban board with drag-and-drop',
    status: 'completed',
    todos: [
      {
        id: 'todo-2-1',
        description: 'Install MUI dependencies',
        status: 'completed',
        createdAt: '2025-09-30T09:15:00.000Z',
      },
      {
        id: 'todo-2-2',
        description: 'Create Kanban board component',
        status: 'completed',
        createdAt: '2025-09-30T09:30:00.000Z',
      },
      {
        id: 'todo-2-3',
        description: 'Implement drag-and-drop',
        status: 'completed',
        createdAt: '2025-09-30T10:00:00.000Z',
      },
    ],
    createdAt: '2025-09-30T09:10:00.000Z',
    updatedAt: '2025-09-30T15:00:00.000Z',
  },
  {
    id: 'agent-task-3',
    humanTaskId: 'human-task-3',
    agentName: 'Infrastructure Automation Specialist',
    role: 'Configure GitHub Actions',
    status: 'pending',
    todos: [
      {
        id: 'todo-3-1',
        description: 'Create workflow YAML',
        status: 'pending',
        createdAt: '2025-09-30T14:15:00.000Z',
      },
      {
        id: 'todo-3-2',
        description: 'Setup GKE deployment',
        status: 'pending',
        createdAt: '2025-09-30T14:30:00.000Z',
      },
    ],
    createdAt: '2025-09-30T14:10:00.000Z',
    updatedAt: '2025-09-30T14:10:00.000Z',
  },
  {
    id: 'agent-task-4',
    humanTaskId: 'human-task-4',
    agentName: 'Data Platform Specialist',
    role: 'Debug MongoDB connection',
    status: 'blocked',
    todos: [
      {
        id: 'todo-4-1',
        description: 'Check connection string',
        status: 'completed',
        createdAt: '2025-09-30T08:15:00.000Z',
      },
      {
        id: 'todo-4-2',
        description: 'Verify network access',
        status: 'blocked',
        createdAt: '2025-09-30T08:30:00.000Z',
      },
    ],
    createdAt: '2025-09-30T08:10:00.000Z',
    updatedAt: '2025-09-30T12:00:00.000Z',
  },
];

// Priority test data for badge color validation
export const priorityTestCases = [
  {
    priority: 'high',
    expectedColor: 'red',
    description: 'Critical production bug',
  },
  {
    priority: 'medium',
    expectedColor: 'yellow',
    description: 'Feature enhancement',
  },
  {
    priority: 'low',
    expectedColor: 'green',
    description: 'Documentation update',
  },
];

// Status column definitions for Kanban board
export const kanbanColumns = [
  {
    id: 'pending',
    title: 'Pending',
    description: 'Tasks not yet started',
    dataTestId: 'kanban-column-pending',
  },
  {
    id: 'in_progress',
    title: 'In Progress',
    description: 'Tasks currently being worked on',
    dataTestId: 'kanban-column-in-progress',
  },
  {
    id: 'completed',
    title: 'Completed',
    description: 'Successfully finished tasks',
    dataTestId: 'kanban-column-completed',
  },
  {
    id: 'blocked',
    title: 'Blocked',
    description: 'Tasks waiting on dependencies',
    dataTestId: 'kanban-column-blocked',
  },
];