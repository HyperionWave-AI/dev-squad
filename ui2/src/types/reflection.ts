export interface Decision {
  id: string;
  chatId: string;
  taskId?: string;
  timestamp: string;
  context: {
    userRequest: string;
    availableInfo: string;
    uncertainty: string;
  };
  decision: {
    action: string;
    reasoning: string;
    alternatives: string[];
    confidence: number;
  };
  predictions?: {
    successProbability: number;
    timeEstimate: string;
    risks: string[];
  };
}

export interface Outcome {
  id: string;
  decisionId: string;
  timestamp: string;
  outcome: {
    success: boolean;
    actualResult: string;
    rootCause?: string;
    userFeedback?: string;
  };
  analysis: {
    predictionAccuracy: number;
    confidenceCalibration: 'overconfident' | 'underconfident' | 'well-calibrated';
    missedSignals: string[];
  };
}

export interface Lesson {
  id: string;
  patternName: string;
  timestamp: string;
  problem: string;
  solution: string;
  context: string;
  antipattern?: string;
  applicableTo?: string[];
  confidence: number;
}

export interface QueryResult {
  lesson: Lesson;
  score: number;
  relevance: string;
}

export interface DecisionWithOutcome extends Decision {
  outcome?: Outcome;
}
