import { useState } from 'react';
import { TaskDashboard } from './components/TaskDashboard';
import { KnowledgeBrowser } from './components/KnowledgeBrowser';
import './App.css';

type View = 'dashboard' | 'knowledge';

function App() {
  const [currentView, setCurrentView] = useState<View>('dashboard');

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50">
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">
                🚀 Hyperion Coordinator
              </h1>
              <p className="text-sm text-gray-600 mt-1">
                Task & Knowledge Management for Parallel Squad System
              </p>
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setCurrentView('dashboard')}
                className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                  currentView === 'dashboard'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                }`}
              >
                📊 Dashboard
              </button>
              <button
                onClick={() => setCurrentView('knowledge')}
                className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                  currentView === 'knowledge'
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                }`}
              >
                🧠 Knowledge
              </button>
            </div>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {currentView === 'dashboard' && <TaskDashboard />}
        {currentView === 'knowledge' && <KnowledgeBrowser />}
      </main>

      <footer className="bg-white border-t mt-12">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <p className="text-center text-sm text-gray-500">
            Hyperion AI Platform • Coordinator MCP • Local Dev Environment
          </p>
        </div>
      </footer>
    </div>
  );
}

export default App;
