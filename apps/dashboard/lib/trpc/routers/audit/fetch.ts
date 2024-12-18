import { DEFAULT_FETCH_COUNT } from "@/app/(app)/audit/[bucket]/components/table/constants";
import { type Workspace, db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const getAuditLogsInput = z.object({
  bucket: z.string().default("unkey_mutations"),
  events: z.array(z.string()).default([]),
  users: z.array(z.string()).default([]),
  rootKeys: z.array(z.string()).default([]),
  cursor: z
    .object({
      time: z.number(),
      id: z.string(),
    })
    .optional(),
  limit: z.number().min(1).max(100).default(DEFAULT_FETCH_COUNT),
  startTime: z.number().nullable(),
  endTime: z.number().nullable(),
});

export const fetchAuditLog = rateLimitedProcedure(ratelimit.update)
  .input(getAuditLogsInput)
  .query(async ({ ctx, input }) => {
    const { bucket, events, users, rootKeys, cursor, limit, endTime, startTime } = input;

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
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    const selectedActorIds = [...rootKeys, ...users];

    const result = await queryAuditLogs(
      {
        cursor,
        actors: selectedActorIds,
        bucket,
        endTime,
        startTime,
        events,
        limit,
      },
      workspace,
    );

    if (!result) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Audit log bucket not found",
      });
    }

    const items = result.logs;
    // If we got limit + 1 results, there are more pages
    const hasMore = items.length > limit;
    // Remove the extra item we used to check for more pages
    const slicedItems = hasMore ? items.slice(0, -1) : items;

    return {
      items: slicedItems,
      nextCursor:
        hasMore && slicedItems.length > 0
          ? {
              time: slicedItems[slicedItems.length - 1].time,
              id: slicedItems[slicedItems.length - 1].id,
            }
          : undefined,
    };
  });

type QueryOptions = {
  events: string[];
  actors: string[];
  startTime: number | null;
  endTime: number | null;
  bucket: string;
  limit?: number;
  cursor?: {
    time: number;
    id: string;
  };
};

export const queryAuditLogs = async (options: QueryOptions, workspace: Workspace) => {
  const {
    bucket,
    events = [],
    startTime,
    endTime,
    actors = [],
    cursor,
    limit = DEFAULT_FETCH_COUNT,
  } = options;

  const retentionDays =
    workspace.features.auditLogRetentionDays ?? workspace.plan === "free" ? 30 : 90;
  const retentionCutoffUnixMilli = Date.now() - retentionDays * 24 * 60 * 60 * 1000;

  return db.query.auditLogBucket.findFirst({
    where: (table: any, { eq, and }) =>
      and(eq(table.workspaceId, workspace.id), eq(table.name, bucket)),
    with: {
      logs: {
        where: (table: any, { and, inArray, between, lt }) =>
          and(
            events.length > 0 ? inArray(table.event, events) : undefined,
            between(
              table.createdAt,
              Math.max(startTime ?? retentionCutoffUnixMilli, retentionCutoffUnixMilli),
              endTime ?? Date.now(),
            ),
            actors.length > 0 ? inArray(table.actorId, actors) : undefined,
            cursor ? lt(table.time, cursor.time) : undefined,
          ),
        with: {
          targets: true,
        },
        orderBy: (table: any, { desc }) => desc(table.time),
        limit: cursor ? limit + 1 : limit, // Fetch extra item only when paginating
      },
    },
  });
};
