import { clickhouse } from "@/lib/clickhouse";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

/**
 * Maps each chart window in hours to its ClickHouse bucket width in minutes.
 * For example, a 24-hour window uses one 30-minute bucket per point. These widths keep each
 * response near 40 to 60 points while shorter windows retain more detail.
 */
const bucketMinutesByWindowHours = {
  1: 1,
  24: 30,
  168: 240,
} as const;

export const getLogdrainMetrics = workspaceProcedure
  .input(
    z.object({
      drainId: z.string().min(1),
      hours: z.union([z.literal(1), z.literal(24), z.literal(168)]),
    }),
  )
  .query(async ({ ctx, input }) => {
    const drain = await db.query.logdrains.findFirst({
      where: and(
        eq(schema.logdrains.id, input.drainId),
        eq(schema.logdrains.workspaceId, ctx.workspace.id),
      ),
      columns: { id: true },
    });
    if (!drain) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Log drain not found" });
    }

    const endMs = Date.now();
    const result = await clickhouse.logdrains.metrics({
      workspaceId: ctx.workspace.id,
      drainId: input.drainId,
      startMs: endMs - input.hours * 60 * 60 * 1000,
      endMs,
      bucketMinutes: bucketMinutesByWindowHours[input.hours],
    });
    if (!result.val) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch log drain metrics",
      });
    }

    return { series: result.val, bucketMinutes: bucketMinutesByWindowHours[input.hours] };
  });
