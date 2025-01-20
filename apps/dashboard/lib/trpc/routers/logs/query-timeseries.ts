import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { logsTimeseriesParams } from "@unkey/clickhouse/src/logs";

export const queryTimeseries = rateLimitedProcedure(ratelimit.update)
  .input(logsTimeseriesParams.omit({ workspaceId: true }))
  .query(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          //TODO: change error message later
          message:
            "Failed to retrieve timeseries analytics due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    const { startTime, endTime, granularity } = getTimeseriesGranularity(
      input.startTime,
      input.endTime,
    );

    const result = await clickhouse.api.timeseries[granularity]({
      ...input,
      startTime,
      endTime,
      workspaceId: workspace.id,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }
    return result.val;
  });

export const HOUR_IN_MS = 60 * 60 * 1000;
const DAY_IN_MS = 24 * HOUR_IN_MS;
const WEEK_IN_MS = 7 * DAY_IN_MS;

export type TimeseriesGranularity = "perMinute" | "perHour" | "perDay";
type TimeseriesConfig = {
  granularity: TimeseriesGranularity;
  startTime: number;
  endTime: number;
};

export const getTimeseriesGranularity = (
  startTime?: number | null,
  endTime?: number | null,
): TimeseriesConfig => {
  const now = Date.now();

  // If both of them are missing fallback to perMinute and fetch lastHour to show latest
  if (!startTime && !endTime) {
    return {
      granularity: "perMinute",
      startTime: now - HOUR_IN_MS,
      endTime: now,
    };
  }

  // Set default end time if missing
  const effectiveEndTime = endTime ?? now;
  // Set default start time if missing (last hour)
  const effectiveStartTime = startTime ?? effectiveEndTime - HOUR_IN_MS;
  const timeRange = effectiveEndTime - effectiveStartTime;
  let granularity: TimeseriesGranularity;

  if (timeRange > WEEK_IN_MS) {
    // > 7 days
    granularity = "perDay";
  } else if (timeRange > HOUR_IN_MS) {
    // > 1 hour
    granularity = "perHour";
  } else {
    // <= 1 hour
    granularity = "perMinute";
  }

  return {
    granularity,
    startTime: effectiveStartTime,
    endTime: effectiveEndTime,
  };
};
