import { clickhouse } from "@/lib/clickhouse";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../trpc";
import { getAlertInput } from "./schemas";

const dayMs = 24 * 60 * 60 * 1000;
const hourMs = 60 * 60 * 1000;

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
    const result = await clickhouse.alerts.timeseries({
      workspaceId: ctx.workspace.id,
      appId: alert.appId,
      environmentId: alert.environmentId,
      metric: alert.metric,
      startMs: alert.firedAt - dayMs,
      endMs: Math.min(now, (alert.resolvedAt ?? now) + hourMs),
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
      upperBound: alert.baselineMean + alert.thresholdSigma * alert.baselineStddev,
      windowStart: alert.windowStart,
      windowEnd: alert.windowEnd,
    };
  });
