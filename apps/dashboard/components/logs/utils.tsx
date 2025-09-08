import type { CompoundTimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { format } from "date-fns";
import type { TimeseriesData } from "./overview-charts/types";

// Default time buffer for granularity fallbacks (1 minute)
const DEFAULT_TIME_BUFFER_MS = 60_000;

// Helper function to safely convert local Granularity to CompoundTimeseriesGranularity
const getGranularityBuffer = (granularity?: CompoundTimeseriesGranularity): number => {
  if (!granularity) {
    return DEFAULT_TIME_BUFFER_MS; // 1 minute fallback
  }
  try {
    return getTimeBufferForGranularity(granularity);
  } catch {
    return DEFAULT_TIME_BUFFER_MS; // 1 minute fallback
  }
};

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

/**
 * Formats tooltip interval for chart tooltips with timezone and DST handling
 *
 * @param payloadTimestamp - The current timestamp from the tooltip payload
 * @param data - The chart data array
 * @param granularity - Optional granularity for time calculations
 * @param timestampToIndexMap - Map for O(1) timestamp lookups
 * @returns JSX element with formatted interval or empty string if invalid
 */
export const formatTooltipInterval = (
  payloadTimestamp: number | string | undefined,
  data: TimeseriesData[],
  granularity?: CompoundTimeseriesGranularity,
  timestampToIndexMap?: Map<number, number>,
) => {
  if (!payloadTimestamp) {
    return "";
  }

  // Handle missing data
  if (!data?.length) {
    return (
      <div className="px-4">
        <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
          {formatTooltipTimestamp(payloadTimestamp, granularity, data)}
        </span>
      </div>
    );
  }

  // Handle single timestamp case
  if (data.length === 1) {
    return (
      <div className="px-4">
        <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
          {formatTooltipTimestamp(payloadTimestamp, granularity, data)}
        </span>
      </div>
    );
  }

  // Normalize timestamps to numeric for robust comparison
  const currentTimestampNumeric =
    typeof payloadTimestamp === "number" ? payloadTimestamp : +new Date(payloadTimestamp);

  // Find position in the data array using O(1) map lookup or fallback to linear search
  const currentIndex = timestampToIndexMap
    ? (timestampToIndexMap.get(currentTimestampNumeric) ?? -1)
    : data.findIndex((item) => {
        const itemTimestamp =
          typeof item.originalTimestamp === "number"
            ? item.originalTimestamp
            : +new Date(item.originalTimestamp);
        return itemTimestamp === currentTimestampNumeric;
      });

  // If not found, fallback to single timestamp display
  if (currentIndex === -1) {
    return (
      <div className="px-4">
        <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
          {formatTooltipTimestamp(currentTimestampNumeric, granularity, data)}
        </span>
      </div>
    );
  }

  // Compute interval end timestamp
  let intervalEndTimestamp: number;

  // If this is the last item, compute interval end using granularity
  if (currentIndex >= data.length - 1) {
    const inferredGranularityMs = granularity
      ? getGranularityBuffer(granularity)
      : data.length > 1
        ? Math.abs(
            (typeof data[1].originalTimestamp === "number"
              ? data[1].originalTimestamp
              : +new Date(data[1].originalTimestamp)) -
              (typeof data[0].originalTimestamp === "number"
                ? data[0].originalTimestamp
                : +new Date(data[0].originalTimestamp)),
          )
        : DEFAULT_TIME_BUFFER_MS; // 1 minute fallback
    intervalEndTimestamp = currentTimestampNumeric + inferredGranularityMs;
  } else {
    // Use next data point's timestamp
    const nextPoint = data[currentIndex + 1];
    if (!nextPoint?.originalTimestamp) {
      // Fallback to single timestamp if next point is invalid
      return (
        <div className="px-4">
          <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
            {formatTooltipTimestamp(currentTimestampNumeric, granularity, data)}
          </span>
        </div>
      );
    }
    intervalEndTimestamp =
      typeof nextPoint.originalTimestamp === "number"
        ? nextPoint.originalTimestamp
        : +new Date(nextPoint.originalTimestamp);
  }

  // Format both timestamps using normalized numeric values
  const formattedCurrentTimestamp = formatTooltipTimestamp(
    currentTimestampNumeric,
    granularity,
    data,
  );
  const formattedNextTimestamp = formatTooltipTimestamp(intervalEndTimestamp, granularity, data);

  // Get timezone abbreviation from the actual point date for correct DST handling
  const pointDate = new Date(currentTimestampNumeric);
  const timezone =
    new Intl.DateTimeFormat("en-US", {
      timeZoneName: "short",
    })
      .formatToParts(pointDate)
      .find((part) => part.type === "timeZoneName")?.value || "";

  // Return formatted interval with timezone info
  return (
    <div className="px-4">
      <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
        {formattedCurrentTimestamp} - {formattedNextTimestamp} ({timezone})
      </span>
    </div>
  );
};
