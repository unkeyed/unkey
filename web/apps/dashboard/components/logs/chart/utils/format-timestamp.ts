import type { TimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { type TimestampStyle, formatTimestamp } from "@unkey/ui";

export const formatTimestampLabel = (timestamp: string | number | Date) => {
  const isNumericString = typeof timestamp === "string" && /^\d+$/.test(timestamp);
  const input = isNumericString ? Number(timestamp) : timestamp;
  const date = input instanceof Date ? input : new Date(input);
  return formatTimestamp(date, "MMM d, h:mm a").toUpperCase();
};

export const formatTimestampForChart = (
  value: string | number,
  granularity: TimeseriesGranularity,
) => formatTimestamp(new Date(value), TICK_STYLE[granularity]);

const TICK_STYLE: Record<TimeseriesGranularity, TimestampStyle> = {
  perMinute: "h:mm a",
  per5Minutes: "h:mm a",
  per15Minutes: "h:mm a",
  per30Minutes: "h:mm a",
  perHour: "MMM d, h:mm a",
  per2Hours: "MMM d, h:mm a",
  per4Hours: "MMM d, h:mm a",
  per6Hours: "MMM d, h:mm a",
  per12Hours: "MMM d, h:mm a",
  perDay: "MMM d",
  per3Days: "MMM d",
  perWeek: "MMM d",
  perMonth: "MMM yyyy",
  perQuarter: "MMM yyyy",
};
