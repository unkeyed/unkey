import { Metric, Metrics } from "./interface";

export class NoopMetrics implements Metrics {
  public emit<TMetric extends keyof Metric>(_metric: TMetric, _e: Metric[TMetric]): Promise<void> {
    return Promise.resolve();
  }

  public async flush(): Promise<void> {}
}
