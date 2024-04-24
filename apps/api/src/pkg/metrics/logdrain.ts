import { Log } from "@unkey/logs";
import type { Metric } from "@unkey/metrics";
import type { Metrics } from "./interface";

export class LogdrainMetrics implements Metrics {
  public emit(metric: Metric): void {
    const log = new Log<{ type: "metric"; time: number; metric: Metric }>({
      type: "metric",
      time: Date.now(),
      metric,
    });

    console.info(log.toString());
  }

  public async flush(): Promise<void> {
    return Promise.resolve();
  }
}
