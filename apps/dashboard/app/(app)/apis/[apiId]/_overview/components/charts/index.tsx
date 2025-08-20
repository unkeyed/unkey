import { OverviewAreaChart } from "@/components/logs/overview-charts/overview-area-chart";
import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { useEffect, useRef } from "react";
import { useFilters } from "../../hooks/use-filters";
import { useMetricType } from "../../hooks/use-metric-type";
import { useFetchVerificationTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";
import { createOutcomeChartConfig } from "./bar-chart/utils";
import { useFetchCreditSpendTimeseries } from "./credit-spend-chart/hooks/use-fetch-timeseries";
import { useFetchActiveKeysTimeseries } from "./line-chart/hooks/use-fetch-timeseries";

export const KeysOverviewLogsCharts = ({
  apiId,
  onMount,
}: {
  apiId: string;
  onMount: (distanceToTop: number) => void;
}) => {
  const { filters, updateFilters } = useFilters();
  const { isCreditSpendMode } = useMetricType();
  const chartContainerRef = useRef<HTMLDivElement>(null);

  const {
    timeseries: verificationTimeseries,
    isLoading: verificationIsLoading,
    isError: verificationIsError,
  } = useFetchVerificationTimeseries(apiId);

  const {
    timeseries: creditSpendTimeseries,
    isLoading: creditSpendIsLoading,
    isError: creditSpendIsError,
  } = useFetchCreditSpendTimeseries(apiId);

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

  const creditSpendChartConfig = {
    spent_credits: {
      label: "Credits Spent",
      color: "hsl(var(--success-9))",
    },
  };

  // Call onMount for credit spend mode
  useEffect(() => {
    if (isCreditSpendMode && chartContainerRef.current) {
      const rect = chartContainerRef.current.getBoundingClientRect();
      onMount(rect.top + window.scrollY);
    }
  }, [isCreditSpendMode, onMount]);

  if (isCreditSpendMode) {
    return (
      <div
        ref={chartContainerRef}
        className="flex flex-col md:flex-row w-full md:h-[320px]"
      >
        <div className="w-full md:w-1/2 border-r border-gray-4 max-md:h-72">
          <OverviewBarChart
            data={creditSpendTimeseries}
            isLoading={creditSpendIsLoading}
            isError={creditSpendIsError}
            enableSelection
            onMount={onMount}
            onSelectionChange={handleSelectionChange}
            config={creditSpendChartConfig}
            labels={{
              title: "CREDITS SPENT",
              primaryLabel: "CREDITS SPENT",
              primaryKey: "spent_credits",
              secondaryLabel: "",
              secondaryKey: "",
            }}
            showLabels={false}
            tooltipPrefix={null}
            hideTooltipTotal={true}
          />
        </div>
        {/* Only show active keys chart if it has data in credit spend mode */}
        {activeKeysTimeseries.length > 0 && (
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
        )}
      </div>
    );
  }

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
        />
      </div>
    </div>
  );
};
