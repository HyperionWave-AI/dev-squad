// Reflection types for metacognitive self-awareness system

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

export interface ListResponse<T> {
  count: number;
  decisions?: T[];
  outcomes?: T[];
  lessons?: T[];
}

export interface CreateDecisionRequest {
  chatId: string;
  taskId?: string;
  context: Decision['data']['context'];
  decision: Decision['data']['decision'];
  predictions?: Decision['data']['predictions'];
}

export interface CreateOutcomeRequest {
  decisionId: string;
  outcome: Outcome['data']['outcome'];
  analysis: Outcome['data']['analysis'];
}

export interface CreateLessonRequest {
  patternName: string;
  context?: string;
  problem: string;
  solution: string;
  antipattern?: string;
  applicableTo?: string[];
  confidence?: number;
}
