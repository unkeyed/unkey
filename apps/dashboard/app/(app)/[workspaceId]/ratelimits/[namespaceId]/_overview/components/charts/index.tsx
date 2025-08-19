import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { useFilters } from "../../hooks/use-filters";
import { useFetchRatelimitOverviewTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";

// const latencyFormatter = new Intl.NumberFormat("en-US", {
//   maximumFractionDigits: 2,
//   minimumFractionDigits: 2,
// });

export const RatelimitOverviewLogsCharts = ({
  namespaceId,
}: {
  namespaceId: string;
}) => {
  const { filters, updateFilters } = useFilters();

  const { isError, isLoading, timeseries, granularity } =
    useFetchRatelimitOverviewTimeseries(namespaceId);

  // const {
  //   isError: latencyIsError,
  //   isLoading: latencyIsLoading,
  //   timeseries: latencyTimeseries,
  // } = useFetchRatelimitOverviewLatencyTimeseries(namespaceId);

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

  // // Format the latency values with 'ms' suffix
  // const formatLatency = (value: number) =>
  //   `${latencyFormatter.format(value)}ms`;
  //
  // // Define the latency chart config
  // const latencyChartConfig = {
  //   avgLatency: {
  //     label: "Average Latency",
  //     color: "hsl(var(--accent-11))",
  //   },
  //   p99Latency: {
  //     label: "P99 Latency",
  //     color: "hsl(var(--warning-11))",
  //   },
  // };
  //
  // // Define the latency chart labels
  // const latencyChartLabels = {
  //   title: "Latency Chart",
  //   rangeLabel: "DURATION",
  //   metrics: [
  //     {
  //       key: "avgLatency",
  //       label: "AVG",
  //       color: "hsl(var(--accent-11))",
  //       formatter: formatLatency,
  //     },
  //     {
  //       key: "p99Latency",
  //       label: "P99",
  //       color: "hsl(var(--warning-11))",
  //       formatter: formatLatency,
  //     },
  //   ],
  // };
  //
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
      {/*   <OverviewAreaChart */}
      {/*     data={latencyTimeseries} */}
      {/*     isLoading={latencyIsLoading} */}
      {/*     isError={latencyIsError} */}
      {/*     enableSelection */}
      {/*     onSelectionChange={handleSelectionChange} */}
      {/*     config={latencyChartConfig} */}
      {/*     labels={latencyChartLabels} */}
      {/*   /> */}
      {/* </div> */}
    </div>
  );
};
