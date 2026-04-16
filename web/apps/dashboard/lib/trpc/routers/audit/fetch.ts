import { auditLogsQueryPayload } from "@/components/audit-logs-table/schema/audit-logs.schema";
import { auth } from "@/lib/auth/server";
import type { User } from "@/lib/auth/types";
import {
  type Quotas,
  type Workspace,
  and,
  between,
  count,
  db,
  eq,
  gte,
  inArray,
  schema,
} from "@/lib/db";
import { freeTierQuotas } from "@/lib/quotas";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { type AuditLogWithTargets, type AuditQueryLogsParams, auditLog } from "./schema";
import { transformFilters } from "./utils";

const LIMIT = 50;
const MAX_LIMIT = 200;

// In-memory cache for audit log count queries to avoid re-counting on every page navigation
const countCache = new Map<string, { count: number; timestamp: number }>();
const COUNT_CACHE_TTL = 1000 * 60 * 5; // 5 minutes

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

export const fetchAuditLog = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(auditLogsQueryPayload)
  .output(AuditLogsResponse)
  .query(async ({ ctx, input }) => {
    const params = transformFilters(input);
    const pageSize = Math.max(1, Math.min(params.limit ?? LIMIT, MAX_LIMIT));
    const page = Math.max(1, params.page ?? 1);
    const offset = (page - 1) * pageSize;

    const whereConditions = buildWhereConditions(params, ctx.workspace);
    const cacheKey = getCountCacheKey(ctx.workspace.id, params);
    const cachedCount = getCachedCount(cacheKey);

    const [totalCount, logs] = await (cachedCount !== null
      ? Promise.all([
          cachedCount,
          db.query.auditLog.findMany({
            where: and(...whereConditions),
            with: { targets: true },
            orderBy: (table, { desc }) => [desc(table.time), desc(table.id)],
            limit: pageSize,
            offset,
          }),
        ])
      : Promise.all([
          db
            .select({ count: count() })
            .from(schema.auditLog)
            .where(and(...whereConditions))
            .then((result) => {
              const total = result[0]?.count ?? 0;
              countCache.set(cacheKey, { count: total, timestamp: Date.now() });
              return total;
            }),
          db.query.auditLog.findMany({
            where: and(...whereConditions),
            with: { targets: true },
            orderBy: (table, { desc }) => [desc(table.time), desc(table.id)],
            limit: pageSize,
            offset,
          }),
        ]));

    const uniqueUsers = await fetchUsersFromLogs(logs);

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
          id: l.id,
          time: l.time,
          actor: {
            id: l.actorId,
            name: l.actorName,
            type: l.actorType,
          },
          location: l.remoteIp,
          description: l.display,
          userAgent: l.userAgent,
          event: l.event,
          workspaceId: l.workspaceId,
          targets: l.targets.map((t) => ({
            id: t.id,
            type: t.type,
            name: t.name,
            meta: t.meta,
          })),
        },
      };
    });

    return {
      auditLogs: items,
      total: totalCount,
    };
  });

function buildWhereConditions(
  params: Omit<AuditQueryLogsParams, "workspaceId">,
  workspace: Workspace & { quotas: Quotas | null },
) {
  const events = (params.events ?? []).map((e) => e.value);
  const userValues = (params.users ?? []).map((u) => u.value);
  const rootKeyValues = (params.rootKeys ?? []).map((r) => r.value);
  const users = [...userValues, ...rootKeyValues];

  const retentionDays =
    workspace.quotas?.auditLogsRetentionDays || freeTierQuotas.auditLogsRetentionDays;
  const retentionCutoffUnixMilli = Date.now() - retentionDays * 24 * 60 * 60 * 1000;

  const hasTimeFilter = params.startTime !== undefined || params.endTime !== undefined;

  const conditions = [
    eq(schema.auditLog.workspaceId, workspace.id),
    eq(schema.auditLog.bucket, params.bucket),
  ];

  if (events.length > 0) {
    conditions.push(inArray(schema.auditLog.event, events));
  }

  if (hasTimeFilter) {
    conditions.push(
      between(
        schema.auditLog.time,
        Math.max(params.startTime ?? retentionCutoffUnixMilli, retentionCutoffUnixMilli),
        params.endTime ?? Date.now(),
      ),
    );
  } else {
    conditions.push(gte(schema.auditLog.time, retentionCutoffUnixMilli));
  }

  if (users.length > 0) {
    conditions.push(inArray(schema.auditLog.actorId, users));
  }

  return conditions;
}

export const fetchUsersFromLogs = async (
  logs: AuditLogWithTargets[],
): Promise<Record<string, User>> => {
  try {
    const userIds = [...new Set(logs.filter((l) => l.actorType === "user").map((l) => l.actorId))];

    const users = await Promise.all(
      userIds.map((userId) => auth.getUser(userId).catch(() => null)),
    );

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
};
