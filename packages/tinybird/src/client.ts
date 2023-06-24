import type { Tinybird } from "@chronark/zod-bird";
import { z } from "zod";

export const publishKeyVerification = (tb: Tinybird) =>
  tb.buildIngestEndpoint({
    datasource: "key_verifications__v1",
    event: z.object({
      keyId: z.string(),
      apiId: z.string(),
      workspaceId: z.string(),
      time: z.number(),
      ratelimited: z.boolean(),
    }),
  });

export const getUsage = (tb: Tinybird) =>
  tb.buildPipe({
    pipe: "endpoint__get_usage__v2",
    parameters: z.object({
      workspaceId: z.string(),
      apiId: z.string().optional(),
      keyId: z.string().optional(),
    }),
    data: z.object({
      time: z.string().transform((t) => new Date(t).getTime()),
      usage: z.number(),
    }),
    opts: {
      cache: "no-store",
    },
  });
