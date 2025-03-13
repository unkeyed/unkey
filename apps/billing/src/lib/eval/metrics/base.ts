import type { Metric, MetricResult } from "../types";

/**
 * Abstract base class for all metrics
 */
export abstract class BaseMetric<TInput = any, TOutput = any> implements Metric<TInput, TOutput> {
  /**
   * Unique identifier for the metric
   */
  abstract id: string;

  /**
   * Human-readable name for the metric
   */
  abstract name: string;

  /**
   * Description of what the metric evaluates
   */
  abstract description: string;

  /**
   * Default threshold for passing
   */
  protected threshold = 0.7;

  /**
   * Evaluate the input and return a score and explanation
   */
  abstract evaluate(input: TInput): Promise<MetricResult<TOutput>>;

  /**
   * Determine if a score passes the threshold
   */
  protected isPassing(score: number): boolean {
    return score >= this.threshold;
  }

  /**
   * Create a result object
   */
  protected createResult(
    score: number,
    reason: string,
    suggestions: string[] = [],
    output?: TOutput,
  ): MetricResult<TOutput> {
    return {
      score,
      passed: this.isPassing(score),
      reason,
      suggestions,
      output,
    };
  }
}
