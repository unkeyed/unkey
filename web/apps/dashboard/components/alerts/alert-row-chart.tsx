"use client";

import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { InfoTooltip, Skeleton } from "@unkey/ui";
import { useMemo } from "react";
import { alertMetricLabel, formatAlertValue, seriesMetricForAlert } from "./format";
import type { AlertMetric } from "./types";

const MAX_BARS = 48;
const MAX_HEIGHT_PX = 28;

type ChartBar = {
  time: number;
  value: number;
  anomalous: boolean;
};

export function AlertRowChart({
  appId,
  environmentId,
  metric,
  windowStart,
  windowEnd,
}: {
  appId: string;
  environmentId: string;
  metric: AlertMetric;
  windowStart: number;
  windowEnd: number;
}) {
  const query = trpc.alerts.series.useQuery(
    {
      appId,
      environmentId,
      metric: seriesMetricForAlert(metric),
      startMs: Math.max(0, windowStart - 24 * 60 * 60 * 1000),
      endMs: windowEnd + 5 * 60 * 1000,
    },
    { staleTime: 60_000 },
  );
  const bars = useChartBars(query.data?.buckets, windowStart, windowEnd, metric);

  if (query.isLoading) {
    return <Skeleton className="h-7 w-[158px] rounded-md" />;
  }
  if (query.isError) {
    return <ChartMessage>Chart unavailable</ChartMessage>;
  }
  if (!bars.length) {
    return <ChartMessage>No telemetry</ChartMessage>;
  }

  const maxValue = Math.max(...bars.map((bar) => bar.value), 1);
  const isTrafficDrop = metric === "requests_drop";
  const extreme = isTrafficDrop
    ? Math.min(...bars.map((bar) => bar.value))
    : Math.max(...bars.map((bar) => bar.value));

  return (
    <InfoTooltip
      asChild
      delayDuration={300}
      position={{ side: "bottom" }}
      content={
        <div className="flex flex-col gap-1 px-3 py-2">
          <span className="text-xs font-medium text-gray-12">Past 24 hours</span>
          <span className="text-xs text-gray-9">
            {isTrafficDrop ? "Low" : "Peak"} {alertMetricLabel(metric).toLowerCase()}:{" "}
            {formatAlertValue(metric, extreme)}
          </span>
        </div>
      }
    >
      <div
        className="flex h-7 w-[158px] items-end gap-px overflow-hidden rounded-md border border-transparent bg-grayA-2 px-1"
        aria-label={`${alertMetricLabel(metric)} over the past 24 hours. The red bar marks the anomaly window.`}
      >
        {bars.map((bar) => (
          <span
            key={bar.time}
            aria-hidden="true"
            className={cn("flex-1 rounded-t-[1px]", bar.anomalous ? "bg-error-9" : "bg-grayA-6")}
            style={{
              height:
                bar.value === 0
                  ? 2
                  : Math.max(Math.round(Math.sqrt(bar.value / maxValue) * MAX_HEIGHT_PX), 3),
            }}
          />
        ))}
      </div>
    </InfoTooltip>
  );
}

function useChartBars(
  buckets: Array<{ time: number; value: number }> | undefined,
  windowStart: number | undefined,
  windowEnd: number | undefined,
  metric: AlertMetric,
): ChartBar[] {
  return useMemo(() => {
    if (!buckets?.length || windowStart === undefined || windowEnd === undefined) {
      return [];
    }
    const baselineAndAlert = buckets.filter((point) => point.time <= windowEnd);
    const groupSize = Math.max(Math.ceil(baselineAndAlert.length / MAX_BARS), 1);
    const bars: ChartBar[] = [];
    for (let index = 0; index < baselineAndAlert.length; index += groupSize) {
      const group = baselineAndAlert.slice(index, index + groupSize);
      const extreme = group.reduce((selected, point) => {
        if (metric === "requests_drop") {
          return point.value < selected.value ? point : selected;
        }
        return point.value > selected.value ? point : selected;
      });
      bars.push({
        time: extreme.time,
        value: extreme.value,
        anomalous: group.some((point) => point.time >= windowStart && point.time <= windowEnd),
      });
    }
    return bars;
  }, [buckets, metric, windowEnd, windowStart]);
}

function ChartMessage({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-7 w-[158px] items-center justify-center rounded-md bg-grayA-2 px-2 text-xs text-gray-9">
      {children}
    </div>
  );
}
