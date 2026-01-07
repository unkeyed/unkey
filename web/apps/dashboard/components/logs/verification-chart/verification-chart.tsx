"use client";

import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import type { TimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import type { ChartConfig } from "@/components/ui/chart";

export type TimeseriesData = {
  displayX: string;
  originalTimestamp: number;
  valid: number;
  total: number;
  success: number;
  error: number;
  [key: string]: string | number;
};

export type Filter = {
  field: string;
  value: string | number;
  id: string;
  operator: string;
};

export type ChartLabels = {
  title: string;
  primaryLabel: string;
  primaryKey: string;
  secondaryLabel: string;
  secondaryKey: string;
};

export type ChartTooltipItem = {
  label: string;
  dataKey: string;
};

export interface VerificationLogsChartProps {
  // Data fetching (injected)
  dataHook: () => {
    timeseries: TimeseriesData[];
    isLoading: boolean;
    isError: boolean;
    granularity?: TimeseriesGranularity;
  };
  // Filter management (injected)
  filtersHook: () => {
    filters: Filter[];
    updateFilters: (filters: Filter[]) => void;
  };
  // Chart configuration (injected)
  chartConfig: ChartConfig;
  labels: ChartLabels;
  onMount?: (distanceToTop: number) => void;
  tooltipItems?: ChartTooltipItem[];
}

const DEFAULT_TOOLTIP_ITEMS: ChartTooltipItem[] = [
  { label: "Invalid", dataKey: "error" }
];

export function VerificationLogsChart({
  dataHook,
  filtersHook,
  chartConfig,
  labels,
  onMount,
  tooltipItems = DEFAULT_TOOLTIP_ITEMS,
}: VerificationLogsChartProps) {
  const { filters, updateFilters } = filtersHook();
  
  const {
    timeseries,
    isLoading,
    isError,
    granularity,
  } = dataHook();

  const handleSelectionChange = ({
    start,
    end,
  }: {
    start: number;
    end: number;
  }) => {
    const activeFilters = filters.filter(
      (f) => !["startTime", "endTime", "since"].includes(f.field),
    );

    let adjustedEnd = end;
    if (start === end && granularity) {
      adjustedEnd = end + getTimeBufferForGranularity(granularity);
    }
    
    updateFilters([
      ...activeFilters,
      {
        field: "startTime",
        value: start,
        id: crypto.randomUUID(),
        operator: "is",
      },
      {
        field: "endTime",
        value: adjustedEnd,
        id: crypto.randomUUID(),
        operator: "is",
      },
    ]);
  };

  return (
    <div className="w-full md:h-[320px]">
      <OverviewBarChart
        data={timeseries}
        isLoading={isLoading}
        isError={isError}
        enableSelection
        onMount={onMount}
        onSelectionChange={handleSelectionChange}
        config={chartConfig}
        granularity={granularity}
        labels={labels}
        tooltipItems={tooltipItems}
      />
    </div>
  );
}