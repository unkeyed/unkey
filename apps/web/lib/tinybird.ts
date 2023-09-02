import { env } from "@/lib/env";
import { Tinybird } from "@chronark/zod-bird";
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

export const getDailyUsage = tb.buildPipe({
  pipe: "endpoint_get_usage__v2",
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

export const getActiveCountPerApiPerDay = tb.buildPipe({
  pipe: "endpoint_get_active_keys__v2",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string().optional(),
    start: z.number(),
    end: z.number(),
  }),
  data: z.object({
    active: z.number(),
  }),
});

export const getTotalVerificationsForWorkspace = tb.buildPipe({
  pipe: "endpoint_billing_get_verifications_usage__v1",
  parameters: z.object({
    workspaceId: z.string(),
    start: z.number(),
    end: z.number(),
  }),
  data: z.object({ usage: z.number() }),
  opts: {
    cache: "no-store",
  },
});

export const getTotalActiveKeys = tb.buildPipe({
  pipe: "endpoint_billing_get_active_keys_usage__v2",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string().optional(),
    start: z.number(),
    end: z.number(),
  }),
  data: z.object({ usage: z.number() }),
  opts: {
    cache: "no-store",
  },
});

export const getLatestVerifications = tb.buildPipe({
  pipe: "endpoint__get_latest_verifications__v1",
  parameters: z.object({
    keyId: z.string(),
  }),
  data: z.object({
    time: z.number(),
    requestedResource: z.string(),
    ratelimited: z.number().transform((n) => n > 0),
    usageExceeded: z.number().transform((n) => n > 0),
    region: z.string(),
    userAgent: z.string(),
    ipAddress: z.string(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getTotalUsage = tb.buildPipe({
  pipe: "endpoint__get_total_usage_for_key__v1",
  parameters: z.object({
    keyId: z.string(),
  }),
  data: z.object({
    totalUsage: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getLastUsed = tb.buildPipe({
  pipe: "endpoint__get_last_used__v1",
  parameters: z.object({
    keyId: z.string(),
  }),
  data: z.object({
    lastUsed: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});
