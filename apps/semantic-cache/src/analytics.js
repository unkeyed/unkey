import { Tinybird } from "@chronark/zod-bird";
import { z } from "zod";
export class Analytics {
  client;
  constructor(opts) {
    this.client = new Tinybird({ token: opts.tinybirdToken });
  }
  get ingestLogs() {
    return this.client.buildIngestEndpoint({
      datasource: "semantic_cache__v3",
      event: z.object({
        timestamp: z.string(),
        model: z.string(),
        stream: z.boolean(),
        query: z.string(),
        vector: z.array(z.number()),
        response: z.string(),
        cache: z.boolean(),
        timing: z.number(),
        tokens: z.number(),
        requestId: z.string(),
      }),
    });
  }
}
