/**
 * Implement this interface to send metrics to any sink of your choice
 */
export interface Metrics<TMetric extends Record<string, unknown> = Record<string, unknown>> {
  /**
   * Emit  a new metric event
   *
   */
  emit(metric: TMetric): void;

  /**
   * flush persists all metrics to durable storage
   */
  flush(): Promise<void>;
}
