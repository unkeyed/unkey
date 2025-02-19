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

export type TimeseriesConfig = {
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
