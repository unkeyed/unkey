import { env } from "@/lib/env";
import { NoopTinybird, Tinybird } from "@chronark/zod-bird";
import type { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { z } from "zod";

const token = env().TINYBIRD_TOKEN;
const tb = token ? new Tinybird({ token }) : new NoopTinybird();

const datetimeToUnixMilli = z.string().transform((t) => new Date(t).getTime());

/**
 * `t` has the format `2021-01-01 00:00:00`
 *
 * If we transform it as is, we get `1609459200000` which is `2021-01-01 01:00:00` due to fun timezone stuff.
 * So we split the string at the space and take the date part, and then parse that.
 */
const _dateToUnixMilli = z.string().transform((t) => new Date(t.split(" ").at(0) ?? t).getTime());

export const ratelimits = tb.buildPipe({
  pipe: "endpoint__ratelimits_by_workspace__v1",
  parameters: z.object({
    workspaceId: z.string(),
    year: z.number().int(),
    month: z.number().int().min(1).max(12),
  }),

  data: z.object({
    success: z.number().int().nullable().default(0),
    total: z.number().int().nullable().default(0),
  }),
  opts: {
    cache: "no-store",
  },
});

export const auditLogsDataSchema = z
  .object({
    workspaceId: z.string(),
    bucket: z.string(),
    auditLogId: z.string(),
    time: z.number().int(),
    actorType: z.enum(["key", "user", "system"]),
    actorId: z.string(),
    actorName: z.string().nullable(),
    actorMeta: z.string().nullable(),
    event: z.string(),
    description: z.string(),
    resources: z.string().transform((rs) =>
      z
        .array(
          z.object({
            type: z.string(),
            id: z.string(),
            meta: z.record(z.union([z.string(), z.number(), z.boolean(), z.null()])).optional(),
          }),
        )
        .parse(JSON.parse(rs)),
    ),

    location: z.string(),
    userAgent: z.string().nullable(),
  })
  .transform((l) => ({
    workspaceId: l.workspaceId,
    bucket: l.bucket,
    auditLogId: l.auditLogId,
    time: l.time,
    actor: {
      type: l.actorType,
      id: l.actorId,
      name: l.actorName,
      meta: l.actorMeta ? JSON.parse(l.actorMeta) : undefined,
    },
    event: l.event,
    description: l.description,
    resources: l.resources,
    context: {
      location: l.location,
      userAgent: l.userAgent,
    },
  }));

export type UnkeyAuditLog = {
  workspaceId: string;
  event: z.infer<typeof unkeyAuditLogEvents>;
  description: string;
  actor: {
    type: "user" | "key";
    name?: string;
    id: string;
    meta?: Record<string, string | number | boolean | null>;
  };
  resources: Array<{
    type:
      | "key"
      | "api"
      | "workspace"
      | "role"
      | "permission"
      | "keyAuth"
      | "vercelBinding"
      | "vercelIntegration"
      | "ratelimitNamespace"
      | "ratelimitOverride"
      | "gateway"
      | "llmGateway"
      | "webhook"
      | "reporter"
      | "secret"
      | "identity"
      | "auditLogBucket";

    id: string;
    meta?: Record<string, string | number | boolean | null>;
  }>;
  context: {
    userAgent?: string;
    location: string;
  };
};
export const getAllSemanticCacheLogs = tb.buildPipe({
  pipe: "get_all_semantic_cache_logs__v1",
  parameters: z.object({
    limit: z.number().optional(),
    gatewayId: z.string(),
    workspaceId: z.string(),
    interval: z.number(),
  }),
  data: z.object({
    time: z.number(),
    model: z.string(),
    stream: z.number(),
    query: z.string(),
    vector: z.array(z.number()),
    response: z.string(),
    cache: z.number(),
    serviceLatency: z.number(),
    embeddingsLatency: z.number(),
    vectorizeLatency: z.number(),
    inferenceLatency: z.number().optional(),
    cacheLatency: z.number(),
    tokens: z.number(),
    requestId: z.string(),
    workspaceId: z.string(),
    gatewayId: z.string(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getSemanticCachesDaily = tb.buildPipe({
  pipe: "get_semantic_caches_daily__v4",
  parameters: z.object({
    gatewayId: z.string(),
    workspaceId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    model: z.string(),
    time: datetimeToUnixMilli,
    hit: z.number(),
    total: z.number(),
    avgServiceLatency: z.number(),
    avgEmbeddingsLatency: z.number(),
    avgVectorizeLatency: z.number(),
    avgInferenceLatency: z.number().nullable(),
    avgCacheLatency: z.number(),
    avgTokens: z.number(),
    sumTokens: z.number(),
    cachedTokens: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getSemanticCachesHourly = tb.buildPipe({
  pipe: "get_semantic_caches_hourly__v4",
  parameters: z.object({
    gatewayId: z.string(),
    workspaceId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    model: z.string(),
    time: datetimeToUnixMilli,
    hit: z.number(),
    total: z.number(),
    avgServiceLatency: z.number(),
    avgEmbeddingsLatency: z.number(),
    avgVectorizeLatency: z.number(),
    avgInferenceLatency: z.number().nullable(),
    avgCacheLatency: z.number(),
    avgTokens: z.number(),
    sumTokens: z.number(),
    cachedTokens: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});
