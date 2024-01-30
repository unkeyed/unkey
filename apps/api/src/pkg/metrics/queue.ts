import { Queue } from "@cloudflare/workers-types";
import { type Metric, metricSchema } from "@unkey/metrics";
import { BufferQueue } from "@unkey/zod-queue";
import type { Metrics } from "./interface";

export class QueueMetrics implements Metrics {
  private readonly queue: BufferQueue<typeof metricSchema>;

  constructor(opts: {
    queue: Queue<Metric[]>;
  }) {
    this.queue = new BufferQueue({
      queue: opts.queue,
      queueSendOptions: { contentType: "json" },
      schema: metricSchema,
    });
  }

  public emit(metric: Metric): void {
    this.queue.buffer(metric);
  }

  /**
   * Call this at the end of the request handler with .waitUntil()
   */

  public async flush(): Promise<void> {
    await this.queue.flush();
  }
}
