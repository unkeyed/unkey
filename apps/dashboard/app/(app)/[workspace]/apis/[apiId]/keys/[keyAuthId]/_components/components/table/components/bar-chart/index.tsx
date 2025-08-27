import { cn } from "@/lib/utils";
import { useMemo } from "react";
import { UsageColumnSkeleton } from "../skeletons";
import { OutcomeExplainer } from "./components/outcome-explainer";
import { useFetchVerificationTimeseries } from "./use-fetch-timeseries";

type BarData = {
  id: string | number;
  topHeight: number;
  bottomHeight: number;
  totalHeight: number;
};

type VerificationBarChartProps = {
  keyAuthId: string;
  keyId: string;
  maxBars?: number;
  selected: boolean;
};

const MAX_HEIGHT_BUFFER_FACTOR = 1.3;
const MAX_BAR_HEIGHT = 28;

export const VerificationBarChart = ({
  keyAuthId,
  keyId,
  selected,
  maxBars = 30,
}: VerificationBarChartProps) => {
  const { timeseries, isLoading, isError } = useFetchVerificationTimeseries(keyAuthId, keyId);

  const isEmpty = useMemo(
    () => timeseries.reduce((acc, crr) => acc + crr.total, 0) === 0,
    [timeseries],
  );

  const bars = useMemo((): BarData[] => {
    if (isLoading || isError || timeseries.length === 0) {
      // Return empty data if loading, error, or no data
      return Array(maxBars).fill({
        id: 0,
        topHeight: 0,
        bottomHeight: 0,
        totalHeight: 0,
      });
    }
    // Get the most recent data points (or all if less than maxBars)
    const recentData = timeseries.slice(-maxBars);
    // Calculate the maximum total value to normalize heights
    const maxTotal =
      Math.max(...recentData.map((item) => item.total), 1) * MAX_HEIGHT_BUFFER_FACTOR;
    // Generate bars from the data
    return recentData.map((item, index): BarData => {
      // Scale to fit within max height of 28px
      const totalHeight = Math.min(
        Math.round((item.total / maxTotal) * MAX_BAR_HEIGHT),
        MAX_BAR_HEIGHT,
      );
      // Calculate heights proportionally
      const topHeight = item.error
        ? Math.max(Math.round((item.error / item.total) * totalHeight), 1)
        : 0;
      const bottomHeight = Math.max(totalHeight - topHeight, 0);
      return {
        id: index,
        totalHeight,
        topHeight,
        bottomHeight,
      };
    });
  }, [timeseries, isLoading, isError, maxBars]);

  // Pad with empty bars if we have fewer than maxBars data points
  const displayBars = useMemo((): BarData[] => {
    const result = [...bars];
    while (result.length < maxBars) {
      result.unshift({
        id: `empty-${result.length}`,
        topHeight: 0,
        bottomHeight: 0,
        totalHeight: 0,
      });
    }
    return result;
  }, [bars, maxBars]);

  // Loading state - animated pulse effect for bars with grid layout
  if (isLoading) {
    return <UsageColumnSkeleton />;
  }

  // Error state with grid layout
  if (isError) {
    return (
      <div
        className={cn(
          "grid items-end h-[28px] bg-grayA-2 dark:bg-grayA-2 w-[158px] border border-inside px-1 py-0 overflow-hidden rounded-t hover:rounded-md group-hover:rounded-md border-transparent hover:border-grayA-2 group-hover:border-grayA-2",
          selected ? "border-grayA-3 rounded-md" : "",
        )}
        style={{
          gridTemplateColumns: "1fr",
        }}
      >
        <div className="flex items-center justify-center w-full h-full text-xs text-error-9 px-2">
          Error loading data
        </div>
      </div>
    );
  }

  // Empty state with grid layout
  if (isEmpty) {
    return (
      <div
        className={cn(
          "grid items-end h-[28px] bg-grayA-2 dark:bg-grayA-2 w-[158px] border border-inside px-1 py-0 overflow-hidden rounded-t hover:rounded-md group-hover:rounded-md border-transparent hover:border-grayA-2 group-hover:border-grayA-2",
          selected ? "border-grayA-3 rounded-md" : "",
        )}
        style={{
          gridTemplateColumns: "1fr",
        }}
      >
        <div className="flex items-center justify-center w-full h-full text-xs text-grayA-9 px-2">
          No data available
        </div>
      </div>
    );
  }

  // Data display with grid layout
  return (
    <OutcomeExplainer timeseries={timeseries}>
      <div
        className={cn(
          "grid items-end h-[28px] bg-grayA-2 dark:bg-grayA-2 w-[158px] border border-inside px-1 py-0 overflow-hidden rounded-t hover:rounded-md group-hover:rounded-md border-transparent hover:border-grayA-2 group-hover:border-grayA-2",
          selected ? "border-grayA-3 rounded-md" : "",
        )}
        style={{
          gridTemplateColumns: `repeat(${maxBars}, 3px)`,
          gap: "2px",
        }}
      >
        {displayBars.map((bar) => (
          <div key={bar.id} className="flex flex-col">
            <div className="w-[3px] bg-error-9" style={{ height: `${bar.topHeight}px` }} />
            <div className="w-[3px] bg-grayA-5" style={{ height: `${bar.bottomHeight}px` }} />
          </div>
        ))}
      </div>
    </OutcomeExplainer>
  );
};
