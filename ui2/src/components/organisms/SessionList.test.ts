// Simple test to verify hierarchy display functionality
// This would normally be in a proper test file, but creating here for verification

import { organizeSessionsHierarchy, getParentId, isSubchatSession } from './SessionList';

// Test data with parent-child relationships
const testSessions = [
  {
    id: 'parent-1',
    title: 'Main Chat Session',
    timestamp: '2024-01-01T10:00:00Z',
    messageCount: 5,
    isSubchat: false
  },
  {
    id: 'child-1-1',
    title: 'Subchat: Code Review',
    parentSessionId: 'parent-1',
    timestamp: '2024-01-01T10:30:00Z',
    messageCount: 3,
    isSubchat: true
  },
  {
    id: 'child-1-2',
    title: 'Subchat: Bug Fix',
    parentChatId: 'parent-1', // Testing alternative field name
    timestamp: '2024-01-01T11:00:00Z',
    messageCount: 2,
    isSubchat: true
  },
  {
    id: 'parent-2',
    title: 'Another Main Chat',
    timestamp: '2024-01-01T12:00:00Z',
    messageCount: 8,
    isSubchat: false
  }
];

// Test helper functions
console.log('Testing helper functions:');
console.log('getParentId(child-1-1):', getParentId(testSessions[1])); // Should be 'parent-1'
console.log('getParentId(child-1-2):', getParentId(testSessions[2])); // Should be 'parent-1'
console.log('isSubchatSession(child-1-1):', isSubchatSession(testSessions[1])); // Should be true
console.log('isSubchatSession(parent-1):', isSubchatSession(testSessions[0])); // Should be false

// Test hierarchy organization
const { mainSessions, subchatsMap } = organizeSessionsHierarchy(testSessions);
console.log('\nHierarchy organization results:');
console.log('Main sessions count:', mainSessions.length); // Should be 2
console.log('Subchats for parent-1:', subchatsMap.get('parent-1')?.length); // Should be 2
console.log('Subchats for parent-2:', subchatsMap.get('parent-2')?.length); // Should be undefined

export { testSessions };