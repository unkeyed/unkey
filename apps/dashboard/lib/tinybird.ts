import { time } from "node:console";
import { env } from "@/lib/env";
import { NoopTinybird, Tinybird } from "@chronark/zod-bird";
import { newId } from "@unkey/id";
import { auditLogSchemaV1, unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import { z } from "zod";
import type { MaybeArray } from "./types";

const token = env().TINYBIRD_TOKEN;
const tb = token ? new Tinybird({ token }) : new NoopTinybird();

const datetimeToUnixMilli = z.string().transform((t) => new Date(t).getTime());

/**
 * `t` has the format `2021-01-01 00:00:00`
 *
 * If we transform it as is, we get `1609459200000` which is `2021-01-01 01:00:00` due to fun timezone stuff.
 * So we split the string at the space and take the date part, and then parse that.
 */
const dateToUnixMilli = z.string().transform((t) => new Date(t.split(" ").at(0) ?? t).getTime());

export const getDailyVerifications = tb.buildPipe({
  pipe: "endpoint__get_daily_verifications__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string().optional(),
    keyId: z.string().optional(),
  }),
  data: z.object({
    time: datetimeToUnixMilli,
    success: z.number(),
    rateLimited: z.number(),
    usageExceeded: z.number(),
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

export const getTotalVerifications = tb.buildPipe({
  pipe: "endpoint__all_verifications__v1",
  data: z.object({ verifications: z.number() }),
  opts: {
    cache: "no-store",
  },
});

export const getLatestVerifications = tb.buildPipe({
  pipe: "endpoint__get_latest_verifications__v2",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
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

export const getTotalVerificationsForKey = tb.buildPipe({
  pipe: "endpoint__get_total_usage_for_key__v1",
  parameters: z.object({
    keyId: z.string(),
  }),
  data: z.object({ totalUsage: z.number() }),
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

export const getActiveKeysPerHourForAllWorkspaces = tb.buildPipe({
  pipe: "endpoint_billing_get_active_keys_per_workspace_per_hour__v2__v1",

  data: z.object({
    usage: z.number(),
    workspaceId: z.string(),
    time: datetimeToUnixMilli,
  }),
  opts: {
    cache: "no-store",
  },
});

export const getVerificationsPerHourForAllWorkspaces = tb.buildPipe({
  pipe: "endpoint__billing_verifications_per_hour__v1",

  data: z.object({
    verifications: z.number(),
    workspaceId: z.string(),
    time: datetimeToUnixMilli,
  }),
  opts: {
    cache: "no-store",
  },
});

export const activeKeys = tb.buildPipe({
  pipe: "endpoint__active_keys_by_workspace__v1",
  parameters: z.object({
    workspaceId: z.string(),
    year: z.number().int(),
    month: z.number().int().min(1).max(12),
  }),
  data: z.object({
    keys: z.number().int().nullable().default(0),
  }),
  opts: {
    cache: "no-store",
  },
});

export const verifications = tb.buildPipe({
  pipe: "endpoint__verifications_by_workspace__v1",
  parameters: z.object({
    workspaceId: z.string(),
    year: z.number().int(),
    month: z.number().int().min(1).max(12),
  }),

  data: z.object({
    success: z.number().int().nullable().default(0),
    ratelimited: z.number().int().nullable().default(0),
    usageExceeded: z.number().int().nullable().default(0),
  }),
  opts: {
    cache: "no-store",
  },
});

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

export const getVerificationsMonthly = tb.buildPipe({
  pipe: "get_verifications_monthly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    keyId: z.string().optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    success: z.number(),
    rateLimited: z.number(),
    usageExceeded: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getVerificationsWeekly = tb.buildPipe({
  pipe: "get_verifications_weekly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    keyId: z.string().optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    success: z.number(),
    rateLimited: z.number(),
    usageExceeded: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getVerificationsDaily = tb.buildPipe({
  pipe: "get_verifications_daily__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    keyId: z.string().optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    success: z.number(),
    rateLimited: z.number(),
    usageExceeded: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getVerificationsHourly = tb.buildPipe({
  pipe: "get_verifications_hourly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    keyId: z.string().optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: datetimeToUnixMilli,
    success: z.number(),
    rateLimited: z.number(),
    usageExceeded: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getActiveKeysHourly = tb.buildPipe({
  pipe: "get_active_keys_hourly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: datetimeToUnixMilli,
    keys: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getActiveKeysDaily = tb.buildPipe({
  pipe: "get_active_keys_daily__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    keys: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getActiveKeysWeekly = tb.buildPipe({
  pipe: "get_active_keys_weekly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    keys: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getActiveKeysMonthly = tb.buildPipe({
  pipe: "get_active_keys_monthly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    keys: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

/**
 * Across the entire time period
 */
export const getActiveKeys = tb.buildPipe({
  pipe: "get_active_keys__v1",
  parameters: z.object({
    workspaceId: z.string(),
    apiId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    keys: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getQ1ActiveWorkspaces = tb.buildPipe({
  pipe: "get_q1_goal_distinct_workspaces__v1",
  parameters: z.object({}),
  data: z.object({
    workspaces: z.number(),
    time: datetimeToUnixMilli,
  }),
  opts: {
    cache: "no-store",
  },
});

export const getAuditLogActors = tb.buildPipe({
  pipe: "endpoint__audit_log_actor_ids__v1",
  parameters: z.object({
    workspaceId: z.string(),
    type: z.enum(["user", "key"]).optional(),
  }),
  data: z.object({
    actorId: z.string(),
    actorType: z.enum(["user", "key"]).optional(),
    lastSeen: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getAuditLogs = tb.buildPipe({
  pipe: "endpoint__audit_logs__v1",
  parameters: z.object({
    workspaceId: z.string(),
    bucket: z.string().default("unkey_mutations"),
    before: z.number().int().optional(),
    after: z.number().int(),
    events: z.array(z.string()).optional(),
    actorIds: z.array(z.string()).optional(),
  }),

  data: z
    .object({
      workspaceId: z.string(),
      bucket: z.string(),
      auditLogId: z.string(),
      time: z.number().int(),
      actorType: z.enum(["key", "user"]),
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
    })),
  opts: {
    cache: "no-store",
  },
});

export function ingestAuditLogs(
  logs: MaybeArray<{
    workspaceId: string;
    event: z.infer<typeof unkeyAuditLogEvents>;
    description: string;
    actor: {
      type: "user" | "key";
      name?: string;
      id: string;
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
        | "secret";

      id: string;
      meta?: Record<string, string | number | boolean | null>;
    }>;
    context: {
      userAgent?: string;
      location: string;
    };
  }>,
) {
  return tb.buildIngestEndpoint({
    datasource: "audit_logs__v2",
    event: auditLogSchemaV1
      .merge(
        z.object({
          event: unkeyAuditLogEvents,
          auditLogId: z.string().default(newId("auditLog")),
          bucket: z.string().default("unkey_mutations"),
          time: z.number().default(Date.now()),
        }),
      )
      .transform((l) => ({
        ...l,
        actor: {
          ...l.actor,
          meta: l.actor.meta ? JSON.stringify(l.actor.meta) : undefined,
        },
        resources: JSON.stringify(l.resources),
      })),
  })(logs);
}

export const getRatelimitsHourly = tb.buildPipe({
  pipe: "get_ratelimits_hourly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    namespaceId: z.string(),
    identifier: z.array(z.string()).optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: datetimeToUnixMilli,
    success: z.number(),
    total: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getRatelimitsMinutely = tb.buildPipe({
  pipe: "get_ratelimits_minutely__v1",
  parameters: z.object({
    workspaceId: z.string(),
    namespaceId: z.string(),
    identifier: z.array(z.string()).optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: datetimeToUnixMilli,
    success: z.number(),
    total: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});
export const getRatelimitsDaily = tb.buildPipe({
  pipe: "get_ratelimits_daily__v1",
  parameters: z.object({
    workspaceId: z.string(),
    namespaceId: z.string(),
    identifier: z.array(z.string()).optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    success: z.number(),
    total: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getRatelimitsMonthly = tb.buildPipe({
  pipe: "get_ratelimits_monthly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    namespaceId: z.string(),
    identifier: z.array(z.string()).optional(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.object({
    time: dateToUnixMilli,
    success: z.number(),
    total: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getRatelimitIdentifiersMinutely = tb.buildPipe({
  pipe: "get_ratelimit_identifiers_minutely__v1",
  parameters: z.object({
    workspaceId: z.string(),
    namespaceId: z.string(),
    start: z.number(),
    end: z.number(),
    orderBy: z.enum(["success", "total"]).optional().default("total"),
  }),
  data: z.object({
    identifier: z.string(),
    success: z.number(),
    total: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getRatelimitIdentifiersHourly = tb.buildPipe({
  pipe: "get_ratelimit_identifiers_hourly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    namespaceId: z.string(),
    start: z.number(),
    end: z.number(),
    orderBy: z.enum(["success", "total"]).optional().default("total"),
  }),
  data: z.object({
    identifier: z.string(),
    success: z.number(),
    total: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getRatelimitIdentifiersDaily = tb.buildPipe({
  pipe: "get_ratelimit_identifiers_daily__v1",
  parameters: z.object({
    workspaceId: z.string(),
    namespaceId: z.string(),
    start: z.number(),
    end: z.number(),
    orderBy: z.enum(["success", "total"]).optional().default("total"),
  }),
  data: z.object({
    identifier: z.string(),
    success: z.number(),
    total: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getRatelimitIdentifiersMonthly = tb.buildPipe({
  pipe: "get_ratelimit_identifiers_monthly__v1",
  parameters: z.object({
    workspaceId: z.string(),
    namespaceId: z.string(),
    start: z.number(),
    end: z.number(),
    orderBy: z.enum(["success", "total"]).optional().default("total"),
  }),
  data: z.object({
    identifier: z.string(),
    success: z.number(),
    total: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getRatelimitLastUsed = tb.buildPipe({
  pipe: "get_ratelimits_last_used__v1",
  parameters: z.object({
    workspaceId: z.string(),
    namespaceId: z.string(),
    identifier: z.array(z.string()).optional(),
  }),
  data: z.object({
    lastUsed: z.number(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getRatelimitEvents = tb.buildPipe({
  pipe: "get_ratelimit_events__v1",
  parameters: z.object({
    workspaceId: z.string(),
    namespaceId: z.string(),
    after: z.number().optional(),
    before: z.number().optional(),
    limit: z.number().optional(),
    success: z
      .boolean()
      .optional()
      .transform((b) => (typeof b === "boolean" ? (b ? 1 : 0) : undefined)),
    ipAddress: z.array(z.string()).optional(),
    country: z.array(z.string()).optional(),
    identifier: z.array(z.string()).optional(),
  }),
  data: z.object({
    identifier: z.string(),
    requestId: z.string(),
    time: z.number(),
    success: z
      .number()
      .transform((n) => n > 0)
      .optional(),

    remaining: z.number(),
    limit: z.number(),
    country: z.string(),
    ipAddress: z.string(),
  }),
  opts: {
    cache: "no-store",
  },
});

export const getAllSemanticCacheLogs = tb.buildPipe({
  pipe: "get_all_semantic_cache_logs__v1",
  parameters: z.object({
    limit: z.number().optional(),
    gatewayId: z.string(),
    workspaceId: z.string(),
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
  pipe: "get_semantic_caches_daily__v2",
  parameters: z.object({
    gatewayId: z.string(),
    workspaceId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.any(),
  opts: {
    cache: "no-store",
  },
});

export const getSemanticCachesHourly = tb.buildPipe({
  pipe: "get_semantic_caches_hourly__v2",
  parameters: z.object({
    gatewayId: z.string(),
    workspaceId: z.string(),
    start: z.number().optional(),
    end: z.number().optional(),
  }),
  data: z.any(),
  opts: {
    cache: "no-store",
  },
});

// public get getVerificationsByOwnerId() {
//   return this.client.buildPipe({
//     pipe: "get_verifictions_by_keySpaceId__v1",
//     parameters: z.object({
//       workspaceId: z.string(),
//       keySpaceId: z.string(),
//       start: z.number(),
//       end: z.number(),
//     }),
//     data: z.object({
//       ownerId: z.string(),
//       verifications: z.number(),
//     }),
//   });
// }
// }
