import { auditQueryLogsPayload } from "@/app/(app)/[workspaceSlug]/audit/components/table/query-logs.schema";
import { auth } from "@/lib/auth/server";
import type { User } from "@/lib/auth/types";
import { type Workspace, db } from "@/lib/db";
import { z } from "zod";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "../../trpc";
import { type AuditLogWithTargets, type AuditQueryLogsParams, auditLog } from "./schema";
import { transformFilters } from "./utils";

const AuditLogsResponse = z.object({
  auditLogs: z.array(auditLog),
  hasMore: z.boolean(),
  nextCursor: z.number().optional(),
});

type AuditLogsResponse = z.infer<typeof AuditLogsResponse>;

export const fetchAuditLog = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(auditQueryLogsPayload)
  .output(AuditLogsResponse)
  .query(async ({ ctx, input }) => {
    const params = transformFilters(input);
    const logs = await queryAuditLogs(params, ctx.workspace);

    const { slicedItems, hasMore } = omitLastItemForPagination(logs, params.limit);
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
      nextCursor: hasMore && items.length > 0 ? items[items.length - 1].auditLog.time : undefined,
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

  const cursor = params.cursor;

  const retentionDays =
    (workspace.features.auditLogRetentionDays ?? workspace.plan === "free") ? 30 : 90;
  const retentionCutoffUnixMilli = Date.now() - retentionDays * 24 * 60 * 60 * 1000;

  // By default we need the last "50"(LIMIT) records.
  const hasTimeFilter = params.startTime !== undefined || params.endTime !== undefined;

  const logs = await db.query.auditLog.findMany({
    where: (table, { eq, and, inArray, between, lt, gte }) =>
      and(
        eq(table.workspaceId, workspace.id),
        eq(table.bucket, params.bucket),
        events.length > 0 ? inArray(table.event, events) : undefined,
        // Apply time filters only if explicitly provided, otherwise just respect retention period
        hasTimeFilter
          ? between(
              table.time,
              Math.max(params.startTime ?? retentionCutoffUnixMilli, retentionCutoffUnixMilli),
              params.endTime ?? Date.now(),
            )
          : gte(table.time, retentionCutoffUnixMilli), // Only enforce retention period
        users.length > 0 ? inArray(table.actorId, users) : undefined,
        cursor ? lt(table.time, cursor) : undefined,
      ),
    with: {
      targets: true,
    },
    orderBy: (table, { desc }) => desc(table.time),
    limit: params.limit + 1,
  });
  return logs;
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
