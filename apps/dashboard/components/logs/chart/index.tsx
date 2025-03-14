// GenericTimeseriesChart.tsx
"use client";

import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { formatNumber } from "@/lib/fmt";
import { Grid } from "@unkey/icons";
import { useEffect, useRef, useState } from "react";
import { Bar, BarChart, ReferenceArea, ResponsiveContainer, YAxis } from "recharts";
import { LogsChartError } from "./components/logs-chart-error";
import { LogsChartLoading } from "./components/logs-chart-loading";
import { calculateTimePoints } from "./utils/calculate-timepoints";
import { formatTimestampLabel, formatTimestampTooltip } from "./utils/format-timestamp";

type Selection = {
  start: string | number;
  end: string | number;
  startTimestamp?: number;
  endTimestamp?: number;
};

type TimeseriesData = {
  originalTimestamp: number;
  total: number;
  [key: string]: any;
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
}: LogsTimeseriesBarChartProps) {
  const chartRef = useRef<HTMLDivElement>(null);
  const [selection, setSelection] = useState<Selection>({ start: "", end: "" });

  // biome-ignore lint/correctness/useExhaustiveDependencies: We need this to re-trigger distanceToTop calculation
  useEffect(() => {
    if (onMount) {
      const distanceToTop = chartRef.current?.getBoundingClientRect().top ?? 0;
      onMount(distanceToTop);
    }
  }, [onMount, isLoading, isError]);

  const handleMouseDown = (e: any) => {
    if (!enableSelection) {
      return;
    }
    const timestamp = e.activePayload?.[0]?.payload?.originalTimestamp;
    setSelection({
      start: e.activeLabel,
      end: e.activeLabel,
      startTimestamp: timestamp,
      endTimestamp: timestamp,
    });
  };

  const handleMouseMove = (e: any) => {
    if (!enableSelection) {
      return;
    }
    if (selection.start) {
      const timestamp = e.activePayload?.[0]?.payload?.originalTimestamp;
      setSelection((prev) => ({
        ...prev,
        end: e.activeLabel,
        startTimestamp: timestamp,
      }));
    }
  };

  const handleMouseUp = () => {
    if (!enableSelection) {
      return;
    }
    if (selection.start && selection.end && onSelectionChange) {
      if (!selection.startTimestamp || !selection.endTimestamp) {
        return;
      }

      const [start, end] = [selection.startTimestamp, selection.endTimestamp].sort((a, b) => a - b);
      onSelectionChange({ start, end });
    }
    setSelection({
      start: "",
      end: "",
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
            ).map((time, i) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
              <div key={i} className="z-10">
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
                      const originalTimestamp = tooltipPayload[0]?.payload?.originalTimestamp;
                      return originalTimestamp ? (
                        <div>
                          <span className="font-mono text-accent-9 text-xs px-4">
                            {formatTimestampTooltip(originalTimestamp)}
                          </span>
                        </div>
                      ) : (
                        ""
                      );
                    }}
                  />
                );
              }}
            />
            {Object.keys(config).map((key) => (
              <Bar key={key} dataKey={key} stackId="a" fill={config[key].color} />
            ))}
            {enableSelection && selection.start && selection.end && (
              <ReferenceArea
                isAnimationActive
                x1={Math.min(Number(selection.start), Number(selection.end))}
                x2={Math.max(Number(selection.start), Number(selection.end))}
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
