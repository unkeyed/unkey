import { z } from "zod";

export const metricSchema = z.discriminatedUnion("metric", [
  z.object({
    metric: z.literal("metric.cache.read"),
    key: z.string(),
    hit: z.boolean(),
    status: z.enum(["fresh", "stale"]).optional(),
    latency: z.number(),
    tier: z.string(),
    namespace: z.string(),
  }),
  z.object({
    metric: z.literal("metric.cache.write"),
    key: z.string(),
    tier: z.string(),
    latency: z.number(),
    namespace: z.string(),
  }),
  z.object({
    metric: z.literal("metric.cache.remove"),
    key: z.string(),
    tier: z.string(),
    namespace: z.string(),
    latency: z.number(),
  }),
  z.object({
    metric: z.literal("metric.key.verification"),
    valid: z.boolean(),
    code: z.string(),
    workspaceId: z.string().optional(),
    apiId: z.string().optional(),
    keyId: z.string().optional(),
  }),
  z.object({
    metric: z.literal("metric.http.request"),
    path: z.string(),
    method: z.string(),
    status: z.number(),
    error: z.string().optional(),
    serviceLatency: z.number(),
    // Regional data might be different on non-cloudflare deployments
    colo: z.string().optional(),
    continent: z.string().optional(),
    country: z.string().optional(),
    city: z.string().optional(),
    userAgent: z.string().optional(),
    fromAgent: z.string().optional(),
  }),
  z.object({
    metric: z.literal("metric.db.read"),
    query: z.enum(["getKeyAndApiByHash", "loadFromOrigin", "getKeysByKeyAuthId"]),
    latency: z.number(),
  }),
  z.object({
    metric: z.literal("metric.ratelimit"),
    workspaceId: z.string(),
    namespaceId: z.string().optional(),
    identifier: z.string(),
    latency: z.number(),
    mode: z.enum(["sync", "async"]),
    success: z.boolean().optional(),
    error: z.boolean().optional(),
  }),
  z.object({
    metric: z.literal("metric.usagelimit"),
    keyId: z.string(),
    latency: z.number(),
  }),
  z.object({
    metric: z.literal("metric.ratelimit.accuracy"),
    workspaceId: z.string(),
    namespaceId: z.string().optional(),
    identifier: z.string(),
    responded: z.boolean(),
    correct: z.boolean(),
  }),

  z.object({
    metric: z.literal("metric.vault.latency"),
    op: z.enum([
      "encrypt",
      "encryptBulk",
      "decrypt",
      "reEncrypt",
      "createDEK",
      "liveness",
      "reEncryptDEKs",
    ]),
    latency: z.number(),
  }),
  z.object({
    metric: z.literal("metric.server.latency"),
    country: z.string(),
    continent: z.string(),
    latency: z.number(),
    platform: z.string(),
    colo: z.string(),
  }),
]);

export type Metric = z.infer<typeof metricSchema>;
