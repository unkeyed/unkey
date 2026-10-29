"use client";
import { LogsTimeseriesBarChart } from "@/components/logs/chart";
import { useFilters } from "../../hooks/use-filters";
import { useFetchTimeseries } from "./hooks/use-fetch-timeseries";

export function LogsChart() {
  const { filters, updateFilters } = useFilters();
  const { timeseries, isLoading, isError, bucketMs, granularity } = useFetchTimeseries();

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

    // end is the start of the last selected bar, and endTime is inclusive, so
    // the window ends on the last millisecond of that bar
    const adjustedEnd = bucketMs ? end + bucketMs - 1 : end;

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
          color: "var(--color-gray-4)",
        },
        clientError: {
          label: "Warning",
          subLabel: "4xx",
          color: "var(--color-warning-9)",
        },
        serverError: {
          label: "Error",
          subLabel: "5xx",
          color: "var(--color-error-9)",
        },
      }}
      onSelectionChange={handleSelectionChange}
      isLoading={isLoading}
      isError={isError}
      enableSelection={true}
      granularity={granularity}
    />
  );
}
