import { clickhouse } from "@/lib/clickhouse";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../trpc";
import { getAlertInput } from "./schemas";
import { alertTimeseriesRange } from "./timeseries-range";

export const getAlertTimeseries = workspaceProcedure
  .input(getAlertInput)
  .query(async ({ ctx, input }) => {
    const alert = await db.query.alertEvents.findFirst({
      where: and(
        eq(schema.alertEvents.id, input.alertId),
        eq(schema.alertEvents.workspaceId, ctx.workspace.id),
      ),
      columns: {
        appId: true,
        environmentId: true,
        metric: true,
        firedAt: true,
        resolvedAt: true,
        baselineMean: true,
        baselineStddev: true,
        thresholdSigma: true,
        windowStart: true,
        windowEnd: true,
      },
    });
    if (!alert) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Alert not found" });
    }

    const now = Date.now();
    const range = alertTimeseriesRange({
      firedAt: alert.firedAt,
      resolvedAt: alert.resolvedAt,
      now,
    });
    const result = await clickhouse.alerts.timeseries({
      workspaceId: ctx.workspace.id,
      appId: alert.appId,
      environmentId: alert.environmentId,
      metric: alert.metric,
      ...range,
    });
    if (!result.val) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch alert timeseries",
      });
    }

    return {
      buckets: result.val,
      baselineMean: alert.baselineMean,
      lowerBound: Math.max(0, alert.baselineMean - alert.thresholdSigma * alert.baselineStddev),
      upperBound:
        alert.metric === "memory_utilization"
          ? 0.9
          : alert.metric === "oom_killed" || alert.metric === "crash_loop"
            ? 1
            : alert.baselineMean + alert.thresholdSigma * alert.baselineStddev,
      windowStart: alert.windowStart,
      windowEnd: alert.windowEnd,
    };
  });
