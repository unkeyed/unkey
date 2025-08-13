import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { ChartHeader } from "../../../../../_overview/components/charts/chart-header";
import { useViewMode } from "../../../../../_overview/hooks/use-view-mode";
import { useFilters } from "../../hooks/use-filters";
import { useFetchVerificationTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";
import { createChartLabels, createOutcomeChartConfig } from "./bar-chart/utils";

export const KeyDetailsLogsChart = ({
  keyspaceId,
  keyId,
  onMount,
}: {
  keyId: string;
  keyspaceId: string;
  onMount: (distanceToTop: number) => void;
}) => {
  const { filters, updateFilters } = useFilters();
  const { viewMode, setViewMode } = useViewMode();

  const {
    timeseries: verificationTimeseries,
    isLoading: verificationIsLoading,
    isError: verificationIsError,
    granularity,
  } = useFetchVerificationTimeseries(keyId, keyspaceId);

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

  const chartLabels = createChartLabels(viewMode);
  const chartConfig = createOutcomeChartConfig(undefined, viewMode);

  const tooltipItems =
    viewMode === "credits"
      ? [{ label: "Credits Spent", dataKey: "spent_credits" }]
      : [{ label: "Invalid", dataKey: "error" }];

  return (
    <div className="w-full md:h-[320px] flex flex-col">
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
  );
};
