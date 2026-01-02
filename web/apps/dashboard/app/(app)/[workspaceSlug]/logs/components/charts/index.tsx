"use client";
import { LogsTimeseriesBarChart } from "@/components/logs/chart";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { useFilters } from "../../hooks/use-filters";
import { useFetchTimeseries } from "./hooks/use-fetch-timeseries";

export function LogsChart({
  onMount,
}: {
  onMount: (distanceToTop: number) => void;
}) {
  const { filters, updateFilters } = useFilters();
  const { timeseries, isLoading, isError, granularity } = useFetchTimeseries();

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
    <LogsTimeseriesBarChart
      data={timeseries}
      config={{
        success: {
          label: "Success",
          subLabel: "2xx",
          color: "hsl(var(--accent-4))",
        },
        warning: {
          label: "Warning",
          subLabel: "4xx",
          color: "hsl(var(--warning-9))",
        },
        error: {
          label: "Error",
          subLabel: "5xx",
          color: "hsl(var(--error-9))",
        },
      }}
      onMount={onMount}
      onSelectionChange={handleSelectionChange}
      isLoading={isLoading}
      isError={isError}
      enableSelection={true}
    />
  );
}
