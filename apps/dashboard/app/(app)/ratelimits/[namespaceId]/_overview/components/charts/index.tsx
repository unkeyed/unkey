import { convertDateToLocal } from "@/components/logs/chart/utils/convert-date-to-local";
import { useFilters } from "../../hooks/use-filters";
import { LogsTimeseriesBarChart } from "./bar-chart";
import { useFetchRatelimitOverviewTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";
import { LogsTimeseriesAreaChart } from "./line-chart";
import { useFetchRatelimitOverviewLatencyTimeseries } from "./line-chart/hooks/use-fetch-timeseries";

export const RatelimitOverviewLogsCharts = ({
  namespaceId,
}: {
  namespaceId: string;
}) => {
  const { filters, updateFilters } = useFilters();
  const { isError, isLoading, timeseries } =
    useFetchRatelimitOverviewTimeseries(namespaceId);
  const { latencyIsError, latencyIsLoading, latencyTimeseries } =
    useFetchRatelimitOverviewLatencyTimeseries(namespaceId);

  const handleSelectionChange = ({
    start,
    end,
  }: {
    start: number;
    end: number;
  }) => {
    const activeFilters = filters.filter(
      (f) => !["startTime", "endTime", "since"].includes(f.field)
    );

    updateFilters([
      ...activeFilters,
      {
        field: "startTime",
        value: convertDateToLocal(start),
        id: crypto.randomUUID(),
        operator: "is",
      },
      {
        field: "endTime",
        value: convertDateToLocal(end),
        id: crypto.randomUUID(),
        operator: "is",
      },
    ]);
  };
  console.log({ latencyTimeseries });

  return (
    <div className="flex w-full h-[320px]">
      <div className="w-1/2 border-r border-gray-4">
        <LogsTimeseriesBarChart
          data={timeseries}
          isLoading={isLoading}
          isError={isError}
          enableSelection
          onSelectionChange={handleSelectionChange}
          config={{
            success: {
              label: "Passed",
              color: "hsl(var(--accent-4))",
            },
            error: {
              label: "Blocked",
              color: "hsl(var(--orange-9))",
            },
          }}
        />
      </div>
      <div className="w-1/2">
        <LogsTimeseriesAreaChart
          data={latencyTimeseries}
          isLoading={latencyIsLoading}
          isError={latencyIsError}
          config={{
            avgLatency: {
              color: "hsl(var(--accent-8))",
              label: "Average Latency",
            },
            p99Latency: {
              color: "hsl(var(--warning-11))",
              label: "P99 Latency",
            },
          }}
        />
      </div>
    </div>
  );
};
