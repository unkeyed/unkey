import { DEFAULT_FETCH_COUNT } from "@/app/(app)/audit/[bucket]/components/table/constants";
import { type Workspace, db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { SelectAuditLog, SelectAuditLogTarget } from "@unkey/db/src/schema";
import { z } from "zod";

export const DEFAULT_BUCKET_NAME = "unkey_mutations";
export type AuditLogWithTargets = SelectAuditLog & {
  targets: Array<SelectAuditLogTarget>;
};
export const getAuditLogsInput = z.object({
  bucket: z.string().default(DEFAULT_BUCKET_NAME),
  events: z.array(z.string()).default([]),
  users: z.array(z.string()).default([]),
  rootKeys: z.array(z.string()).default([]),
  cursor: z.string().nullish(),
  limit: z.number().min(1).max(100).default(DEFAULT_FETCH_COUNT).optional(),
  startTime: z.number().nullish(),
  endTime: z.number().nullish(),
});

export const fetchAuditLog = rateLimitedProcedure(ratelimit.update)
  .input(getAuditLogsInput)
  .query(async ({ ctx, input }) => {
    const {
      bucket,
      events,
      users,
      rootKeys,
      cursor,
      limit,
      endTime,
      startTime,
    } = input;

    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve workspace logs due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    const selectedActorIds = [...rootKeys, ...users];

    const result = await queryAuditLogs(
      {
        cursor,
        users: selectedActorIds,
        bucket,
        endTime,
        startTime,
        events,
        limit,
      },
      workspace
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

    return {
      items: slicedItems,
      nextCursor:
        hasMore && slicedItems.length > 0
          ? slicedItems[slicedItems.length - 1].id
          : null,
    };
  });

export type QueryOptions = Omit<z.infer<typeof getAuditLogsInput>, "rootKeys">;

export const queryAuditLogs = async (
  options: QueryOptions,
  workspace: Workspace
) => {
  const {
    bucket,
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
      and(eq(table.workspaceId, workspace.id), eq(table.name, bucket)),
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
