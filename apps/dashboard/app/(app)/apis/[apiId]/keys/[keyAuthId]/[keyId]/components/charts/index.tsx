import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { useFilters } from "../../hooks/use-filters";
import { useFetchVerificationTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";
import { createOutcomeChartConfig } from "./bar-chart/utils";

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
      (f) => !["startTime", "endTime", "since"].includes(f.field)
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
    <div className="w-full md:h-[320px]">
      <OverviewBarChart
        data={verificationTimeseries}
        isLoading={verificationIsLoading}
        isError={verificationIsError}
        enableSelection
        onMount={onMount}
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
    </div>
  );
};
