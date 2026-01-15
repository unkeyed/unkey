import { createOutcomeChartConfig } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/[keyId]/components/charts/bar-chart/utils";
import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import {
  type TimeseriesGranularity,
  getTimeBufferForGranularity,
} from "@/lib/trpc/routers/utils/granularity";
import { useFilters } from "../../hooks/use-filters";
import { useFetchIdentityTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";

export const IdentityDetailsLogsChart = ({
  identityId,
  onMount,
}: {
  identityId: string;
  onMount: (distanceToTop: number) => void;
}) => {
  const { filters, updateFilters } = useFilters();

  const {
    timeseries: verificationTimeseries,
    isLoading: verificationIsLoading,
    isError: verificationIsError,
    granularity,
  } = useFetchIdentityTimeseries(identityId);

  // Convert schema granularity to TimeseriesGranularity format for chart
  const granularityMap: Record<string, TimeseriesGranularity> = {
    minute: "perMinute",
    fiveMinutes: "per5Minutes",
    fifteenMinutes: "per15Minutes",
    thirtyMinutes: "per30Minutes",
    hour: "perHour",
    twoHours: "per2Hours",
    fourHours: "per4Hours",
    sixHours: "per6Hours",
    twelveHours: "per12Hours",
    day: "perDay",
    threeDays: "per3Days",
    week: "perWeek",
    month: "perMonth",
    quarter: "perQuarter",
  };

  const mappedGranularity = granularity ? granularityMap[granularity] : undefined;

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
    if (start === end && mappedGranularity) {
      adjustedEnd = end + getTimeBufferForGranularity(mappedGranularity);
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
        granularity={mappedGranularity}
        labels={{
          title: "IDENTITY REQUESTS",
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
