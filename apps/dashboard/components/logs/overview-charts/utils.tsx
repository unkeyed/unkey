import type { CompoundTimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { format } from "date-fns";
import { parseTimestamp } from "../parseTimestamp";
import type { TimeseriesData } from "./types";

// Default time buffer for granularity fallbacks (1 minute)
export const DEFAULT_TIME_BUFFER_MS = 60_000;

// Singleton DateTimeFormat for timezone abbreviation extraction
const TZ_FORMATTER = new Intl.DateTimeFormat("en-US", {
  timeZoneName: "short",
});

// Define types for tooltip payload structure
type TooltipPayloadItem = {
  payload?: {
    originalTimestamp?: string | number | Date;
    [key: string]: unknown;
  };
  [key: string]: unknown;
};

/**
 * Get appropriate formatted time based on granularity (12-hour format without timezone)
 */
function formatTimeForGranularity(date: Date, granularity?: CompoundTimeseriesGranularity): string {
  if (!granularity) {
    return format(date, "h:mma");
  }

  switch (granularity) {
    case "perMinute":
      return format(date, "h:mm:ssa");
    case "per5Minutes":
    case "per15Minutes":
    case "per30Minutes":
      return format(date, "h:mma");
    case "perHour":
    case "per2Hours":
    case "per4Hours":
    case "per6Hours":
    case "per12Hours":
      return format(date, "MMM d h:mma");
    case "perDay":
      return format(date, "MMM d");
    case "per3Days":
      return format(date, "MMM d");
    case "perWeek":
      return format(date, "MMM d");
    case "perMonth":
      return format(date, "MMM yy");
    default:
      return format(date, "h:mma");
  }
}

/**
 * Get timezone abbreviation for a given date
 */
function getTimezoneAbbreviation(date?: Date): string {
  const timezone =
    TZ_FORMATTER.formatToParts(date || new Date()).find((part) => part.type === "timeZoneName")
      ?.value || "";
  return timezone;
}

/**
 * Creates a tooltip formatter that displays time intervals between data points
 * with granularity-aware timestamp formatting
 *
 * @param data - The chart data array containing timestamp information
 * @param timeFormat - Optional custom time format (will be overridden if granularity is provided)
 * @param granularity - Optional granularity to determine appropriate time format
 * @returns A formatter function for use with chart tooltips
 */
export function createTimeIntervalFormatter(
  data?: TimeseriesData[],
  timeFormat = "h:mm a",
  granularity?: CompoundTimeseriesGranularity,
) {
  return (tooltipPayload: TooltipPayloadItem[]) => {
    // Basic validation checks
    if (!tooltipPayload?.[0]?.payload) {
      return "";
    }

    const currentPayload = tooltipPayload[0].payload;
    const currentTimestamp = currentPayload.originalTimestamp;

    // Validate timestamp exists
    if (!currentTimestamp) {
      return "";
    }

    // Use granularity-aware format if available, otherwise use provided timeFormat
    const currentDate = new Date(parseTimestamp(currentTimestamp));
    const formattedCurrentTimestamp = granularity
      ? formatTimeForGranularity(currentDate, granularity)
      : format(currentDate, timeFormat);

    // Precompute timezone abbreviation using the current timestamp for consistency
    const timezoneAbbr = getTimezoneAbbreviation(currentDate);

    // If we don't have necessary data, fallback to displaying just the current point
    if (!currentTimestamp || !data?.length) {
      return (
        <div className="px-4">
          <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
            {formattedCurrentTimestamp} ({timezoneAbbr})
          </span>
        </div>
      );
    }

    // Find position in the data array
    // Normalize both timestamps to numeric epoch values for reliable comparison
    const normalizedCurrentTimestamp = parseTimestamp(currentTimestamp);
    const currentIndex = data.findIndex((item) => {
      if (!item?.originalTimestamp) {
        return false;
      }
      const normalizedItemTimestamp = parseTimestamp(item.originalTimestamp);
      // Use Number.isNaN to handle NaN cases safely
      return (
        !Number.isNaN(normalizedCurrentTimestamp) &&
        !Number.isNaN(normalizedItemTimestamp) &&
        normalizedCurrentTimestamp === normalizedItemTimestamp
      );
    });

    // If this is the last item or not found, just show current timestamp
    if (currentIndex === -1 || currentIndex >= data.length - 1) {
      return (
        <div className="px-4">
          <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
            {formattedCurrentTimestamp} ({timezoneAbbr})
          </span>
        </div>
      );
    }

    // Get the next point in the sequence
    const nextPoint = data[currentIndex + 1];
    if (!nextPoint) {
      return (
        <div>
          <span className="font-mono text-accent-9 text-xs px-4">{formattedCurrentTimestamp}</span>
        </div>
      );
    }

    // Format the next timestamp with the same format
    const nextDate = new Date(parseTimestamp(nextPoint.originalTimestamp));
    const formattedNextTimestamp = granularity
      ? formatTimeForGranularity(nextDate, granularity)
      : format(nextDate, timeFormat);

    // Compute timezone abbreviations for both timestamps to handle DST boundaries
    const startTimezoneAbbr = timezoneAbbr;
    const endTimezoneAbbr = getTimezoneAbbreviation(nextDate);

    // Format timezone display: single if same, or both with arrow if different
    const timezoneDisplay =
      startTimezoneAbbr === endTimezoneAbbr
        ? startTimezoneAbbr
        : `${startTimezoneAbbr} → ${endTimezoneAbbr}`;

    // Return formatted interval with timezone info
    return (
      <div className="px-4">
        <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
          {formattedCurrentTimestamp} - {formattedNextTimestamp} ({timezoneDisplay})
        </span>
      </div>
    );
  };
}
