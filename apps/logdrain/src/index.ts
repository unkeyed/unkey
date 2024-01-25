import { Axiom } from "@axiomhq/js";

export interface Env {
  // Example binding to a Queue. Learn more at https://developers.cloudflare.com/queues/javascript-apis/
  METRICS: Queue;

  AXIOM_TOKEN: string;
}

type Metric = unknown;

export default {
  // The queue handler is invoked when a batch of messages is ready to be delivered
  // https://developers.cloudflare.com/queues/platform/javascript-apis/#messagebatch
  async queue(batch: MessageBatch<Metric>, env: Env): Promise<void> {
    const ax = new Axiom({ token: env.AXIOM_TOKEN });
    const dataset = queueToDataset[batch.queue];
    if (!dataset) {
      throw new Error(`I don't know where to put this: ${batch.queue}`);
    }

    console.log(`Ingesting ${batch.messages.length} into ${dataset}`);

    ax.ingest(dataset, batch.messages);
    await ax.flush();
    batch.ackAll();
  },
};

const queueToDataset: Record<string, string> = {
  "metrics-development": "cf_api_metrics_development",
  "metrics-preview": "cf_api_metrics_preview",
  "metrics-production": "cf_api_metrics_production",
};
