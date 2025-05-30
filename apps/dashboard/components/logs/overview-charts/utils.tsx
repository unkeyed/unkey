import { format } from "date-fns";
import type { TimeseriesData } from "./types";

// Define types for tooltip payload structure
type TooltipPayloadItem = {
  payload?: {
    originalTimestamp?: string | number | Date;
    [key: string]: unknown;
  };
  [key: string]: unknown;
};

/**
 * Creates a tooltip formatter that displays time intervals between data points
 * with a custom timestamp format that matches the bottom axis
 *
 * @param data - The chart data array containing timestamp information
 * @param timeFormat - Optional custom time format (defaults to "HH:mm")
 * @returns A formatter function for use with chart tooltips
 */
export function createTimeIntervalFormatter(data?: TimeseriesData[], timeFormat = "HH:mm") {
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

    // Format timestamp with the provided format (defaults to just hours:minutes)
    const formattedCurrentTimestamp = format(new Date(currentTimestamp), timeFormat);

    // If we don't have necessary data, fallback to displaying just the current point
    if (!currentTimestamp || !data?.length) {
      return (
        <div>
          <span className="font-mono text-accent-9 text-xs px-4">{formattedCurrentTimestamp}</span>
        </div>
      );
    }

    // Find position in the data array
    const currentIndex = data.findIndex((item) => item?.originalTimestamp === currentTimestamp);

    // If this is the last item or not found, just show current timestamp
    if (currentIndex === -1 || currentIndex >= data.length - 1) {
      return (
        <div>
          <span className="font-mono text-accent-9 text-xs px-4">{formattedCurrentTimestamp}</span>
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
    const formattedNextTimestamp = format(new Date(nextPoint.originalTimestamp), timeFormat);

    // Return formatted interval
    return (
      <div>
        <span className="font-mono text-accent-9 text-xs px-4">
          {formattedCurrentTimestamp} - {formattedNextTimestamp}
        </span>
      </div>
    );
  };
}
