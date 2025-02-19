import type { RatelimitQueryTimeseriesPayload } from "@/app/(app)/ratelimits/[namespaceId]/logs/components/charts/query-timeseries.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { RatelimitLogsTimeseriesParams } from "@unkey/clickhouse/src/ratelimits";
import { HOUR_IN_MS, MONTH_IN_MS, WEEK_IN_MS } from "./constants";

export type TimeseriesGranularity = "perMinute" | "perHour" | "perDay" | "perMonth";

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
  if (timeRange > MONTH_IN_MS) {
    // > 30 days
    granularity = "perMonth";
  } else if (timeRange > WEEK_IN_MS) {
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

export function transformRatelimitFilters(params: RatelimitQueryTimeseriesPayload): {
  params: Omit<RatelimitLogsTimeseriesParams, "workspaceId" | "namespaceId">;
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
      identifiers:
        params.identifiers?.filters.map((f) => ({
          operator: f.operator,
          value: f.value,
        })) || null,
    },
    granularity: timeConfig.granularity,
  };
}
