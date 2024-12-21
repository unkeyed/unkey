import { auditQueryLogsPayload } from "@/app/(app)/audit/components/table/query-logs.schema";
import { type Workspace, db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import type { User } from "@/lib/auth/types";
import { auth } from '@/lib/auth/server';
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { type AuditLogWithTargets, type AuditQueryLogsParams, auditLog } from "./schema";
import { transformFilters } from "./utils";

const AuditLogsResponse = z.object({
  auditLogs: z.array(auditLog),
  hasMore: z.boolean(),
  nextCursor: z
    .object({
      time: z.number().int(),
      auditId: z.string(),
    })
    .optional(),
});

type AuditLogsResponse = z.infer<typeof AuditLogsResponse>;

export const fetchAuditLog = rateLimitedProcedure(ratelimit.read)
  .input(auditQueryLogsPayload)
  .output(AuditLogsResponse)
  .query(async ({ ctx, input }) => {
    const params = transformFilters(input);
    const result = await queryAuditLogs(params, ctx.workspace);

    if (!result) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Audit log bucket not found",
      });
    }

    const { slicedItems, hasMore } = omitLastItemForPagination(result.logs, params.limit);
    const uniqueUsers = await fetchUsersFromLogs(slicedItems);

    const items: AuditLogsResponse["auditLogs"] = slicedItems.map((l) => {
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
      hasMore,
      nextCursor:
        hasMore && items.length > 0
          ? {
              time: items[items.length - 1].auditLog.time,
              auditId: items[items.length - 1].auditLog.id,
            }
          : undefined,
    };
  });

export const queryAuditLogs = async (
  params: Omit<AuditQueryLogsParams, "workspaceId">,
  workspace: Workspace,
) => {
  const events = (params.events ?? []).map((e) => e.value);
  const userValues = (params.users ?? []).map((u) => u.value);
  const rootKeyValues = (params.rootKeys ?? []).map((r) => r.value);
  const users = [...userValues, ...rootKeyValues];

  const cursor =
    params.cursorTime !== null && params.cursorAuditId !== null
      ? { time: params.cursorTime, auditId: params.cursorAuditId }
      : null;

  const retentionDays =
    workspace.features.auditLogRetentionDays ?? workspace.plan === "free" ? 30 : 90;
  const retentionCutoffUnixMilli = Date.now() - retentionDays * 24 * 60 * 60 * 1000;

  return db.query.auditLogBucket.findFirst({
    where: (table, { eq, and }) =>
      and(eq(table.workspaceId, workspace.id), eq(table.name, params.bucket)),
    with: {
      logs: {
        where: (table, { and, or, inArray, between, lt, eq }) =>
          and(
            events.length > 0 ? inArray(table.event, events) : undefined,
            between(
              table.createdAt,
              Math.max(params.startTime ?? retentionCutoffUnixMilli, retentionCutoffUnixMilli),
              params.endTime ?? Date.now(),
            ),
            users.length > 0 ? inArray(table.actorId, users) : undefined,
            cursor
              ? or(
                  lt(table.time, cursor.time),
                  and(eq(table.time, cursor.time), lt(table.id, cursor.auditId)),
                )
              : undefined,
          ),
        with: {
          targets: true,
        },
        orderBy: (table, { desc }) => desc(table.id),
        limit: params.limit + 1,
      },
    },
  });
};

export function omitLastItemForPagination(items: AuditLogWithTargets[], limit: number) {
  // If we got limit + 1 results, there are more pages
  const hasMore = items.length > limit;
  // Remove the extra item we used to check for more pages
  const slicedItems = hasMore ? items.slice(0, -1) : items;
  return { slicedItems, hasMore };
}

export const fetchUsersFromLogs = async (
  logs: AuditLogWithTargets[],
): Promise<Record<string, User>> => {
  try {
    // Get unique user IDs from logs
    const userIds = [...new Set(logs.filter((l) => l.actorType === "user").map((l) => l.actorId))];

    // Fetch all users in parallel
    const users = await Promise.all(
      userIds.map((userId) => auth.getUser(userId).catch(() => null)),
    );

    // Convert array to record object
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
