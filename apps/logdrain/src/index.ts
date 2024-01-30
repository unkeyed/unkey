import { Axiom } from "@axiomhq/js";
import type { MessageBatch, Queue } from "@cloudflare/workers-types";
import { Metric, metricSchema } from "@unkey/metrics";
import { BufferQueue } from "@unkey/zod-queue";
import { z } from "zod";
export interface Env {
  // Example binding to a Queue. Learn more at https://developers.cloudflare.com/queues/javascript-apis/
  METRICS: Queue;

  AXIOM_TOKEN: string;
}

export default {
  async queue(batch: MessageBatch<unknown>, env: Env): Promise<void> {
    switch (batch.queue) {
      case "metrics-development":
      case "metrics-preview":
      case "metrics-production":
        return await flushMetrics(batch, env);

      case "logs-development":
      case "logs-preview":
      case "logs-production":
        return await flushLogs(batch, env);

      default:
        console.error("Unhandled queue", batch.queue);
    }
  },
};

async function flushMetrics(batch: MessageBatch<unknown>, env: Env) {
  const q = new BufferQueue({ queue: env.METRICS, schema: metricSchema });
  const { messages } = q.unpack(batch as MessageBatch<Metric[]>);

  const queueToDataset: Record<string, string> = {
    "metrics-development": "cf_api_metrics_development",
    "metrics-preview": "cf_api_metrics_preview",
    "metrics-production": "cf_api_metrics_production",
  };
  const dataset = queueToDataset[batch.queue];
  if (!dataset) {
    throw new Error(`I don't know where to put this: ${batch.queue}`);
  }

  const ax = new Axiom({ token: env.AXIOM_TOKEN });

  ax.ingest(dataset, messages);
  await ax
    .flush()
    .then(() => {
      console.log(`Ingested ${messages.length} into ${dataset} on axiom`);
      batch.ackAll();
    })
    .catch((err) => {
      console.error(err);
      batch.retryAll();
    });
}

async function flushLogs(batch: MessageBatch<unknown>, env: Env) {
  const schema = z
    .object({
      message: z.string(),
      level: z.enum(["debug", "info", "warn", "error"]),
    })
    .passthrough();

  const q = new BufferQueue({ queue: env.METRICS, schema });
  const { messages } = q.unpack(batch as MessageBatch<z.infer<typeof schema>[]>);

  const queueToDataset: Record<string, string> = {
    "logs-development": "cf_api_logs_development",
    "logs-preview": "cf_api_logs_preview",
    "logs-production": "cf_api_logs_production",
  };
  const dataset = queueToDataset[batch.queue];
  if (!dataset) {
    throw new Error(`I don't know where to put this: ${batch.queue}`);
  }

  const ax = new Axiom({ token: env.AXIOM_TOKEN });

  ax.ingest(dataset, messages);
  await ax
    .flush()
    .then(() => {
      console.log(`Ingested ${messages.length} into ${dataset} on axiom`);
      batch.ackAll();
    })
    .catch((err) => {
      console.error(err);
      batch.retryAll();
    });
}
