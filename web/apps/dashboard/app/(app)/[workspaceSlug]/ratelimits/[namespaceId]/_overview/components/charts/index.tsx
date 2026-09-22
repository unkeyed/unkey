import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { useFilters } from "../../hooks/use-filters";
import { useFetchRatelimitOverviewTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";

export const RatelimitOverviewLogsCharts = ({
  namespaceId,
}: {
  namespaceId: string;
}) => {
  const { filters, updateFilters } = useFilters();

  const { isError, isLoading, timeseries, tokensTimeseries, granularity } =
    useFetchRatelimitOverviewTimeseries(namespaceId);

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

  const sharedConfig = {
    success: {
      label: "Passed",
      color: "var(--color-gray-4)",
    },
    error: {
      label: "Blocked",
      color: "var(--color-orange-9)",
    },
  };

  return (
    <div className="flex w-full h-[320px] divide-x divide-gray-4">
      <div className="w-1/2">
        <OverviewBarChart
          data={timeseries}
          isLoading={isLoading}
          isError={isError}
          enableSelection
          onSelectionChange={handleSelectionChange}
          granularity={granularity}
          config={sharedConfig}
          labels={{
            title: "REQUESTS",
            primaryLabel: "PASSED",
            primaryKey: "success",
            secondaryLabel: "BLOCKED",
            secondaryKey: "error",
          }}
        />
      </div>
      <div className="w-1/2">
        <OverviewBarChart
          data={tokensTimeseries}
          isLoading={isLoading}
          isError={isError}
          enableSelection
          onSelectionChange={handleSelectionChange}
          granularity={granularity}
          config={sharedConfig}
          labels={{
            title: "TOKENS",
            primaryLabel: "PASSED",
            primaryKey: "success",
            secondaryLabel: "BLOCKED",
            secondaryKey: "error",
          }}
        />
      </div>
    </div>
  );
};
