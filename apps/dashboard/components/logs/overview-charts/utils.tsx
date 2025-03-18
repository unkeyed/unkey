// components/logs/chart/utils/format-interval.ts

import { formatTimestampTooltip } from "../chart/utils/format-timestamp";
import type { TimeseriesData } from "./types";

/**
 * Creates a tooltip formatter that displays time intervals between data points
 *
 * @param data - The chart data array containing timestamp information
 * @returns A formatter function for use with chart tooltips
 */
export function createTimeIntervalFormatter(data?: TimeseriesData[]) {
  return (tooltipPayload: any[]) => {
    // Basic validation checks
    if (!tooltipPayload?.[0]?.payload) {
      return "";
    }

    const currentPayload = tooltipPayload[0].payload;
    const currentTimestamp = currentPayload.originalTimestamp;
    const currentDisplayX = currentPayload.displayX;

    // If we don't have necessary data, fallback to displaying just the current point
    if (!currentTimestamp || !currentDisplayX || !data?.length) {
      return (
        <div>
          <span className="font-mono text-accent-9 text-xs px-4">
            {currentDisplayX || formatTimestampTooltip(currentTimestamp)}
          </span>
        </div>
      );
    }

    // Find position in the data array
    const currentIndex = data.findIndex((item) => item?.originalTimestamp === currentTimestamp);

    // If this is the last item or not found, just show current timestamp
    if (currentIndex === -1 || currentIndex >= data.length - 1) {
      return (
        <div>
          <span className="font-mono text-accent-9 text-xs px-4">{currentDisplayX}</span>
        </div>
      );
    }

    // Get the next point in the sequence
    const nextPoint = data[currentIndex + 1];
    if (!nextPoint) {
      return (
        <div>
          <span className="font-mono text-accent-9 text-xs px-4">{currentDisplayX}</span>
        </div>
      );
    }

    // Format the next timestamp
    const nextDisplayX =
      nextPoint.displayX ||
      (nextPoint.originalTimestamp ? formatTimestampTooltip(nextPoint.originalTimestamp) : "");

    // Return formatted interval
    return (
      <div>
        <span className="font-mono text-accent-9 text-xs px-4">
          {currentDisplayX} - {nextDisplayX}
        </span>
      </div>
    );
  };
}
