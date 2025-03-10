import { useFilters } from "../../hooks/use-filters";
import { LogsTimeseriesBarChart } from "./bar-chart";
import { useFetchVerificationTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";
import { createOutcomeChartConfig } from "./bar-chart/utils";
import { LogsTimeseriesAreaChart } from "./line-chart";
import { useFetchActiveKeysTimeseries } from "./line-chart/hooks/use-fetch-timeseries";

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
      (f) => !["startTime", "endTime", "since"].includes(f.field),
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
        <LogsTimeseriesBarChart
          data={verificationTimeseries}
          isLoading={verificationIsLoading}
          isError={verificationIsError}
          enableSelection
          onSelectionChange={handleSelectionChange}
          config={createOutcomeChartConfig()}
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
