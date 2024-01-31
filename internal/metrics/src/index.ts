import { z } from "zod";

export const metricSchema = z.discriminatedUnion("metric", [
  z.object({
    metric: z.literal("metric.cache.read"),
    key: z.string(),
    hit: z.boolean(),
    latency: z.number(),
    tier: z.string(),
    namespace: z.string(),
  }),
  z.object({
    metric: z.literal("metric.cache.write"),
    key: z.string(),
    tier: z.string(),
    namespace: z.string(),
  }),
  z.object({
    metric: z.literal("metric.cache.purge"),
    key: z.string(),
    tier: z.string(),
    namespace: z.string(),
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
    requestId: z.string(),
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
    query: z.enum(["getKeyAndApiByHash"]),
    latency: z.number(),
  }),
  z.object({
    metric: z.literal("metric.ratelimit"),
    keyId: z.string(),
    latency: z.number(),
    tier: z.enum(["memory", "durable", "total"]),
  }),
  z.object({
    metric: z.literal("metric.usagelimit"),
    keyId: z.string(),
    latency: z.number(),
  }),
]);

export type Metric = z.infer<typeof metricSchema>;
