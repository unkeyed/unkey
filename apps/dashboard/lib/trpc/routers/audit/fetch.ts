import { DEFAULT_FETCH_COUNT } from "@/app/(app)/audit/[bucket]/components/table/constants";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const getAuditLogsInput = z.object({
  bucket: z.string().nullable(),
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
});

export const fetchAuditLog = rateLimitedProcedure(ratelimit.update)
  .input(getAuditLogsInput)
  .query(async ({ ctx, input }) => {
    const { bucket, events, users, rootKeys, cursor, limit } = input;

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

    const retentionDays =
      workspace.features.auditLogRetentionDays ?? (workspace.plan === "free" ? 30 : 90);

    const retentionCutoffUnixMilli = Date.now() - retentionDays * 24 * 60 * 60 * 1000;

    const selectedActorIds = [...rootKeys, ...users];

    const logs = await db.query.auditLogBucket.findFirst({
      where: (table, { eq, and }) =>
        and(eq(table.workspaceId, workspace.id), eq(table.name, bucket ?? "unkey_mutations")),
      with: {
        logs: {
          where: (table, { and, inArray, gte, lt }) =>
            and(
              events.length > 0 ? inArray(table.event, events) : undefined,
              gte(table.createdAt, retentionCutoffUnixMilli),
              selectedActorIds.length > 0 ? inArray(table.actorId, selectedActorIds) : undefined,
              cursor ? lt(table.time, cursor.time) : undefined,
            ),
          with: {
            targets: true,
          },
          orderBy: (table, { desc }) => desc(table.time),
          limit: limit + 1, // Fetch one extra to determine if there are more results
        },
      },
    });

    if (!logs) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Audit log bucket not found",
      });
    }

    const items = logs.logs;
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
