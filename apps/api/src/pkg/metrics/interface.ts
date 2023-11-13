export type Metric = {
  "metric.cache.read": {
    key: string;
    hit: boolean;
    latency: number;
    tier: string;
    resource: string;
  };
  "metric.cache.write": {
    key: string;
    tier: string;
    resource: string;
  };
  "metric.cache.purge": {
    key: string;
    tier: string;
    resource: string;
  };
  "metric.http.request": {
    path: string;
    method: string;
    status: number;
    error?: string;
    serviceLatency: number;
    requestId: string;
    // Regional data might be different on non-cloudflare deployments
    colo?: string;
    continent?: string;
    country?: string;
    city?: string;
    userAgent?: string;
  };
  "metric.db.read": {
    query: "getKeyAndApiByHash";
    latency: number;
  };
  "metric.ratelimit": {
    keyId: string;
    latency: number;
    tier: "memory" | "durable" | "total";
  };
  "metric.usagelimit": {
    keyId: string;
    latency: number;
  }
};

export interface Metrics {
  /**
   * Emit stores a new metric event
   */
  emit<TMetric extends keyof Metric>(metric: TMetric, e: Metric[TMetric]): void;

  /**
   * flush persists all metrics to durable storage
   */
  flush(): Promise<void>;
}
