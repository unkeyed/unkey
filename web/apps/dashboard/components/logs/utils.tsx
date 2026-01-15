import type { TimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { format } from "date-fns";
import type { TimeseriesData } from "./overview-charts/types";
import { parseTimestamp } from "./parse-timestamp";

// Default time buffer for granularity fallbacks (1 minute)
const DEFAULT_TIME_BUFFER_MS = 60_000;

// Singleton DateTimeFormat for timezone abbreviation extraction
const TZ_FORMATTER = new Intl.DateTimeFormat("en-US", {
  timeZoneName: "short",
});

/**
 * Maps schema granularity strings to TimeseriesGranularity format
 * Used to convert granularity values from API responses to the standard format
 */
export const SCHEMA_GRANULARITY_MAP: Record<string, TimeseriesGranularity> = {
  minute: "perMinute",
  fiveMinutes: "per5Minutes",
  fifteenMinutes: "per15Minutes",
  thirtyMinutes: "per30Minutes",
  hour: "perHour",
  twoHours: "per2Hours",
  fourHours: "per4Hours",
  sixHours: "per6Hours",
  twelveHours: "per12Hours",
  day: "perDay",
  threeDays: "per3Days",
  week: "perWeek",
  month: "perMonth",
  quarter: "perQuarter",
} as const;

/**
 * Converts schema granularity string to TimeseriesGranularity format
 * @param schemaGranularity The granularity string from API response
 * @param fallback Fallback granularity if mapping not found (defaults to "perHour")
 * @returns TimeseriesGranularity format
 */
export const mapSchemaGranularity = (
  schemaGranularity: string,
  fallback: TimeseriesGranularity = "perHour",
): TimeseriesGranularity => {
  return SCHEMA_GRANULARITY_MAP[schemaGranularity] || fallback;
};

// Helper function to safely convert local Granularity to TimeseriesGranularity
const getGranularityBuffer = (granularity?: TimeseriesGranularity): number => {
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
  granularity?: TimeseriesGranularity,
  data?: TimeseriesData[],
): string => {
  // Handle null/undefined early
  if (timestamp == null) {
    return "";
  }

  // Parse timestamp using shared helper
  const timestampMs = parseTimestamp(timestamp);

  // Validate parsed timestamp
  if (timestampMs === 0 && timestamp !== 0 && timestamp !== "0") {
    return "";
  }

  // Check for NaN/Infinity values
  if (!Number.isFinite(timestampMs)) {
    return "";
  }

  const date = new Date(timestampMs);

  // If we have data, check if it spans multiple days
  if (data && data.length > 1) {
    const firstDay = new Date(parseTimestamp(data[0].originalTimestamp));
    const lastDay = new Date(parseTimestamp(data[data.length - 1].originalTimestamp));

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
  granularity?: TimeseriesGranularity,
  timestampToIndexMap?: Map<number, number>,
) => {
  if (payloadTimestamp == null) {
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
  const currentTimestampNumeric = parseTimestamp(payloadTimestamp);

  // Find position in the data array using O(1) map lookup or fallback to linear search
  const currentIndex = timestampToIndexMap
    ? (timestampToIndexMap.get(currentTimestampNumeric) ?? -1)
    : data.findIndex((item) => {
        const itemTimestamp = parseTimestamp(item.originalTimestamp);
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
            parseTimestamp(data[1].originalTimestamp) - parseTimestamp(data[0].originalTimestamp),
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
    intervalEndTimestamp = parseTimestamp(nextPoint.originalTimestamp);
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
    TZ_FORMATTER.formatToParts(pointDate).find((part) => part.type === "timeZoneName")?.value || "";

  // Return formatted interval with timezone info
  return (
    <div className="px-4">
      <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
        {formattedCurrentTimestamp} - {formattedNextTimestamp} ({timezone})
      </span>
    </div>
  );
};
