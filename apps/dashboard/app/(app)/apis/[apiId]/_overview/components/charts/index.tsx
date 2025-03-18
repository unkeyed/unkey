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
  } = useFetchVerificationTimeseries(apiId);

  const {
    timeseries: activeKeysTimeseries,
    isLoading: activeKeysIsLoading,
    isError: activeKeysIsError,
    granularity,
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
    <div className="flex w-full h-[320px]">
      <div className="w-1/2 border-r border-gray-4">
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
        />
      </div>
      <div className="w-1/2">
        <OverviewAreaChart
          data={activeKeysTimeseries}
          isLoading={activeKeysIsLoading}
          isError={activeKeysIsError}
          enableSelection
          onSelectionChange={handleSelectionChange}
          config={keysChartConfig}
          labels={keysChartLabels}
        />
      </div>
    </div>
  );
};
