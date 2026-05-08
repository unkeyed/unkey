import { auditLogsQueryPayload } from "@/components/audit-logs-table/schema/audit-logs.schema";
import { auth } from "@/lib/auth/server";
import type { User } from "@/lib/auth/types";
import { clickhouse } from "@/lib/clickhouse";
import type { Quotas, Workspace } from "@/lib/db";
import { freeTierQuotas } from "@/lib/quotas";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { type AuditQueryLogsParams, auditLog } from "./schema";
import { transformFilters } from "./utils";

const LIMIT = 50;
const MAX_LIMIT = 200;

const countCache = new Map<string, { count: number; timestamp: number }>();
const COUNT_CACHE_TTL = 1000 * 60 * 5;

function getCountCacheKey(workspaceId: string, params: Omit<AuditQueryLogsParams, "workspaceId">) {
  return JSON.stringify({
    workspaceId,
    bucket: params.bucket,
    events: params.events,
    users: params.users,
    rootKeys: params.rootKeys,
    startTime: params.startTime,
    endTime: params.endTime,
  });
}

function getCachedCount(key: string): number | null {
  const cached = countCache.get(key);
  if (cached && Date.now() - cached.timestamp < COUNT_CACHE_TTL) {
    return cached.count;
  }
  if (cached) {
    countCache.delete(key);
  }
  return null;
}

const AuditLogsResponse = z.object({
  auditLogs: z.array(auditLog),
  total: z.number(),
});

type AuditLogsResponse = z.infer<typeof AuditLogsResponse>;

function nullIfEmpty(s: string): string | null {
  return s.length === 0 ? null : s;
}

function parseJSON(s: string): unknown {
  if (s.length === 0) {
    return null;
  }
  try {
    return JSON.parse(s);
  } catch {
    return s;
  }
}

export const fetchAuditLog = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(auditLogsQueryPayload)
  .output(AuditLogsResponse)
  .query(async ({ ctx, input }) => {
    const params = transformFilters(input);
    const pageSize = Math.max(1, Math.min(params.limit ?? LIMIT, MAX_LIMIT));
    const page = Math.max(1, params.page ?? 1);
    const offset = (page - 1) * pageSize;

    const queryArgs = buildQueryArgs(params, ctx.workspace, pageSize, offset);
    const cacheKey = getCountCacheKey(ctx.workspace.id, params);
    const cachedCount = getCachedCount(cacheKey);

    const { logsQuery, totalQuery } = clickhouse.auditLogs.logs(queryArgs);

    const [countResult, logsResult] = await Promise.all([
      cachedCount !== null
        ? Promise.resolve({ val: [{ total_count: cachedCount }], err: null })
        : totalQuery,
      logsQuery,
    ]);

    if (countResult.err || logsResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching audit logs from clickhouse.",
      });
    }

    const totalCount = countResult.val[0]?.total_count ?? 0;
    if (cachedCount === null) {
      countCache.set(cacheKey, { count: totalCount, timestamp: Date.now() });
    }

    const logs = logsResult.val;
    const uniqueUsers = await fetchUsersByActorIds(
      logs.filter((l) => l.actor_type === "user").map((l) => l.actor_id),
    );

    const items: AuditLogsResponse["auditLogs"] = logs.map((l) => {
      const user = uniqueUsers[l.actor_id];
      return {
        user: user
          ? {
              username: user.email,
              firstName: user.firstName,
              lastName: user.lastName,
              imageUrl: user.avatarUrl,
            }
          : undefined,
        auditLog: {
          id: l.event_id,
          time: l.time,
          actor: {
            id: l.actor_id,
            name: nullIfEmpty(l.actor_name),
            type: l.actor_type,
          },
          location: nullIfEmpty(l.remote_ip),
          description: l.description,
          userAgent: nullIfEmpty(l.user_agent),
          event: l.event,
          workspaceId: ctx.workspace.id,
          targets: l.targets.map(([type, id, name, meta]) => ({
            id,
            type,
            name: nullIfEmpty(name),
            meta: parseJSON(meta),
          })),
        },
      };
    });

    return {
      auditLogs: items,
      total: totalCount,
    };
  });

// Platform audit events (key.create, api.delete, etc.) are owned by the
// Unkey platform, not by the customer: they record Unkey's actions on a
// customer's resources. In ClickHouse those rows carry an empty
// workspace_id (meaning "platform-owned") and a bucket_id of
// `unkey_audit_<customer_ws_id>`. Once customer-emitted audit logs ship
// alongside, their rows will carry the customer's workspace_id as the
// owner and a customer-chosen bucket_id. Same query shape, different
// filter values.
//
// This contract is mirrored by the Go worker that writes these rows, see
// svc/ctrl/worker/auditlogexport/run_export_handler.go
// (unkeyPlatformWorkspaceID and unkeyPlatformBucketID). The two MUST stay
// in sync; changing one without the other silently breaks the dashboard.
const UNKEY_PLATFORM_WORKSPACE_ID = "";
const platformBucketFor = (customerWorkspaceId: string) => `unkey_audit_${customerWorkspaceId}`;

function buildQueryArgs(
  params: Omit<AuditQueryLogsParams, "workspaceId">,
  workspace: Workspace & { quotas: Quotas | null },
  limit: number,
  offset: number,
) {
  const events = (params.events ?? []).map((e) => e.value);
  const userValues = (params.users ?? []).map((u) => u.value);
  const rootKeyValues = (params.rootKeys ?? []).map((r) => r.value);
  const actorIds = [...userValues, ...rootKeyValues];

  const retentionDays =
    workspace.quotas?.auditLogsRetentionDays || freeTierQuotas.auditLogsRetentionDays;
  const retentionCutoffUnixMilli = Date.now() - retentionDays * 24 * 60 * 60 * 1000;

  const startTime = Math.max(
    params.startTime ?? retentionCutoffUnixMilli,
    retentionCutoffUnixMilli,
  );
  const endTime = params.endTime ?? Date.now();

  return {
    workspaceId: UNKEY_PLATFORM_WORKSPACE_ID,
    bucketId: platformBucketFor(workspace.id),
    limit,
    offset,
    startTime,
    endTime,
    events,
    actorIds,
  };
}

async function fetchUsersByActorIds(actorIds: string[]): Promise<Record<string, User>> {
  try {
    const unique = [...new Set(actorIds)];
    const users = await Promise.all(unique.map((userId) => auth.getUser(userId).catch(() => null)));

    return users.reduce(
      (acc, user) => {
        if (user) {
          acc[user.id] = user;
        }
        return acc;
      },
      {} as Record<string, User>,
    );
  } catch (error) {
    console.error("Error fetching users:", error);
    return {};
  }
}
