/**
 * Base interface for all evaluation metrics
 */
export interface Metric<TInput = any, TOutput = any> {
  /**
   * Unique identifier for the metric
   */
  id: string;
  
  /**
   * Human-readable name for the metric
   */
  name: string;
  
  /**
   * Description of what the metric evaluates
   */
  description: string;
  
  /**
   * Evaluate the input and return a score and explanation
   */
  evaluate: (input: TInput) => Promise<MetricResult<TOutput>>;
}

/**
 * Result of a metric evaluation
 */
export interface MetricResult<TOutput = any> {
  /**
   * Score between 0 and 1, where 1 is the best possible score
   */
  score: number;
  
  /**
   * Whether the evaluation passed (typically score >= threshold)
   */
  passed: boolean;
  
  /**
   * Explanation of the score
   */
  reason: string;
  
  /**
   * Suggestions for improvement
   */
  suggestions?: string[];
  
  /**
   * Additional output data specific to the metric
   */
  output?: TOutput;
}

/**
 * Input for SEO metrics
 */
export interface SEOMetricInput {
  /**
   * The content to evaluate
   */
  content: string;
  
  /**
   * The primary keyword or term
   */
  primaryKeyword: string;
  
  /**
   * Secondary keywords or related terms
   */
  secondaryKeywords?: string[];
  
  /**
   * Additional context or metadata
   */
  context?: Record<string, any>;
}

/**
 * Combined results of multiple metric evaluations
 */
export interface EvaluationResult {
  /**
   * Overall score (average of all metrics)
   */
  score: number;
  
  /**
   * Whether all metrics passed
   */
  passed: boolean;
  
  /**
   * Total number of evaluations
   */
  total: number;
  
  /**
   * Number of passed evaluations
   */
  passedCount: number;
  
  /**
   * Number of failed evaluations
   */
  failedCount: number;
  
  /**
   * Duration of the evaluation in milliseconds
   */
  durationMs: number;
  
  /**
   * Results for each metric
   */
  results: Record<string, MetricResult>;
  
  /**
   * All suggestions combined
   */
  suggestions: string[];
} 