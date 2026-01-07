import { VerificationLogsChart } from "@/components/logs/verification-chart";
import type { ChartLabels } from "@/components/logs/verification-chart";
import { useFilters } from "../../hooks/use-filters";
import { useFetchVerificationTimeseries } from "./bar-chart/hooks/use-fetch-timeseries";
import { createOutcomeChartConfig } from "./bar-chart/utils";

const KEY_CHART_LABELS: ChartLabels = {
  title: "KEY REQUESTS",
  primaryLabel: "VALID",
  primaryKey: "success",
  secondaryLabel: "INVALID",
  secondaryKey: "error",
};

const KEY_TOOLTIP_ITEMS = [{ label: "Invalid", dataKey: "error" }];

export const KeyDetailsLogsChart = ({
  keyspaceId,
  keyId,
  onMount,
}: {
  keyId: string;
  keyspaceId: string;
  onMount: (distanceToTop: number) => void;
}) => {
  return (
    <VerificationLogsChart
      dataHook={() => useFetchVerificationTimeseries(keyId, keyspaceId)}
      filtersHook={useFilters}
      chartConfig={createOutcomeChartConfig()}
      labels={KEY_CHART_LABELS}
      onMount={onMount}
      tooltipItems={KEY_TOOLTIP_ITEMS}
    />
  );
};