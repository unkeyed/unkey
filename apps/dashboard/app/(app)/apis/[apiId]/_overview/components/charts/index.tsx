import { OverviewAreaChart } from "@/components/logs/overview-charts/overview-area-chart";
import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { useFilters } from "../../hooks/use-filters";
import { useViewMode } from "../../hooks/use-view-mode";
import { useFetchVerificationTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";
import { createChartLabels, createOutcomeChartConfig } from "./bar-chart/utils";
import { ChartHeader } from "./chart-header";
import { useFetchActiveKeysTimeseries } from "./line-chart/hooks/use-fetch-timeseries";

export const KeysOverviewLogsCharts = ({
  apiId,
  onMount,
}: {
  apiId: string;
  onMount: (distanceToTop: number) => void;
}) => {
  const { filters, updateFilters } = useFilters();
  const { viewMode, setViewMode } = useViewMode();

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

  const chartLabels = createChartLabels(viewMode);
  const chartConfig = createOutcomeChartConfig(undefined, viewMode);

  const tooltipItems =
    viewMode === "credits"
      ? [{ label: "Credits Spent", dataKey: "spent_credits" }]
      : [{ label: "Invalid", dataKey: "error" }];

  return (
    <div className="flex flex-col md:flex-row w-full md:h-[320px]">
      <div className="w-full md:w-1/2 border-r border-gray-4 max-md:h-72 flex flex-col">
        <ChartHeader title={chartLabels.title} viewMode={viewMode} onViewModeChange={setViewMode} />
        <div className="flex-1">
          <OverviewBarChart
            data={verificationTimeseries}
            isLoading={verificationIsLoading}
            isError={verificationIsError}
            enableSelection
            onMount={onMount}
            onSelectionChange={handleSelectionChange}
            config={chartConfig}
            labels={chartLabels}
            tooltipItems={tooltipItems}
          />
        </div>
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
        />
      </div>
    </div>
  );
};
