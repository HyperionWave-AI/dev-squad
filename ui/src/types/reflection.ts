// Base reflection type
export interface Reflection {
  id: string;
  type: 'decision' | 'outcome' | 'lesson';
  chatId?: string;
  taskId?: string;
  timestamp: string;
  data: Record<string, any>;
  confidence?: number;
  tags: string[];
  relatedReflections?: string[];
}

export interface Decision extends Reflection {
  type: 'decision';
  data: {
    context: {
      userRequest: string;
      availableInfo: string;
      uncertainty?: string;
    };
    decision: {
      action: string;
      reasoning: string;
      alternatives?: string[];
      confidence: number;
    };
    predictions?: {
      successProbability?: number;
      timeEstimate?: string;
      risks?: string[];
    };
  };
  // Convenience accessors for backwards compatibility
  context?: {
    userRequest: string;
    availableInfo: string;
    uncertainty?: string;
  };
  decision?: {
    action: string;
    reasoning: string;
    alternatives?: string[];
    confidence: number;
  };
  predictions?: {
    successProbability?: number;
    timeEstimate?: string;
    risks?: string[];
  };
}

export interface Outcome extends Reflection {
  type: 'outcome';
  data: {
    decisionId: string;
    outcome: {
      success: boolean;
      actualResult: string;
      userFeedback?: string;
      rootCause?: string;
    };
    analysis: {
      predictionAccuracy?: number;
      missedSignals?: string[];
      confidenceCalibration?: 'overconfident' | 'underconfident' | 'well-calibrated';
    };
  };
}

export interface Lesson extends Reflection {
  type: 'lesson';
  data: {
    patternName: string;
    context?: string;
    problem: string;
    solution: string;
    antipattern?: string;
    applicableTo?: string[];
  };
}

export interface QueryResult {
  lesson: Lesson;
  score: number;
  relevance: string;
}

export interface ListResponse<T> {
  count: number;
  decisions?: T[];
  outcomes?: T[];
  lessons?: T[];
}

export interface DecisionWithOutcome extends Decision {
  outcome?: Outcome;
}
