import { OverviewBarChart } from "@/components/logs/overview-charts/overview-bar-chart";
import { getTimeBufferForGranularity } from "@/lib/trpc/routers/utils/granularity";
import { useEffect, useRef } from "react";
import { useFilters } from "../../hooks/use-filters";
import { useMetricType } from "../../hooks/use-metric-type";
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
  const { isCreditSpendMode } = useMetricType();
  const chartContainerRef = useRef<HTMLDivElement>(null);

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
      <div ref={chartContainerRef} className="w-full md:h-[320px]">
        <OverviewBarChart
          data={verificationTimeseries}
          isLoading={verificationIsLoading}
          isError={verificationIsError}
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
    );
  }

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
