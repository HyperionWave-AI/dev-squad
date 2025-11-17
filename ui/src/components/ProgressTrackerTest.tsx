/**
 * Progress Tracker Test Component
 * 
 * Simple test component to verify the enhanced ProgressTracker functionality.
 */

import React, { useState } from 'react';
import { ProgressTracker, type TrackableEvent } from './organisms/ProgressTracker';

export const ProgressTrackerTest: React.FC = () => {
  const [events, setEvents] = useState<TrackableEvent[]>([]);
  const [showTracker, setShowTracker] = useState(false);

  const testBasicScenario = () => {
    setShowTracker(true);
    
    const testEvents: TrackableEvent[] = [
      {
        id: 'test-1',
        type: 'progress',
        step: 1,
        totalSteps: 3,
        description: 'Starting test',
        status: 'completed',
        timestamp: new Date(),
      },
      {
        id: 'test-2',
        type: 'progress',
        step: 2,
        totalSteps: 3,
        description: 'Processing data',
        status: 'in_progress',
        timestamp: new Date(),
      },
      {
        id: 'test-3',
        type: 'progress',
        step: 3,
        totalSteps: 3,
        description: 'Finalizing',
        status: 'pending',
        timestamp: new Date(),
      },
    ];
    
    setEvents(testEvents);
  };

  const clearTest = () => {
    setEvents([]);
    setShowTracker(false);
  };

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">Progress Tracker Test</h1>
      
      <div className="space-x-4 mb-6">
        <button
          onClick={testBasicScenario}
          className="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
        >
          Test Basic Scenario
        </button>
        
        <button
          onClick={clearTest}
          className="px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600"
        >
          Clear Test
        </button>
      </div>

      <div className="bg-gray-100 p-4 rounded">
        <p>Events: {events.length}</p>
        <p>Tracker visible: {showTracker ? 'Yes' : 'No'}</p>
      </div>

      {showTracker && (
        <ProgressTracker
          events={events}
          showTypingIndicator={false}
          onClose={clearTest}
        />
      )}
    </div>
  );
};