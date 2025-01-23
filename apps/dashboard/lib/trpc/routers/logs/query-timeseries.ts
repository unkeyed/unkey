import { queryTimeseriesPayload } from "@/app/(app)/logs-v2/components/charts/query-timeseries.schema";
import { HOUR_IN_MS, WEEK_IN_MS } from "@/app/(app)/logs-v2/constants";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import type { LogsTimeseriesParams } from "@unkey/clickhouse/src/logs";
import type { z } from "zod";
import { getTimestampFromRelative } from "./utils/getTimestampFromRelative";

export const queryTimeseries = rateLimitedProcedure(ratelimit.update)
  .input(queryTimeseriesPayload)
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

    const { params: transformedInputs, granularity } = transformFilters(input);
    const result = await clickhouse.api.timeseries[granularity]({
      ...transformedInputs,
      workspaceId: workspace.id,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }
    return { timeseries: result.val, granularity };
  });

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

export function transformFilters(params: z.infer<typeof queryTimeseriesPayload>): {
  params: Omit<LogsTimeseriesParams, "workspaceId">;
  granularity: TimeseriesGranularity;
} {
  let timeConfig: TimeseriesConfig;

  if (params.since !== "") {
    const startTime = getTimestampFromRelative(params.since);
    const endTime = Date.now();
    timeConfig = getTimeseriesGranularity(startTime, endTime);
  } else {
    timeConfig = getTimeseriesGranularity(params.startTime, params.endTime);
  }

  return {
    params: {
      startTime: timeConfig.startTime,
      endTime: timeConfig.endTime,
      hosts: params.host?.filters.map((f) => f.value) || [],
      methods: params.method?.filters.map((f) => f.value) || [],
      paths:
        params.path?.filters.map((f) => ({
          operator: f.operator,
          value: f.value,
        })) || [],
      statusCodes: params.status?.filters.map((f) => f.value) || [],
    },
    granularity: timeConfig.granularity,
  };
}
