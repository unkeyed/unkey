import { Metrics, Metric } from "./interface"



export class NoopMetrics implements Metrics {




  public emit<TMetric extends keyof Metric>(metric: TMetric, e: Metric[TMetric]): Promise<void> {
    return Promise.resolve()
  }

  public async flush(): Promise<void> { }

}
