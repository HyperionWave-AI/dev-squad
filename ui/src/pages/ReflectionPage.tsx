import { useState, useEffect } from 'react';
import { reflectionService } from '../services/reflectionService';
import type { Decision, Outcome, Lesson } from '../types/reflection';

type TabType = 'lessons' | 'decisions' | 'outcomes';

export default function ReflectionPage() {
  const [activeTab, setActiveTab] = useState<TabType>('lessons');
  const [lessons, setLessons] = useState<Lesson[]>([]);
  const [decisions, setDecisions] = useState<Decision[]>([]);
  const [outcomes, setOutcomes] = useState<Outcome[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [patternFilter, setPatternFilter] = useState('');

  useEffect(() => {
    loadData();
  }, [activeTab, patternFilter]);

  async function loadData() {
    setLoading(true);
    setError(null);
    try {
      if (activeTab === 'lessons') {
        const response = await reflectionService.listLessons({
          pattern: patternFilter || undefined,
          limit: 50
        });
        setLessons(response.lessons || []);
      } else if (activeTab === 'decisions') {
        const response = await reflectionService.listDecisions({ limit: 50 });
        setDecisions(response.decisions || []);
      } else if (activeTab === 'outcomes') {
        const response = await reflectionService.listOutcomes({ limit: 50 });
        setOutcomes(response.outcomes || []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load reflections');
    } finally {
      setLoading(false);
    }
  }

  function getConfidenceColor(confidence?: number): string {
    if (!confidence) return 'bg-gray-500';
    if (confidence >= 0.9) return 'bg-green-600';
    if (confidence >= 0.7) return 'bg-blue-500';
    if (confidence >= 0.5) return 'bg-yellow-500';
    return 'bg-orange-500';
  }

  function getCalibrationColor(calibration?: string): string {
    if (calibration === 'well-calibrated') return 'text-green-600';
    if (calibration === 'overconfident') return 'text-orange-600';
    if (calibration === 'underconfident') return 'text-blue-600';
    return 'text-gray-600';
  }

  return (
    <div className="h-full flex flex-col bg-gray-50 dark:bg-gray-900">
      {/* Header */}
      <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 p-6">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-gray-100 flex items-center gap-3">
          <span className="text-4xl">🧠</span>
          Metacognitive Self-Awareness
        </h1>
        <p className="text-gray-600 dark:text-gray-400 mt-2">
          System learning and decision tracking - Building wisdom from experience
        </p>
      </div>

      {/* Tabs */}
      <div className="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
        <div className="flex gap-1 px-6">
          {[
            { id: 'lessons' as const, label: 'Lessons', icon: '📚', count: lessons.length },
            { id: 'decisions' as const, label: 'Decisions', icon: '🎯', count: decisions.length },
            { id: 'outcomes' as const, label: 'Outcomes', icon: '📊', count: outcomes.length }
          ].map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`px-6 py-3 font-medium transition-colors border-b-2 ${
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                  : 'border-transparent text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'
              }`}
            >
              <span className="mr-2">{tab.icon}</span>
              {tab.label}
              {tab.count > 0 && (
                <span className="ml-2 px-2 py-0.5 text-xs rounded-full bg-gray-200 dark:bg-gray-700">
                  {tab.count}
                </span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        {error && (
          <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-700 dark:text-red-400">
            {error}
          </div>
        )}

        {activeTab === 'lessons' && (
          <div className="space-y-4">
            {/* Filter */}
            <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
              <input
                type="text"
                placeholder="Filter by pattern name..."
                value={patternFilter}
                onChange={(e) => setPatternFilter(e.target.value)}
                className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100"
              />
            </div>

            {loading ? (
              <div className="text-center py-12 text-gray-500">Loading lessons...</div>
            ) : lessons.length === 0 ? (
              <div className="text-center py-12 text-gray-500">No lessons found</div>
            ) : (
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                {lessons.map(lesson => (
                  <div
                    key={lesson.id}
                    className="bg-white dark:bg-gray-800 rounded-lg p-6 border border-gray-200 dark:border-gray-700 hover:border-blue-500 dark:hover:border-blue-400 transition-colors"
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex-1">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">
                          {lesson.data.patternName}
                        </h3>
                        {lesson.data.context && (
                          <p className="text-sm text-gray-500 dark:text-gray-400">{lesson.data.context}</p>
                        )}
                      </div>
                      {lesson.confidence && (
                        <div className="flex items-center gap-2">
                          <span className={`w-3 h-3 rounded-full ${getConfidenceColor(lesson.confidence)}`} />
                          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                            {(lesson.confidence * 100).toFixed(0)}%
                          </span>
                        </div>
                      )}
                    </div>

                    <div className="space-y-2 text-sm">
                      <div>
                        <span className="font-medium text-red-600 dark:text-red-400">Problem:</span>{' '}
                        <span className="text-gray-700 dark:text-gray-300">{lesson.data.problem}</span>
                      </div>
                      <div>
                        <span className="font-medium text-green-600 dark:text-green-400">Solution:</span>{' '}
                        <span className="text-gray-700 dark:text-gray-300">{lesson.data.solution}</span>
                      </div>
                      {lesson.data.antipattern && (
                        <div>
                          <span className="font-medium text-orange-600 dark:text-orange-400">Anti-pattern:</span>{' '}
                          <span className="text-gray-700 dark:text-gray-300">{lesson.data.antipattern}</span>
                        </div>
                      )}
                    </div>

                    {lesson.data.applicableTo && lesson.data.applicableTo.length > 0 && (
                      <div className="mt-3 flex flex-wrap gap-2">
                        {lesson.data.applicableTo.map((tag, idx) => (
                          <span
                            key={idx}
                            className="px-2 py-1 text-xs rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300"
                          >
                            {tag}
                          </span>
                        ))}
                      </div>
                    )}

                    <div className="mt-3 text-xs text-gray-400">
                      {new Date(lesson.timestamp).toLocaleString()}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === 'decisions' && (
          <div className="space-y-4">
            {loading ? (
              <div className="text-center py-12 text-gray-500">Loading decisions...</div>
            ) : decisions.length === 0 ? (
              <div className="text-center py-12 text-gray-500">No decisions recorded</div>
            ) : (
              <div className="space-y-4">
                {decisions.map(decision => (
                  <div
                    key={decision.id}
                    className="bg-white dark:bg-gray-800 rounded-lg p-6 border border-gray-200 dark:border-gray-700 hover:border-blue-500 transition-colors"
                  >
                    <div className="flex items-start justify-between mb-4">
                      <div className="flex-1">
                        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
                          {decision.data.decision.action}
                        </h3>
                        <p className="text-sm text-gray-600 dark:text-gray-400">
                          {decision.data.context.userRequest}
                        </p>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className={`w-3 h-3 rounded-full ${getConfidenceColor(decision.data.decision.confidence)}`} />
                        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                          {(decision.data.decision.confidence * 100).toFixed(0)}%
                        </span>
                      </div>
                    </div>

                    <div className="grid md:grid-cols-2 gap-4 text-sm">
                      <div>
                        <div className="font-medium text-gray-700 dark:text-gray-300 mb-2">Reasoning:</div>
                        <p className="text-gray-600 dark:text-gray-400">{decision.data.decision.reasoning}</p>
                      </div>
                      {decision.data.decision.alternatives && decision.data.decision.alternatives.length > 0 && (
                        <div>
                          <div className="font-medium text-gray-700 dark:text-gray-300 mb-2">Alternatives Considered:</div>
                          <ul className="list-disc list-inside text-gray-600 dark:text-gray-400">
                            {decision.data.decision.alternatives.map((alt, idx) => (
                              <li key={idx}>{alt}</li>
                            ))}
                          </ul>
                        </div>
                      )}
                    </div>

                    {decision.data.predictions && (
                      <div className="mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg text-sm">
                        <div className="font-medium text-blue-700 dark:text-blue-300 mb-2">Predictions:</div>
                        {decision.data.predictions.successProbability && (
                          <div>Success: {(decision.data.predictions.successProbability * 100).toFixed(0)}%</div>
                        )}
                        {decision.data.predictions.timeEstimate && (
                          <div>Time: {decision.data.predictions.timeEstimate}</div>
                        )}
                      </div>
                    )}

                    <div className="mt-3 text-xs text-gray-400">
                      {new Date(decision.timestamp).toLocaleString()}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {activeTab === 'outcomes' && (
          <div className="space-y-4">
            {loading ? (
              <div className="text-center py-12 text-gray-500">Loading outcomes...</div>
            ) : outcomes.length === 0 ? (
              <div className="text-center py-12 text-gray-500">No outcomes recorded</div>
            ) : (
              <div className="space-y-4">
                {outcomes.map(outcome => (
                  <div
                    key={outcome.id}
                    className="bg-white dark:bg-gray-800 rounded-lg p-6 border border-gray-200 dark:border-gray-700"
                  >
                    <div className="flex items-start justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <span className="text-2xl">
                          {outcome.data.outcome.success ? '✅' : '❌'}
                        </span>
                        <div>
                          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
                            {outcome.data.outcome.success ? 'Success' : 'Failed'}
                          </h3>
                          <p className="text-sm text-gray-600 dark:text-gray-400">
                            Decision: {outcome.data.decisionId.substring(0, 8)}...
                          </p>
                        </div>
                      </div>
                      {outcome.data.analysis.confidenceCalibration && (
                        <span className={`font-medium ${getCalibrationColor(outcome.data.analysis.confidenceCalibration)}`}>
                          {outcome.data.analysis.confidenceCalibration}
                        </span>
                      )}
                    </div>

                    <div className="space-y-3 text-sm">
                      <div>
                        <div className="font-medium text-gray-700 dark:text-gray-300 mb-1">Result:</div>
                        <p className="text-gray-600 dark:text-gray-400">{outcome.data.outcome.actualResult}</p>
                      </div>

                      {outcome.data.analysis.predictionAccuracy !== undefined && (
                        <div>
                          <div className="font-medium text-gray-700 dark:text-gray-300 mb-1">Prediction Accuracy:</div>
                          <div className="flex items-center gap-2">
                            <div className="flex-1 bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                              <div
                                className="bg-green-500 h-2 rounded-full"
                                style={{ width: `${outcome.data.analysis.predictionAccuracy * 100}%` }}
                              />
                            </div>
                            <span className="text-gray-700 dark:text-gray-300">
                              {(outcome.data.analysis.predictionAccuracy * 100).toFixed(0)}%
                            </span>
                          </div>
                        </div>
                      )}

                      {outcome.data.analysis.missedSignals && outcome.data.analysis.missedSignals.length > 0 && (
                        <div>
                          <div className="font-medium text-gray-700 dark:text-gray-300 mb-1">Missed Signals:</div>
                          <ul className="list-disc list-inside text-gray-600 dark:text-gray-400">
                            {outcome.data.analysis.missedSignals.map((signal, idx) => (
                              <li key={idx}>{signal}</li>
                            ))}
                          </ul>
                        </div>
                      )}
                    </div>

                    <div className="mt-3 text-xs text-gray-400">
                      {new Date(outcome.timestamp).toLocaleString()}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Stats Footer */}
      <div className="bg-white dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 p-4">
        <div className="flex justify-around text-center">
          <div>
            <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">{lessons.length}</div>
            <div className="text-xs text-gray-500">Lessons Learned</div>
          </div>
          <div>
            <div className="text-2xl font-bold text-green-600 dark:text-green-400">{decisions.length}</div>
            <div className="text-xs text-gray-500">Decisions Tracked</div>
          </div>
          <div>
            <div className="text-2xl font-bold text-purple-600 dark:text-purple-400">{outcomes.length}</div>
            <div className="text-xs text-gray-500">Outcomes Recorded</div>
          </div>
        </div>
      </div>
    </div>
  );
}
