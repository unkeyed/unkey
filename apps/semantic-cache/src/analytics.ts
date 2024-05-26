import { Tinybird } from "@chronark/zod-bird";
import { z } from "zod";

export const eventSchema = z.object({
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
});

export class Analytics {
  public readonly client: Tinybird;

  constructor(opts: {
    tinybirdToken: string;
  }) {
    this.client = new Tinybird({ token: opts.tinybirdToken });
  }

  public get ingestLogs() {
    return this.client.buildIngestEndpoint({
      datasource: "semantic_cache__v3",
      event: eventSchema,
    });
  }
}
