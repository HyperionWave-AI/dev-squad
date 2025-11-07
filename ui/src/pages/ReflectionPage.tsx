import { useState, useEffect } from 'react';
import * as Tabs from '@radix-ui/react-tabs';
import * as Accordion from '@radix-ui/react-accordion';
import { Brain, TrendingUp, Search, AlertCircle, CheckCircle, XCircle, ChevronDown } from 'lucide-react';
import { reflectionService } from '@/services/reflectionService';
import type { Decision, Lesson } from '@/types/reflection';
import { Button } from '@/components/atoms/Button';
import { Input } from '@/components/atoms/Input';
import { Badge } from '@/components/atoms/Badge';
import { PageHeader } from '@/components/organisms/PageHeader';

export function ReflectionPage() {
  const [decisions, setDecisions] = useState<Decision[]>([]);
  const [lessons, setLessons] = useState<Lesson[]>([]);
  const [queryResults, setQueryResults] = useState<Lesson[]>([]);
  const [loading, setLoading] = useState(false);
  const [queryText, setQueryText] = useState('');

  useEffect(() => {
    loadDecisions();
    loadLessons();
  }, []);

  const loadDecisions = async () => {
    try {
      setLoading(true);
      const { decisions: data } = await reflectionService.listDecisions(50);
      setDecisions(data || []);
    } catch (error) {
      console.error('Failed to load decisions:', error);
      setDecisions([]);
    } finally {
      setLoading(false);
    }
  };

  const loadLessons = async () => {
    try {
      const { lessons: data } = await reflectionService.listLessons(50);
      setLessons(data || []);
    } catch (error) {
      console.error('Failed to load lessons:', error);
      setLessons([]);
    }
  };

  const handleQuery = async () => {
    if (!queryText.trim()) return;
    try {
      setLoading(true);
      const { lessons: data } = await reflectionService.queryLessons(queryText, 10);
      setQueryResults(data || []);
    } catch (error) {
      console.error('Failed to query lessons:', error);
      setQueryResults([]);
    } finally {
      setLoading(false);
    }
  };

  const getConfidenceBadge = (confidence: number) => {
    if (confidence >= 0.9) return <Badge variant="success">High ({(confidence * 100).toFixed(0)}%)</Badge>;
    if (confidence >= 0.7) return <Badge variant="warning">Medium ({(confidence * 100).toFixed(0)}%)</Badge>;
    return <Badge variant="default">Low ({(confidence * 100).toFixed(0)}%)</Badge>;
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950 p-6">
      <div className="max-w-7xl mx-auto space-y-6">
        {/* Header */}
        <PageHeader
          title="Reflection System"
          description="Metacognitive learning - Track decisions, extract lessons, and query relevant patterns"
          icon={<Brain className="h-8 w-8" />}
          gradientFrom="#ec4899"
          gradientTo="#f43f5e"
        />

        {/* Tabs Card with Glassmorphism */}
        <div className="backdrop-blur-sm bg-white/80 border border-white/20 shadow-2xl rounded-2xl overflow-hidden">
          <Tabs.Root defaultValue="decisions" className="w-full">
            <Tabs.List className="flex gap-1 border-b border-white/30 bg-white/40 px-4 backdrop-blur-sm">
              <Tabs.Trigger
                value="decisions"
                className="px-6 py-3 text-sm font-medium text-gray-600 hover:text-gray-900 hover:bg-white/50 rounded-t-lg transition-all data-[state=active]:text-blue-600 data-[state=active]:bg-white data-[state=active]:border-b-2 data-[state=active]:border-blue-600 data-[state=active]:-mb-px"
              >
                <div className="flex items-center gap-2">
                  <TrendingUp className="h-4 w-4" />
                  Decisions
                  <Badge variant="default" className="ml-1">{decisions.length}</Badge>
                </div>
              </Tabs.Trigger>
              <Tabs.Trigger
                value="lessons"
                className="px-6 py-3 text-sm font-medium text-gray-600 hover:text-gray-900 hover:bg-white/50 rounded-t-lg transition-all data-[state=active]:text-blue-600 data-[state=active]:bg-white data-[state=active]:border-b-2 data-[state=active]:border-blue-600 data-[state=active]:-mb-px"
              >
                <div className="flex items-center gap-2">
                  <Brain className="h-4 w-4" />
                  Lessons Learned
                  <Badge variant="default" className="ml-1">{lessons.length}</Badge>
                </div>
              </Tabs.Trigger>
              <Tabs.Trigger
                value="query"
                className="px-6 py-3 text-sm font-medium text-gray-600 hover:text-gray-900 hover:bg-white/50 rounded-t-lg transition-all data-[state=active]:text-blue-600 data-[state=active]:bg-white data-[state=active]:border-b-2 data-[state=active]:border-blue-600 data-[state=active]:-mb-px"
              >
                <div className="flex items-center gap-2">
                  <Search className="h-4 w-4" />
                  Query Lessons
                </div>
              </Tabs.Trigger>
            </Tabs.List>

            {/* Decisions Tab */}
            <Tabs.Content value="decisions" className="p-6">
              {loading ? (
                <div className="text-center py-12">
                  <div className="inline-block h-8 w-8 animate-spin rounded-full border-4 border-solid border-blue-600 border-r-transparent"></div>
                  <p className="mt-4 text-gray-600">Loading decisions...</p>
                </div>
              ) : decisions.length === 0 ? (
                <div className="text-center py-16 text-gray-500">
                  <Brain className="h-16 w-16 mx-auto mb-4 opacity-30" />
                  <p className="text-lg font-medium">No decisions recorded yet</p>
                  <p className="text-sm mt-2">Decisions will appear here as they are recorded by the system</p>
                </div>
              ) : (
                <Accordion.Root type="single" collapsible className="space-y-4">
                  {decisions.map((decision) => {
                    // Add safety checks for data structure
                    if (!decision?.data?.decision) return null;

                    return (
                    <Accordion.Item
                      key={decision.id}
                      value={decision.id}
                      className="backdrop-blur-sm bg-white/90 border border-white/25 shadow-2xl rounded-2xl overflow-hidden hover:shadow-xl hover:-translate-y-1 transition-all duration-300"
                    >
                      <Accordion.Header>
                        <Accordion.Trigger className="group w-full px-6 py-4 text-left hover:bg-gray-50 transition-colors flex justify-between items-center">
                          <div className="flex-1">
                            <div className="font-semibold text-gray-900 text-lg">{decision.data.decision?.action || 'No action specified'}</div>
                            <div className="text-sm text-gray-500 mt-1.5 flex items-center gap-2">
                              <span>{new Date(decision.timestamp).toLocaleDateString()}</span>
                              <span className="text-gray-300">•</span>
                              <span>{new Date(decision.timestamp).toLocaleTimeString()}</span>
                            </div>
                          </div>
                          <div className="flex items-center gap-3">
                            {getConfidenceBadge(decision.data.decision?.confidence || 0)}
                            <ChevronDown className="h-5 w-5 text-gray-400 transition-transform group-data-[state=open]:rotate-180" />
                          </div>
                        </Accordion.Trigger>
                      </Accordion.Header>
                      <Accordion.Content className="px-6 py-4 bg-white/50 border-t border-white/30 backdrop-blur-sm">
                        <div className="space-y-4">
                          {decision.data.context?.userRequest && (
                            <div className="backdrop-blur-sm bg-blue-50/80 rounded-xl p-4 border border-blue-200/40 shadow-lg">
                              <h4 className="font-semibold text-gray-900 mb-2 flex items-center gap-2">
                                <div className="h-1.5 w-1.5 rounded-full bg-blue-600"></div>
                                Context
                              </h4>
                              <p className="text-sm text-gray-700 leading-relaxed">{decision.data.context.userRequest}</p>
                            </div>
                          )}
                          {decision.data.decision?.reasoning && (
                            <div className="backdrop-blur-sm bg-emerald-50/80 rounded-xl p-4 border border-emerald-200/40 shadow-lg">
                              <h4 className="font-semibold text-gray-900 mb-2 flex items-center gap-2">
                                <div className="h-1.5 w-1.5 rounded-full bg-green-600"></div>
                                Reasoning
                              </h4>
                              <p className="text-sm text-gray-700 leading-relaxed">{decision.data.decision.reasoning}</p>
                            </div>
                          )}
                          {decision.data.decision?.alternatives && decision.data.decision.alternatives.length > 0 && (
                            <div className="backdrop-blur-sm bg-yellow-50/80 rounded-xl p-4 border border-yellow-200/40 shadow-lg">
                              <h4 className="font-semibold text-gray-900 mb-2 flex items-center gap-2">
                                <div className="h-1.5 w-1.5 rounded-full bg-yellow-600"></div>
                                Alternatives Considered
                              </h4>
                              <ul className="text-sm text-gray-700 space-y-1.5">
                                {decision.data.decision.alternatives.map((alt, i) => (
                                  <li key={i} className="flex items-start gap-2">
                                    <span className="text-gray-400 mt-1">•</span>
                                    <span className="flex-1">{alt}</span>
                                  </li>
                                ))}
                              </ul>
                            </div>
                          )}
                          {decision.data.predictions && (
                            <div className="backdrop-blur-sm bg-purple-50/80 rounded-xl p-4 border border-purple-200/40 shadow-lg">
                              <h4 className="font-semibold text-gray-900 mb-3 flex items-center gap-2">
                                <div className="h-1.5 w-1.5 rounded-full bg-purple-600"></div>
                                Predictions
                              </h4>
                              <div className="grid grid-cols-2 gap-4">
                                {decision.data.predictions.successProbability !== undefined && (
                                  <div className="backdrop-blur-sm bg-white/60 rounded-lg px-3 py-2 border border-white/20">
                                    <div className="text-xs text-gray-500 mb-1">Success Probability</div>
                                    <div className="text-lg font-semibold text-gray-900">
                                      {(decision.data.predictions.successProbability * 100).toFixed(0)}%
                                    </div>
                                  </div>
                                )}
                                {decision.data.predictions.timeEstimate && (
                                  <div className="backdrop-blur-sm bg-white/60 rounded-lg px-3 py-2 border border-white/20">
                                    <div className="text-xs text-gray-500 mb-1">Time Estimate</div>
                                    <div className="text-lg font-semibold text-gray-900">
                                      {decision.data.predictions.timeEstimate}
                                    </div>
                                  </div>
                                )}
                              </div>
                            </div>
                          )}
                        </div>
                      </Accordion.Content>
                    </Accordion.Item>
                    );
                  })}
                </Accordion.Root>
              )}
            </Tabs.Content>

            {/* Lessons Tab */}
            <Tabs.Content value="lessons" className="p-6">
              {lessons.length === 0 ? (
                <div className="text-center py-16 text-gray-500">
                  <Brain className="h-16 w-16 mx-auto mb-4 opacity-30" />
                  <p className="text-lg font-medium">No lessons extracted yet</p>
                  <p className="text-sm mt-2">Lessons will appear here as they are extracted from experiences</p>
                </div>
              ) : (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  {lessons.map((lesson) => (
                    <div key={lesson.id} className="backdrop-blur-sm bg-white/90 border border-white/25 shadow-2xl rounded-2xl p-6 space-y-4 hover:shadow-xl hover:-translate-y-1 transition-all duration-300">
                      <div className="flex justify-between items-start gap-4">
                        <h3 className="font-bold text-lg text-gray-900">{lesson.data.patternName}</h3>
                        {lesson.confidence && getConfidenceBadge(lesson.confidence)}
                      </div>

                      <div className="space-y-3">
                        <div className="backdrop-blur-sm bg-red-50/80 border-l-4 border-red-500 rounded-xl p-3 shadow-lg">
                          <h4 className="text-sm font-semibold text-red-900 mb-1.5 flex items-center gap-2">
                            <AlertCircle className="h-4 w-4" />
                            Problem
                          </h4>
                          <p className="text-sm text-red-800 leading-relaxed">{lesson.data.problem}</p>
                        </div>

                        <div className="backdrop-blur-sm bg-emerald-50/80 border-l-4 border-green-500 rounded-xl p-3 shadow-lg">
                          <h4 className="text-sm font-semibold text-green-900 mb-1.5 flex items-center gap-2">
                            <CheckCircle className="h-4 w-4" />
                            Solution
                          </h4>
                          <p className="text-sm text-green-800 leading-relaxed">{lesson.data.solution}</p>
                        </div>

                        {lesson.data.antipattern && (
                          <div className="backdrop-blur-sm bg-orange-50/80 border-l-4 border-orange-500 rounded-xl p-3 shadow-lg">
                            <h4 className="text-sm font-semibold text-orange-900 mb-1.5 flex items-center gap-2">
                              <XCircle className="h-4 w-4" />
                              Anti-pattern (What NOT to do)
                            </h4>
                            <p className="text-sm text-orange-800 leading-relaxed">{lesson.data.antipattern}</p>
                          </div>
                        )}

                        {lesson.data.context && (
                          <div className="backdrop-blur-sm bg-blue-50/80 border-l-4 border-blue-500 rounded-xl p-3 shadow-lg">
                            <h4 className="text-sm font-semibold text-blue-900 mb-1.5">Context</h4>
                            <p className="text-sm text-blue-800 leading-relaxed">{lesson.data.context}</p>
                          </div>
                        )}
                      </div>

                      {lesson.data.applicableTo && lesson.data.applicableTo.length > 0 && (
                        <div className="pt-3 border-t border-gray-200">
                          <div className="flex flex-wrap gap-2">
                            {lesson.data.applicableTo.map((tag, i) => (
                              <Badge key={i} variant="outline" className="bg-gray-50">{tag}</Badge>
                            ))}
                          </div>
                        </div>
                      )}

                      <div className="text-xs text-gray-500 pt-2 border-t border-gray-100">
                        {new Date(lesson.timestamp).toLocaleString()}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </Tabs.Content>

            {/* Query Tab */}
            <Tabs.Content value="query" className="p-6">
              <div className="space-y-6">
                <div className="backdrop-blur-sm bg-gradient-to-br from-blue-50/50 to-purple-50/50 border border-blue-200/30 rounded-2xl p-6 shadow-xl">
                  <h3 className="text-xl font-bold mb-3 flex items-center gap-2 text-gray-900">
                    <Search className="h-6 w-6 text-blue-600" />
                    Query Relevant Lessons
                  </h3>
                  <p className="text-sm text-gray-700 mb-5 leading-relaxed">
                    Describe the current situation, decision, or action you're about to take. The system will find relevant lessons from past experiences to guide your decision.
                  </p>
                  <div className="flex gap-3">
                    <Input
                      placeholder="e.g., 'about to modify database schema', 'implementing authentication'..."
                      value={queryText}
                      onChange={(e) => setQueryText(e.target.value)}
                      onKeyPress={(e) => e.key === 'Enter' && handleQuery()}
                      className="flex-1 bg-white border-gray-300 shadow-sm"
                    />
                    <Button
                      onClick={handleQuery}
                      disabled={loading || !queryText.trim()}
                      className="px-6 bg-blue-600 hover:bg-blue-700 text-white shadow-sm"
                    >
                      {loading ? (
                        <span className="flex items-center gap-2">
                          <div className="h-4 w-4 animate-spin rounded-full border-2 border-solid border-white border-r-transparent"></div>
                          Searching...
                        </span>
                      ) : (
                        'Query'
                      )}
                    </Button>
                  </div>
                </div>

                {queryResults.length > 0 && (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <h4 className="text-lg font-semibold text-gray-900">
                        Found {queryResults.length} Relevant {queryResults.length === 1 ? 'Lesson' : 'Lessons'}
                      </h4>
                    </div>
                    <div className="space-y-4">
                      {queryResults.map((lesson) => (
                        <div key={lesson.id} className="backdrop-blur-sm bg-white/90 border border-white/25 shadow-2xl rounded-2xl p-6 space-y-4 hover:shadow-xl hover:-translate-y-1 transition-all duration-300">
                          <div className="flex justify-between items-start gap-4">
                            <h3 className="font-bold text-lg text-gray-900">{lesson.data.patternName}</h3>
                            <div className="flex items-center gap-2 flex-shrink-0">
                              {lesson.confidence && getConfidenceBadge(lesson.confidence)}
                            </div>
                          </div>

                          <div className="space-y-3">
                            <div className="backdrop-blur-sm bg-red-50/80 border-l-4 border-red-500 rounded-xl p-4 shadow-lg">
                              <h4 className="text-sm font-semibold text-red-900 mb-2 flex items-center gap-2">
                                <AlertCircle className="h-4 w-4" />
                                Problem
                              </h4>
                              <p className="text-sm text-red-800 leading-relaxed">{lesson.data.problem}</p>
                            </div>

                            <div className="backdrop-blur-sm bg-emerald-50/80 border-l-4 border-green-500 rounded-xl p-4 shadow-lg">
                              <h4 className="text-sm font-semibold text-green-900 mb-2 flex items-center gap-2">
                                <CheckCircle className="h-4 w-4" />
                                Solution
                              </h4>
                              <p className="text-sm text-green-800 leading-relaxed">{lesson.data.solution}</p>
                            </div>

                            {lesson.data.antipattern && (
                              <div className="backdrop-blur-sm bg-orange-50/80 border-l-4 border-orange-500 rounded-xl p-4 shadow-lg">
                                <h4 className="text-sm font-semibold text-orange-900 mb-2 flex items-center gap-2">
                                  <XCircle className="h-4 w-4" />
                                  What NOT to do
                                </h4>
                                <p className="text-sm text-orange-800 leading-relaxed">{lesson.data.antipattern}</p>
                              </div>
                            )}

                            {lesson.data.context && (
                              <div className="backdrop-blur-sm bg-blue-50/80 border-l-4 border-blue-500 rounded-xl p-4 shadow-lg">
                                <h4 className="text-sm font-semibold text-blue-900 mb-2">Context</h4>
                                <p className="text-sm text-blue-800 leading-relaxed">{lesson.data.context}</p>
                              </div>
                            )}
                          </div>

                          {lesson.data.applicableTo && lesson.data.applicableTo.length > 0 && (
                            <div className="pt-3 border-t border-gray-200">
                              <div className="flex flex-wrap gap-2">
                                {lesson.data.applicableTo.map((tag, i) => (
                                  <Badge key={i} variant="outline" className="bg-gray-50">{tag}</Badge>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {!loading && queryText && queryResults.length === 0 && (
                  <div className="text-center py-12 backdrop-blur-sm bg-white/60 rounded-2xl border border-white/20 shadow-xl">
                    <Search className="h-12 w-12 mx-auto mb-3 text-gray-400" />
                    <p className="text-gray-600 font-medium">No matching lessons found</p>
                    <p className="text-sm text-gray-500 mt-2">Try a different query or add more details</p>
                  </div>
                )}
              </div>
            </Tabs.Content>
          </Tabs.Root>
        </div>
      </div>
    </div>
  );
}
