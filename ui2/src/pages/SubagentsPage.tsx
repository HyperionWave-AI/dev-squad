import React, { useState, useEffect } from 'react';
import * as Accordion from '@radix-ui/react-accordion';
import { Bot, Search } from 'lucide-react';
import { subagentsService } from '@/services/subagentsService';
import type { Subagent } from '@/types/subagent';
import { Input } from '@/components/atoms/Input';
import { Badge } from '@/components/atoms/Badge';
import { PageHeader } from '@/components/organisms/PageHeader';

export function SubagentsPage() {
  const [subagents, setSubagents] = useState<Subagent[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);

  useEffect(() => {
    loadSubagents();
  }, []);

  const loadSubagents = async () => {
    try {
      setLoading(true);
      const { subagents: data } = await subagentsService.listSubagents();
      setSubagents(data);
    } catch (error) {
      console.error('Failed to load subagents:', error);
    } finally {
      setLoading(false);
    }
  };

  const categories = Array.from(new Set(subagents.map(a => a.category)));

  const filteredAgents = subagents.filter(agent => {
    const matchesSearch = !searchQuery ||
      agent.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      agent.description.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesCategory = !selectedCategory || agent.category === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <div className="container mx-auto p-6 space-y-6 max-w-7xl">
        {/* Header */}
        <PageHeader
          title="Subagents"
          description="Browse available specialist agents and their capabilities"
          icon={<Bot className="h-8 w-8" />}
          gradientFrom="#3b82f6"
          gradientTo="#8b5cf6"
        />

        {/* Search & Filters - Glassmorphic Container */}
        <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-4 shadow-lg">
          <div className="flex gap-4">
            <div className="flex-1">
              <Input
                placeholder="Search agents..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                icon={<Search className="h-4 w-4" />}
              />
            </div>
            <div className="flex gap-2">
              {categories.map((cat) => (
                <button
                  key={cat}
                  onClick={() => setSelectedCategory(selectedCategory === cat ? null : cat)}
                  className={`px-3 py-2 rounded-md text-sm transition-colors ${
                    selectedCategory === cat
                      ? 'bg-blue-500 text-white'
                      : 'bg-white/50 dark:bg-gray-700/50 hover:bg-white/70 dark:hover:bg-gray-700/70'
                  }`}
                >
                  {cat}
                </button>
              ))}
            </div>
          </div>
        </div>

        {/* Content */}
        {loading ? (
          <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-12 shadow-lg">
            <div className="text-center text-gray-600 dark:text-gray-400">Loading subagents...</div>
          </div>
        ) : filteredAgents.length === 0 ? (
          <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-12 shadow-lg">
            <div className="text-center text-gray-600 dark:text-gray-400">
              <Bot className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>No agents found</p>
            </div>
          </div>
        ) : (
          <Accordion.Root type="single" collapsible className="space-y-4">
            {filteredAgents.map((agent) => (
              <Accordion.Item
                key={agent.name}
                value={agent.name}
                className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg overflow-hidden shadow-lg"
              >
                <Accordion.Header>
                  <Accordion.Trigger className="w-full px-4 py-3 text-left hover:bg-white/50 dark:hover:bg-gray-700/50 transition-colors flex justify-between items-center">
                    <div className="flex-1">
                      <div className="font-semibold text-gray-900 dark:text-gray-100">{agent.name}</div>
                      <div className="text-sm text-gray-600 dark:text-gray-400 mt-1">{agent.description}</div>
                    </div>
                    <Badge variant="outline">{agent.category}</Badge>
                  </Accordion.Trigger>
                </Accordion.Header>
                <Accordion.Content className="px-4 py-3 bg-white/30 dark:bg-gray-700/30">
                  <div className="space-y-3">
                    <div>
                      <h4 className="text-sm font-semibold mb-1 text-gray-900 dark:text-gray-100">Tools</h4>
                      <div className="flex flex-wrap gap-1">
                        {agent.tools.map((tool, i) => (
                          <Badge key={i} variant="outline" className="text-xs">{tool}</Badge>
                        ))}
                      </div>
                    </div>
                    {agent.examples && agent.examples.length > 0 && (
                      <div>
                        <h4 className="text-sm font-semibold mb-1 text-gray-900 dark:text-gray-100">Examples</h4>
                        <ul className="text-sm list-disc list-inside space-y-1 text-gray-700 dark:text-gray-300">
                          {agent.examples.map((ex, i) => (
                            <li key={i}>{ex}</li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>
                </Accordion.Content>
              </Accordion.Item>
            ))}
          </Accordion.Root>
        )}
      </div>
    </div>
  );
}
