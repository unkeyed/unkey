// GenericTimeseriesChart.tsx
"use client";

import { formatTimestampLabel } from "@/components/logs/chart/utils/format-timestamp";
import { formatTooltipInterval } from "@/components/logs/utils";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { formatNumber } from "@/lib/fmt";
import type { CompoundTimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { Grid } from "@unkey/icons";
import { useEffect, useMemo, useRef, useState } from "react";
import { Bar, BarChart, ReferenceArea, ResponsiveContainer, YAxis } from "recharts";
import { parseTimestamp } from "../parse-timestamp";
import { LogsChartError } from "./components/logs-chart-error";
import { LogsChartLoading } from "./components/logs-chart-loading";
import { calculateTimePoints } from "./utils/calculate-timepoints";

type Selection = {
  start: number | undefined;
  end: number | undefined;
  startTimestamp?: number;
  endTimestamp?: number;
};

type TimeseriesData = {
  originalTimestamp: number;
  total: number;
  [key: string]: unknown;
};

export type ChartMouseEvent = {
  activeLabel?: string | number;
  activePayload?: ReadonlyArray<{
    payload: TimeseriesData;
  }>;
};

type LogsTimeseriesBarChartProps = {
  data?: TimeseriesData[];
  config: ChartConfig;
  height?: number;
  onMount?: (distanceToTop: number) => void;
  onSelectionChange?: (selection: { start: number; end: number }) => void;
  isLoading?: boolean;
  isError?: boolean;
  enableSelection?: boolean;
  granularity?: CompoundTimeseriesGranularity;
};

export function LogsTimeseriesBarChart({
  data,
  config,
  height = 50,
  onMount,
  onSelectionChange,
  isLoading,
  isError,
  enableSelection = false,
  granularity,
}: LogsTimeseriesBarChartProps) {
  const chartRef = useRef<HTMLDivElement>(null);
  const [selection, setSelection] = useState<Selection>({
    start: undefined,
    end: undefined,
  });

  // Precompute timestamp-to-index mapping for O(1) lookup
  const timestampToIndexMap = useMemo(() => {
    const map = new Map<number, number>();
    if (data?.length) {
      data.forEach((item, index) => {
        if (item?.originalTimestamp) {
          const normalizedTimestamp = parseTimestamp(item.originalTimestamp);
          if (Number.isFinite(normalizedTimestamp)) {
            map.set(normalizedTimestamp, index);
          }
        }
      });
    }
    return map;
  }, [data]);
  // biome-ignore lint/correctness/useExhaustiveDependencies: We need this to re-trigger distanceToTop calculation
  useEffect(() => {
    if (onMount) {
      const distanceToTop = chartRef.current?.getBoundingClientRect().top ?? 0;
      onMount(distanceToTop);
    }
  }, [onMount, isLoading, isError]);

  const handleMouseDown = (e: ChartMouseEvent) => {
    if (!enableSelection || e.activeLabel === undefined) {
      return;
    }

    const timestamp = e.activePayload?.[0]?.payload?.originalTimestamp;
    const numericLabel = Number(e.activeLabel);

    if (!Number.isFinite(numericLabel) || !timestamp) {
      return;
    }

    setSelection({
      start: numericLabel,
      end: numericLabel,
      startTimestamp: timestamp,
      endTimestamp: timestamp,
    });
  };

  const handleMouseMove = (e: ChartMouseEvent) => {
    if (!enableSelection || e.activeLabel === undefined) {
      return;
    }
    if (selection.start !== undefined) {
      const timestamp = e.activePayload?.[0]?.payload?.originalTimestamp;
      const numericLabel = Number(e.activeLabel);

      if (!Number.isFinite(numericLabel) || !timestamp) {
        return;
      }

      setSelection((prev) => ({
        ...prev,
        end: numericLabel,
        endTimestamp: timestamp,
      }));
    }
  };

  const handleMouseUp = () => {
    if (!enableSelection) {
      return;
    }
    if (selection.start !== undefined && selection.end !== undefined && onSelectionChange) {
      if (selection.startTimestamp === undefined || selection.endTimestamp === undefined) {
        return;
      }

      const [start, end] = [selection.startTimestamp, selection.endTimestamp].sort((a, b) => a - b);
      onSelectionChange({ start, end });
    }
    setSelection({
      start: undefined,
      end: undefined,
      startTimestamp: undefined,
      endTimestamp: undefined,
    });
  };

  if (isError) {
    return <LogsChartError />;
  }

  if (isLoading) {
    return <LogsChartLoading />;
  }

  return (
    <div className="w-full relative" ref={chartRef}>
      <div className="px-2 text-accent-11 font-mono absolute top-0 text-xxs w-full flex justify-between pointer-events-none">
        {data
          ? calculateTimePoints(
              data[0]?.originalTimestamp ?? Date.now(),
              data.at(-1)?.originalTimestamp ?? Date.now(),
            ).map((time) => (
              <div key={time.getTime()} className="z-10">
                {formatTimestampLabel(time)}
              </div>
            ))
          : null}
      </div>
      <ResponsiveContainer width="100%" height={height} className="border-b border-gray-4">
        <ChartContainer config={config}>
          <BarChart
            data={data}
            margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
            barCategoryGap={0.5}
            onMouseDown={handleMouseDown}
            onMouseMove={handleMouseMove}
            onMouseUp={handleMouseUp}
            onMouseLeave={handleMouseUp}
          >
            <YAxis domain={[0, (dataMax: number) => dataMax * 1.5]} hide />
            <ChartTooltip
              position={{ y: 50 }}
              isAnimationActive
              wrapperStyle={{ zIndex: 1000 }}
              cursor={{
                fill: "hsl(var(--accent-3))",
                strokeWidth: 1,
                strokeDasharray: "5 5",
                strokeOpacity: 0.7,
              }}
              content={({ active, payload, label }) => {
                if (!active || !payload?.length || payload?.[0]?.payload.total === 0) {
                  return null;
                }
                console.log(payload);
                return (
                  <ChartTooltipContent
                    payload={payload}
                    label={label}
                    active={active}
                    bottomExplainer={
                      <div className="grid gap-1.5 pt-2 border-t border-gray-4">
                        <div className="flex w-full [&>svg]:size-4 gap-4 px-4 items-center">
                          <Grid className="text-gray-6" />
                          <div className="flex gap-4 leading-none justify-between w-full py-1 items-center">
                            <div className="flex gap-4 items-center min-w-[80px]">
                              <span className="capitalize text-accent-9 text-xs w-[2ch] inline-block">
                                All
                              </span>
                              <span className="capitalize text-accent-12 text-xs">Total</span>
                            </div>
                            <div className="ml-auto">
                              <span className="font-mono tabular-nums text-accent-12">
                                {formatNumber(payload[0]?.payload?.total)}
                              </span>
                            </div>
                          </div>
                        </div>
                      </div>
                    }
                    className="rounded-lg shadow-lg border border-gray-4"
                    labelFormatter={(_, tooltipPayload) => {
                      const payloadTimestamp = tooltipPayload?.[0]?.payload?.originalTimestamp;
                      return formatTooltipInterval(
                        payloadTimestamp,
                        data || [],
                        granularity,
                        timestampToIndexMap,
                      );
                    }}
                  />
                );
              }}
            />
            {Object.keys(config).map((key) => (
              <Bar key={key} dataKey={key} stackId="a" fill={config[key].color} />
            ))}
            {enableSelection && selection.start !== undefined && selection.end !== undefined && (
              <ReferenceArea
                isAnimationActive
                x1={Math.min(selection.start, selection.end)}
                x2={Math.max(selection.start, selection.end)}
                fill="hsl(var(--chart-selection))"
                radius={[4, 4, 0, 0]}
              />
            )}
          </BarChart>
        </ChartContainer>
      </ResponsiveContainer>
    </div>
  );
}
