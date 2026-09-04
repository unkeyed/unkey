import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../trpc";
import { alertSeriesInput } from "./schemas";
import { alertSeriesRange } from "./series-range";

export const getAlertSeries = workspaceProcedure
  .input(alertSeriesInput)
  .query(async ({ ctx, input }) => {
    const range = alertSeriesRange({ ...input, now: Date.now() });
    if (range.startMs >= range.endMs) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "The selected range has no closed metric buckets",
      });
    }

    const app = await db.query.apps.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.appId), eq(table.workspaceId, ctx.workspace.id)),
      columns: { createdAt: true },
    });
    if (!app) {
      throw new TRPCError({ code: "NOT_FOUND", message: "App not found" });
    }

    const result = await clickhouse.alerts.series({
      workspaceId: ctx.workspace.id,
      appId: input.appId,
      appCreatedAtMs: app.createdAt,
      environmentId: input.environmentId,
      metric: input.metric,
      resolution: input.resolution,
      ...range,
    });
    if (!result.val) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch anomaly metric series",
        cause: result.err,
      });
    }

    return {
      buckets: result.val,
      startMs: range.startMs,
      endMs: range.endMs,
    };
  });
