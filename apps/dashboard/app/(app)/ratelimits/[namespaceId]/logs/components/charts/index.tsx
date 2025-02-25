"use client";
import { LogsTimeseriesBarChart } from "@/components/logs/chart";
import { useRatelimitLogsContext } from "../../context/logs";
import { useFilters } from "../../hooks/use-filters";
import { useFetchRatelimitTimeseries } from "./hooks/use-fetch-timeseries";

export function RatelimitLogsChart({
  onMount,
}: {
  onMount: (distanceToTop: number) => void;
}) {
  const { namespaceId } = useRatelimitLogsContext();
  const { filters, updateFilters } = useFilters();
  const { timeseries, isLoading, isError } = useFetchRatelimitTimeseries(namespaceId);

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
        value: end,
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
          label: "Passed",
          subLabel: "Passed",
          color: "hsl(var(--accent-4))",
        },
        error: {
          label: "Blocked",
          subLabel: "Blocked",
          color: "hsl(var(--warning-9))",
        },
      }}
      onMount={onMount}
      onSelectionChange={handleSelectionChange}
      isLoading={isLoading}
      isError={isError}
      enableSelection
    />
  );
}
