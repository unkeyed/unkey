import { useFilters } from "../../hooks/use-filters";
import { LogsTimeseriesBarChart } from "./bar-chart";
import { useFetchVerificationTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";
import { createOutcomeChartConfig } from "./bar-chart/utils";

export const KeysOverviewLogsCharts = ({
  keyspaceId,
}: {
  keyspaceId: string;
}) => {
  const { filters, updateFilters } = useFilters();
  const { timeseries, isLoading, isError } =
    useFetchVerificationTimeseries(keyspaceId);

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
      <div className="w-full">
        <LogsTimeseriesBarChart
          data={timeseries}
          isLoading={isLoading}
          isError={isError}
          enableSelection
          onSelectionChange={handleSelectionChange}
          config={createOutcomeChartConfig()}
        />
      </div>
    </div>
  );
};
