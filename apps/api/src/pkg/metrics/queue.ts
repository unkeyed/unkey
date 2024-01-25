import { Env } from "../env";
import { Metric, Metrics } from "./interface";
export class QueueMetrics implements Metrics {
  private readonly defaultFields: Record<string, unknown>;
  private readonly drain: Env["METRICS"];
  private promises: Record<string, Promise<void>> = {};

  constructor(opts: {
    drain: Env["METRICS"];
    defaultFields?: Record<string, unknown>;
  }) {
    this.defaultFields = opts.defaultFields ?? {};
    this.drain = opts.drain;
  }

  public emit<TMetric extends keyof Metric>(metric: TMetric, e: Metric[TMetric]): void {
    const p = this.drain.send({
      _time: Date.now(),
      ...this.defaultFields,
      metric,
      ...e,
    });

    this.promises[crypto.randomUUID()] = p;
  }

  /**
   * flush sends the metrics to axiom
   *
   * Call this at the end of the request handler with .waitUntil()
   */

  public async flush(): Promise<void> {
    for (const id of Object.keys(this.promises)) {
      await this.promises[id];
      delete this.promises[id];
    }
  }
}
