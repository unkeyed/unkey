import type { CompoundTimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { format } from "date-fns";
import type { TimeseriesData } from "./overview-charts/types";

/**
 * Helper function to format tooltip timestamps based on granularity and data span
 */
export const formatTooltipTimestamp = (
  timestamp: number | string,
  granularity?: CompoundTimeseriesGranularity,
  data?: TimeseriesData[],
): string => {
  // Parse timestamp to number and handle microsecond timestamps
  const timestampNum = typeof timestamp === "string" ? Number.parseFloat(timestamp) : timestamp;

  // Detect microseconds by checking if value has 16 digits or is > 1e13
  const isMicroseconds = timestampNum > 1e13 || timestampNum.toString().length === 16;

  // Convert microseconds to milliseconds if needed
  const timestampMs = isMicroseconds ? Math.floor(timestampNum / 1000) : timestampNum;

  const date = new Date(timestampMs);

  // If we have data, check if it spans multiple days
  if (data && data.length > 1) {
    const firstDay = new Date(data[0].originalTimestamp);
    const lastDay = new Date(data[data.length - 1].originalTimestamp);

    // Check if the data spans multiple calendar days
    const firstDayStr = firstDay.toDateString();
    const lastDayStr = lastDay.toDateString();

    if (firstDayStr !== lastDayStr) {
      // Data spans multiple days, always show date + time
      return format(date, "MMM dd, h:mm a");
    }
  }

  // For granularities less than 12 hours on same day, show only time
  if (
    granularity &&
    [
      "perMinute",
      "per5Minutes",
      "per15Minutes",
      "per30Minutes",
      "perHour",
      "per2Hours",
      "per4Hours",
      "per6Hours",
    ].includes(granularity)
  ) {
    return format(date, "h:mm a");
  }

  // For granularities 12 hours or more, show date + time
  return format(date, "MMM dd, h:mm a");
};
