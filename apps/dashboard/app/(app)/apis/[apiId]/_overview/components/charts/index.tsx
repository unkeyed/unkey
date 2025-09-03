import { OverviewAreaChart } from "@/components/logs/overview-charts/overview-area-chart";
import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { useFilters } from "../../hooks/use-filters";
import { useFetchVerificationTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";
import { createOutcomeChartConfig } from "./bar-chart/utils";
import { useFetchActiveKeysTimeseries } from "./line-chart/hooks/use-fetch-timeseries";

export const KeysOverviewLogsCharts = ({
  apiId,
  onMount,
}: {
  apiId: string;
  onMount: (distanceToTop: number) => void;
}) => {
  const { filters, updateFilters } = useFilters();

  const {
    timeseries: verificationTimeseries,
    isLoading: verificationIsLoading,
    isError: verificationIsError,
    granularity: verificationGranularity,
  } = useFetchVerificationTimeseries(apiId);

  const {
    timeseries: activeKeysTimeseries,
    isLoading: activeKeysIsLoading,
    isError: activeKeysIsError,
    granularity: activeKeysGranularity,
  } = useFetchActiveKeysTimeseries(apiId);

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
    if (start === end && verificationGranularity) {
      adjustedEnd = end + getTimeBufferForGranularity(verificationGranularity);
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

  const keysChartConfig = {
    keys: {
      label: "Active Keys",
      color: "hsl(var(--success-11))",
    },
  };

  const keysChartLabels = {
    title: "Keys Chart",
    rangeLabel: "ACTIVE KEYS",
    metrics: [
      {
        key: "keys",
        label: "AVG",
        color: "hsl(var(--success-11))",
      },
    ],
    showRightSide: false,
    reverse: true,
  };

  return (
    <div className="flex flex-col md:flex-row w-full md:h-[320px]">
      <div className="w-full md:w-1/2 border-r border-gray-4 max-md:h-72">
        <OverviewBarChart
          data={verificationTimeseries}
          isLoading={verificationIsLoading}
          isError={verificationIsError}
          enableSelection
          onMount={onMount}
          onSelectionChange={handleSelectionChange}
          config={createOutcomeChartConfig()}
          labels={{
            title: "REQUESTS",
            primaryLabel: "VALID",
            primaryKey: "success",
            secondaryLabel: "INVALID",
            secondaryKey: "error",
          }}
          tooltipItems={[{ label: "Invalid", dataKey: "error" }]}
          granularity={verificationGranularity}
        />
      </div>
      <div className="w-full md:w-1/2 max-md:h-72">
        <OverviewAreaChart
          data={activeKeysTimeseries}
          isLoading={activeKeysIsLoading}
          isError={activeKeysIsError}
          enableSelection
          onSelectionChange={handleSelectionChange}
          config={keysChartConfig}
          labels={keysChartLabels}
          granularity={activeKeysGranularity}
        />
      </div>
    </div>
  );
};
