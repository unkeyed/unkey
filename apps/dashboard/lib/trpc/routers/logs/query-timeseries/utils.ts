import type { queryTimeseriesPayload } from "@/app/(app)/logs/components/charts/query-timeseries.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { LogsTimeseriesParams } from "@unkey/clickhouse/src/logs";
import type { z } from "zod";
import { DAY_IN_MS, HOUR_IN_MS } from "./constants";

export type TimeseriesGranularity =
  | "perMinute"
  | "per5Minutes"
  | "per15Minutes"
  | "per30Minutes"
  | "perHour"
  | "per2Hours"
  | "per4Hours"
  | "per6Hours"
  | "perDay";

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
  if (timeRange > DAY_IN_MS * 7) {
    granularity = "perDay";
  } else if (timeRange > DAY_IN_MS * 3) {
    granularity = "per6Hours";
  } else if (timeRange > HOUR_IN_MS * 24) {
    granularity = "per4Hours";
  } else if (timeRange > HOUR_IN_MS * 16) {
    granularity = "per2Hours";
  } else if (timeRange > HOUR_IN_MS * 12) {
    granularity = "perHour";
  } else if (timeRange > HOUR_IN_MS * 8) {
    granularity = "per30Minutes";
  } else if (timeRange > HOUR_IN_MS * 6) {
    granularity = "per30Minutes";
  } else if (timeRange > HOUR_IN_MS * 4) {
    granularity = "per15Minutes";
  } else if (timeRange > HOUR_IN_MS * 2) {
    granularity = "per5Minutes";
  } else {
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
