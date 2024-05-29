import { Log } from "@unkey/logs";
import type { Metric } from "@unkey/metrics";
import type { Metrics } from "./interface";

export class LogdrainMetrics implements Metrics {
  private readonly requestId: string;

  constructor(opts: { requestId: string }) {
    this.requestId = opts.requestId;
  }

  public emit(metric: Metric): void {
    const log = new Log<{ type: "metric"; time: number; metric: Metric }>(
      { requestId: this.requestId },
      {
        type: "metric",
        time: Date.now(),
        metric,
      },
    );

    console.info(log.toString());
  }

  public async flush(): Promise<void> {
    return Promise.resolve();
  }
}
