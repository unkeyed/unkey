import { Tinybird } from "@chronark/zod-bird";
import { z } from "zod";

export const eventSchema = z.object({
  timestamp: z.string(),
  model: z.string(), // gpt-3.5-turbo
  stream: z.boolean(),
  prompt: z.string(), // tell me a joke // prompt
  vector: z.array(z.number()), // debugging to check if you're reusing the same vector
  response: z.string(), // llm response
  cache: z.boolean(), // if hit or not
  latency: z
    .object({
      vectorize: z.number(),
      cache: z.number(),
    })
    .transform((o) => JSON.stringify(o)),
  tokens: z.number(), // how many tokens were used
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
