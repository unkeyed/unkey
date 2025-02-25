import { type Workspace, db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { type User, clerkClient } from "@clerk/nextjs/server";
import { TRPCError } from "@trpc/server";
import type {
  SelectAuditLog,
  SelectAuditLogTarget,
} from "@unkey/db/src/schema";
import { z } from "zod";

export const DEFAULT_BUCKET_NAME = "unkey_mutations";
export type AuditLogWithTargets = SelectAuditLog & {
  targets: Array<SelectAuditLogTarget>;
};
export const getAuditLogsInput = z.object({
  bucketName: z.string().default(DEFAULT_BUCKET_NAME),
  events: z.array(z.string()).default([]),
  users: z.array(z.string()).default([]),
  rootKeys: z.array(z.string()).default([]),
  cursor: z.string().nullish(),
  limit: z.number().min(1).max(100).default(50).optional(),
  startTime: z.number().nullish(),
  endTime: z.number().nullish(),
});

export const fetchAuditLog = rateLimitedProcedure(ratelimit.update)
  .input(getAuditLogsInput)
  .query(async ({ ctx, input }) => {
    const {
      bucketName,
      events,
      users,
      rootKeys,
      cursor,
      limit,
      endTime,
      startTime,
    } = input;

    const selectedActorIds = [...rootKeys, ...users];

    const result = await queryAuditLogs(
      {
        cursor,
        users: selectedActorIds,
        bucketName,
        endTime,
        startTime,
        events,
        limit,
      },
      ctx.workspace
    );

    if (!result) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Audit log bucket not found",
      });
    }

    const { slicedItems, hasMore } = omitLastItemForPagination(
      result.logs,
      limit
    );
    const uniqueUsers = await fetchUsersFromLogs(slicedItems);

    const items = slicedItems.map((l) => {
      const user = uniqueUsers[l.actorId];
      return {
        user: user
          ? {
              username: user.username,
              firstName: user.firstName,
              lastName: user.lastName,
              imageUrl: user.imageUrl,
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
      items,
      nextCursor:
        hasMore && items.length > 0
          ? items[items.length - 1].auditLog.id
          : null,
    };
  });

export type QueryOptions = Omit<z.infer<typeof getAuditLogsInput>, "rootKeys">;

export const queryAuditLogs = async (
  options: QueryOptions,
  workspace: Workspace
) => {
  const {
    bucketName,
    events = [],
    startTime,
    endTime,
    users = [],
    cursor,
    limit = DEFAULT_FETCH_COUNT,
  } = options;

  const retentionDays =
    workspace.features.auditLogRetentionDays ?? workspace.plan === "free"
      ? 30
      : 90;
  const retentionCutoffUnixMilli =
    Date.now() - retentionDays * 24 * 60 * 60 * 1000;

  return db.query.auditLogBucket.findFirst({
    where: (table, { eq, and }) =>
      and(eq(table.workspaceId, workspace.id), eq(table.name, bucketName)),
    with: {
      logs: {
        where: (table, { and, inArray, between, lt }) =>
          and(
            events.length > 0 ? inArray(table.event, events) : undefined,
            between(
              table.createdAt,
              Math.max(
                startTime ?? retentionCutoffUnixMilli,
                retentionCutoffUnixMilli
              ),
              endTime ?? Date.now()
            ),
            users.length > 0 ? inArray(table.actorId, users) : undefined,
            cursor ? lt(table.id, cursor) : undefined
          ),
        with: {
          targets: true,
        },
        orderBy: (table, { desc }) => desc(table.id),
        limit: limit + 1,
      },
    },
  });
};

export function omitLastItemForPagination(
  items: AuditLogWithTargets[],
  limit = DEFAULT_FETCH_COUNT
) {
  // If we got limit + 1 results, there are more pages
  const hasMore = items.length > limit;
  // Remove the extra item we used to check for more pages
  const slicedItems = hasMore ? items.slice(0, -1) : items;
  return { slicedItems, hasMore };
}

export const fetchUsersFromLogs = async (
  logs: AuditLogWithTargets[]
): Promise<Record<string, User>> => {
  try {
    // Get unique user IDs from logs
    const userIds = [
      ...new Set(
        logs.filter((l) => l.actorType === "user").map((l) => l.actorId)
      ),
    ];

    // Fetch all users in parallel
    const users = await Promise.all(
      userIds.map((userId) =>
        clerkClient.users.getUser(userId).catch(() => null)
      )
    );

    // Convert array to record object
    return users.reduce((acc, user) => {
      if (user) {
        acc[user.id] = user;
      }
      return acc;
    }, {} as Record<string, User>);
  } catch (error) {
    console.error("Error fetching users:", error);
    return {};
  }
};
