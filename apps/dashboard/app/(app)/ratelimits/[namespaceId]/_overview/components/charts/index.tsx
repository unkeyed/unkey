import { convertDateToLocal } from "@/components/logs/chart/utils/convert-date-to-local";
import { generateLatencyData } from "../../dev-utils";
import { useFilters } from "../../hooks/use-filters";
import { LogsTimeseriesBarChart } from "./bar-chart";
import { useFetchRatelimitOverviewTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";
import { LogsTimeseriesAreaChart } from "./line-chart";

export const RatelimitOverviewLogsCharts = ({
  namespaceId,
}: {
  namespaceId: string;
}) => {
  const { filters, updateFilters } = useFilters();
  const { isError, isLoading, timeseries } = useFetchRatelimitOverviewTimeseries(namespaceId);

  // Generate latency data with matching timestamps
  const latencyData = generateLatencyData(timeseries?.length ?? 0, {
    baseAvgLatency: 150,
    baseP99Latency: 300,
    variability: 0.3,
    trendFactor: 0.4,
  });

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
          data={latencyData}
          isLoading={isLoading}
          isError={isError}
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
