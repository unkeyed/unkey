import { clickhouse } from "@/lib/clickhouse";
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

    const result = await clickhouse.alerts.series({
      workspaceId: ctx.workspace.id,
      appId: input.appId,
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
