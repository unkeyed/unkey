import { Tinybird } from "@chronark/zod-bird";
import { env } from "@/lib/env";
import { z } from "zod";

const tb = new Tinybird({ token: env.TINYBIRD_TOKEN });

export const publishKeyVerification = tb.buildIngestEndpoint({
  datasource: "key_verifications__v1",
  event: z.object({
    keyId: z.string(),
    apiId: z.string(),
    workspaceId: z.string(),
    time: z.number(),
    ratelimited: z.boolean(),
  }),
});

export const getUsage = tb.buildPipe({
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

export const getActiveCount = tb.buildPipe({
  pipe: "endpoint__get_active_keys__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string().optional(),
  }),
  data: z.object({
    active: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});
