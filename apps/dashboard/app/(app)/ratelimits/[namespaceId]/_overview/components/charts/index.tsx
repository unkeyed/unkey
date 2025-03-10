import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import { useFilters } from "../../hooks/use-filters";
import { useFetchRatelimitOverviewTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";

export const RatelimitOverviewLogsCharts = ({
  namespaceId,
}: {
  namespaceId: string;
}) => {
  const { filters, updateFilters } = useFilters();
  const { isError, isLoading, timeseries } =
    useFetchRatelimitOverviewTimeseries(namespaceId);
  // const { latencyIsError, latencyIsLoading, latencyTimeseries } =
  //   useFetchRatelimitOverviewLatencyTimeseries(namespaceId);

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
    <div className="flex w-full h-[320px]">
      <div className="w-full">
        <OverviewBarChart
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
          labels={{
            title: "REQUESTS",
            primaryLabel: "PASSED",
            primaryKey: "success",
            secondaryLabel: "BLOCKED",
            secondaryKey: "error",
          }}
        />
      </div>
      {/* <div className="w-1/2"> */}
      {/*   <LogsTimeseriesAreaChart */}
      {/*     data={latencyTimeseries} */}
      {/*     isLoading={latencyIsLoading} */}
      {/*     isError={latencyIsError} */}
      {/*     enableSelection */}
      {/*     onSelectionChange={handleSelectionChange} */}
      {/*     config={{ */}
      {/*       avgLatency: { */}
      {/*         label: "Average Latency", */}
      {/*       }, */}
      {/*       p99Latency: { */}
      {/*         label: "P99 Latency", */}
      {/*       }, */}
      {/*     }} */}
      {/*   /> */}
      {/* </div> */}
    </div>
  );
};
