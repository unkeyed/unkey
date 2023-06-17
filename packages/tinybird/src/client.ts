import type { Tinybird } from "@chronark/zod-bird";
import { z } from "zod";

export const publishKeyVerification = (tb: Tinybird) =>
  tb.buildIngestEndpoint({
    datasource: "key_verifications__v1",
    event: z.object({
      keyId: z.string(),
      apiId: z.string(),
      tenantId: z.string(),
      time: z.number(),
      ratelimited: z.boolean(),
    }),
  });

export const getApiUsage = (tb: Tinybird) =>
  tb.buildPipe({
    pipe: "endpoint__api_usage__v1",
    parameters: z.object({
      tenantId: z.string(),
      apiId: z.string(),
    }),
    data: z.object({
      time: z.string().transform((t) => new Date(t).getTime()),
      usage: z.number(),
    }),
  });
