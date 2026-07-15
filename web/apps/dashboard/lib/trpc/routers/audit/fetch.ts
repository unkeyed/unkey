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

    const { getLogsQuery, getTotalQuery } = clickhouse.auditLogs.logs(queryArgs);

    const [countResult, logsResult] = await Promise.all([
      cachedCount !== null
        ? Promise.resolve({ val: [{ totalCount: cachedCount }], err: null })
        : getTotalQuery(),
      getLogsQuery(),
    ]);

    if (countResult.err || logsResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching audit logs from clickhouse.",
      });
    }

    const totalCount = countResult.val[0]?.totalCount ?? 0;
    if (cachedCount === null) {
      countCache.set(cacheKey, { count: totalCount, timestamp: Date.now() });
    }

    const logs = logsResult.val;
    const uniqueUsers = await fetchUsersByActorIds(
      logs.filter((l) => l.actorType === "user").map((l) => l.actorId),
    );

    const items: AuditLogsResponse["auditLogs"] = logs.map((l) => {
      const user = uniqueUsers[l.actorId];
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
          id: l.eventId,
          time: l.time,
          actor: {
            id: l.actorId,
            name: nullIfEmpty(l.actorName),
            type: l.actorType,
          },
          location: nullIfEmpty(l.remoteIp),
          description: l.description,
          userAgent: nullIfEmpty(l.userAgent),
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
    workspaceId: workspace.id,
    bucketId: params.bucket,
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
