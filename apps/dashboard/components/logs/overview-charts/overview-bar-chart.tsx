"use client";

import { calculateTimePoints } from "@/components/logs/chart/utils/calculate-timepoints";
import {
  formatTimestampLabel,
  formatTimestampTooltip,
} from "@/components/logs/chart/utils/format-timestamp";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { formatNumber } from "@/lib/fmt";
import { Grid } from "@unkey/icons";
import { useEffect, useRef, useState } from "react";
import { Bar, BarChart, CartesianGrid, ReferenceArea, ResponsiveContainer, YAxis } from "recharts";
import { OverviewChartError } from "./overview-bar-chart-error";
import { OverviewChartLoader } from "./overview-bar-chart-loader";
import type { Selection, TimeseriesData } from "./types";

type ChartTooltipItem = {
  label: string;
  dataKey: string;
};

type ChartLabels = {
  title: string;
  primaryLabel: string;
  primaryKey: string;
  secondaryLabel: string;
  secondaryKey: string;
};

type OverviewBarChartProps = {
  data?: TimeseriesData[];
  config: ChartConfig;
  onSelectionChange?: (selection: { start: number; end: number }) => void;
  isLoading?: boolean;
  isError?: boolean;
  enableSelection?: boolean;
  labels: ChartLabels;
  tooltipItems?: ChartTooltipItem[];
  onMount?: (distanceToTop: number) => void;
};

export function OverviewBarChart({
  data,
  config,
  onSelectionChange,
  isLoading,
  isError,
  enableSelection = false,
  labels,
  tooltipItems = [],
  onMount,
}: OverviewBarChartProps) {
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
        endTimestamp: timestamp,
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
    return <OverviewChartError labels={labels} />;
  }
  if (isLoading) {
    return <OverviewChartLoader labels={labels} />;
  }

  // Calculate totals based on the provided keys
  const totalCount = (data ?? []).reduce(
    (acc, crr) => acc + crr[labels.primaryKey] + crr[labels.secondaryKey],
    0,
  );
  const primaryCount = (data ?? []).reduce((acc, crr) => acc + crr[labels.primaryKey], 0);
  const secondaryCount = (data ?? []).reduce((acc, crr) => acc + crr[labels.secondaryKey], 0);

  return (
    <div className="flex flex-col h-full" ref={chartRef}>
      <div className="pl-5 pt-4 py-3 pr-10 w-full flex justify-between font-sans items-start gap-10 ">
        <div className="flex flex-col gap-1">
          <div className="text-accent-10 text-[11px] leading-4">{labels.title}</div>
          <div className="text-accent-12 text-[18px] font-semibold leading-7">
            {formatNumber(totalCount)}
          </div>
        </div>

        <div className="flex gap-10 items-center">
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-accent-8 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">{labels.primaryLabel}</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7">
              {formatNumber(primaryCount)}
            </div>
          </div>
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-orange-9 rounded h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">{labels.secondaryLabel}</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7">
              {formatNumber(secondaryCount)}
            </div>
          </div>
        </div>
      </div>
      <div className="flex-1 min-h-0">
        <ResponsiveContainer width="100%" height="100%">
          <ChartContainer config={config}>
            <BarChart
              data={data}
              margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
              onMouseDown={handleMouseDown}
              onMouseMove={handleMouseMove}
              onMouseUp={handleMouseUp}
              onMouseLeave={handleMouseUp}
            >
              <YAxis domain={["auto", (dataMax: number) => dataMax * 1]} hide />
              <CartesianGrid
                horizontal
                vertical={false}
                strokeDasharray="3 3"
                stroke="hsl(var(--gray-6))"
                strokeOpacity={0.3}
                strokeWidth={1}
              />
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
                        <div className="grid gap-1.5 pt-2 border-t border-gray-4 select-none">
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
                                  {payload[0]?.payload?.total}
                                </span>
                              </div>
                            </div>
                          </div>

                          {/* Dynamic tooltip items */}
                          {tooltipItems.map((item, index) => (
                            <div
                              key={`${item.label}-${index}`}
                              className="flex w-full [&>svg]:size-4 gap-4 px-4 items-center"
                            >
                              <Grid className="text-gray-6" />
                              <div className="flex gap-4 leading-none justify-between w-full py-1 items-center">
                                <div className="flex gap-4 items-center min-w-[80px]">
                                  <span className="capitalize text-accent-9 text-xs w-[2ch] inline-block">
                                    All
                                  </span>
                                  <span className="capitalize text-accent-12 text-xs">
                                    {item.label}
                                  </span>
                                </div>
                                <div className="ml-auto">
                                  <span className="font-mono tabular-nums text-accent-12">
                                    {payload[0]?.payload?.[item.dataKey]}
                                  </span>
                                </div>
                              </div>
                            </div>
                          ))}
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
                  x1={
                    selection.start && selection.end
                      ? Math.min(Number(selection.start), Number(selection.end))
                      : selection.start
                  }
                  x2={
                    selection.start && selection.end
                      ? Math.max(Number(selection.start), Number(selection.end))
                      : selection.end
                  }
                  fill="hsl(var(--chart-selection))"
                  radius={[4, 4, 0, 0]}
                />
              )}
            </BarChart>
          </ChartContainer>
        </ResponsiveContainer>
      </div>

      <div className="h-8 border-t border-b border-gray-4 px-1 py-2 text-accent-9 font-mono text-xxs w-full flex justify-between ">
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
    </div>
  );
}
