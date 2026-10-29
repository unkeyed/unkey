import { createOutcomeChartConfig } from "@/components/charts/chart-colors";
import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import { OverviewChartFrame } from "@/components/logs/overview-charts/overview-chart-frame";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { useFilters } from "../../hooks/use-filters";
import { useFetchVerificationTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";

export const KeyDetailsLogsChart = ({
  keyspaceId,
  keyId,
}: {
  keyId: string;
  keyspaceId: string;
}) => {
  const { filters, updateFilters } = useFilters();

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

  return (
    <OverviewChartFrame>
      <OverviewBarChart
        data={verificationTimeseries}
        isLoading={verificationIsLoading}
        isError={verificationIsError}
        enableSelection
        onSelectionChange={handleSelectionChange}
        config={createOutcomeChartConfig()}
        granularity={granularity}
        labels={{
          title: "REQUESTS",
          primaryLabel: "VALID",
          primaryKey: "success",
          secondaryLabel: "INVALID",
          secondaryKey: "error",
        }}
        tooltipItems={[{ label: "Invalid", dataKey: "error" }]}
      />
    </OverviewChartFrame>
  );
};
