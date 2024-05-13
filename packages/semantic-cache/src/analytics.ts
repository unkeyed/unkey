import { Tinybird } from "@chronark/zod-bird";
import { z } from "zod";

export class Analytics {
  public readonly client: Tinybird;

  constructor(opts: {
    tinybirdToken: string;
  }) {
    this.client = new Tinybird({ token: opts.tinybirdToken });
  }

  public get ingestLogs() {
    return this.client.buildIngestEndpoint({
      datasource: "semantic_cache__v2",
      event: z.object({
        timestamp: z.number(),
        model: z.string(),
        stream: z.number(),
        query: z.string().optional(),
        vector: z.array(z.number()).optional(),
        response: z.string().optional(),
        cache: z.number().optional(),
        timing: z.number().optional(),
        tokens: z.number().optional(),
        requestId: z.string().optional(),
      }),
    });
  }
}
