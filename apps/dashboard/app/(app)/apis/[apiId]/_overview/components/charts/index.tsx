import { useFilters } from "../../hooks/use-filters";
import { useFetchVerificationTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";
import { createOutcomeChartConfig } from "./bar-chart/utils";
import { useFetchActiveKeysTimeseries } from "./line-chart/hooks/use-fetch-timeseries";
import { LogsTimeseriesAreaChart } from "./line-chart";
import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";

export const KeysOverviewLogsCharts = ({ apiId }: { apiId: string }) => {
  const { filters, updateFilters } = useFilters();

  // Fetch verification data
  const {
    timeseries: verificationTimeseries,
    isLoading: verificationIsLoading,
    isError: verificationIsError,
  } = useFetchVerificationTimeseries(apiId);

  // Fetch active keys data
  const {
    timeseries: activeKeysTimeseries,
    isLoading: activeKeysIsLoading,
    isError: activeKeysIsError,
  } = useFetchActiveKeysTimeseries(apiId);

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
      <div className="w-1/2">
        <OverviewBarChart
          data={verificationTimeseries}
          isLoading={verificationIsLoading}
          isError={verificationIsError}
          enableSelection
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
      <div className="w-1/2 ">
        <LogsTimeseriesAreaChart
          data={activeKeysTimeseries}
          isLoading={activeKeysIsLoading}
          isError={activeKeysIsError}
          enableSelection
          onSelectionChange={handleSelectionChange}
          config={{
            keys: {
              label: "Active Keys",
            },
          }}
        />
      </div>
    </div>
  );
};
