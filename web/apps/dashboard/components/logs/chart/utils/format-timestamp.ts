import type { TimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { type TimestampStyle, formatTimestamp } from "@unkey/ui";

export const formatTimestampLabel = (timestamp: string | number | Date) => {
  const isNumericString = typeof timestamp === "string" && /^\d+$/.test(timestamp);
  const input = isNumericString ? Number(timestamp) : timestamp;
  const date = input instanceof Date ? input : new Date(input);
  return formatTimestamp(date, "dateTime").toUpperCase();
};

export const formatTimestampForChart = (
  value: string | number,
  granularity: TimeseriesGranularity,
) => formatTimestamp(new Date(value), TICK_STYLE[granularity]);

const TICK_STYLE: Record<TimeseriesGranularity, TimestampStyle> = {
  perMinute: "time",
  per5Minutes: "time",
  per15Minutes: "time",
  per30Minutes: "time",
  perHour: "dateTime",
  per2Hours: "dateTime",
  per4Hours: "dateTime",
  per6Hours: "dateTime",
  per12Hours: "dateTime",
  perDay: "monthDay",
  per3Days: "monthDay",
  perWeek: "monthDay",
  perMonth: "monthYear",
  perQuarter: "monthYear",
};
