"use client";

import { calculateTimePoints } from "@/components/logs/chart/utils/calculate-timepoints";
import { formatTimestampLabel } from "@/components/logs/chart/utils/format-timestamp";
import { formatTooltipInterval } from "@/components/logs/utils";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { formatNumber } from "@/lib/fmt";
import type { TimeseriesGranularity } from "@/lib/trpc/routers/utils/granularity";
import { cn } from "@/lib/utils";
import { useMemo, useRef, useState } from "react";
import { Area, AreaChart, CartesianGrid, ReferenceArea, YAxis } from "recharts";
import { parseTimestamp } from "../parse-timestamp";

import { ChartError, ChartLoading } from "@/components/logs/chart/chart-states";
import type { Selection, TimeseriesData } from "./types";

export type ChartMetric = {
  key: string;
  label: string;
  color: string;
  formatter?: (value: number) => string;
};

export type TimeseriesChartLabels = {
  title: string;
  rangeLabel: string;
  metrics: ChartMetric[];
  showRightSide?: boolean;
  reverse?: boolean;
};

export type Granularity = TimeseriesGranularity | undefined;

export interface TimeseriesAreaChartProps {
  data?: TimeseriesData[];
  config: ChartConfig;
  onSelectionChange?: (selection: { start: number; end: number }) => void;
  isLoading?: boolean;
  isError?: boolean;
  enableSelection?: boolean;
  labels: TimeseriesChartLabels;
  granularity?: Granularity;
}

export const OverviewAreaChart = ({
  data = [],
  config,
  onSelectionChange,
  isLoading,
  isError,
  enableSelection = false,
  labels,
  granularity,
}: TimeseriesAreaChartProps) => {
  const chartRef = useRef<HTMLDivElement>(null);
  const [selection, setSelection] = useState<Selection>({ start: "", end: "" });

  // Track if we're currently dragging for selection
  const isDragging = useRef(false);
  const dragStartData = useRef<{ index: number; timestamp: number } | null>(null);

  // Precompute timestamp-to-index map for O(1) lookups during hover/tooltip
  const timestampToIndexMap = useMemo(() => {
    const map = new Map<number, number>();
    data.forEach((item, index) => {
      if (item?.originalTimestamp) {
        const normalizedTimestamp = parseTimestamp(item.originalTimestamp);
        if (Number.isFinite(normalizedTimestamp)) {
          map.set(normalizedTimestamp, index);
        }
      }
    });
    return map;
  }, [data]);

  const labelsWithDefaults = {
    ...labels,
    showRightSide: labels.showRightSide !== undefined ? labels.showRightSide : true,
    reverse: labels.reverse !== undefined ? labels.reverse : false,
  };

  // Helper to find data point from x coordinate
  const getDataPointFromEvent = (
    event: React.MouseEvent | MouseEvent,
    container: HTMLDivElement,
  ): { index: number; timestamp: number } | null => {
    if (!data || data.length === 0) {
      return null;
    }

    const rect = container.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const width = rect.width;
    const relativeX = x / width;
    const index = Math.floor(relativeX * data.length);

    if (index >= 0 && index < data.length) {
      const timestamp = data[index]?.originalTimestamp;
      if (timestamp !== undefined) {
        return { index, timestamp };
      }
    }
    return null;
  };

  // Handle mouse down on container
  const handleMouseDown = (e: React.MouseEvent) => {
    if (!enableSelection || !chartRef.current) {
      return;
    }

    const point = getDataPointFromEvent(e, chartRef.current);
    if (!point) {
      return;
    }

    isDragging.current = true;
    dragStartData.current = point;

    setSelection({
      start: point.index,
      end: point.index,
      startTimestamp: point.timestamp,
      endTimestamp: point.timestamp,
    });
  };

  // Handle mouse move on container
  const handleMouseMove = (e: React.MouseEvent) => {
    if (!enableSelection || !isDragging.current || !dragStartData.current || !chartRef.current) {
      return;
    }

    const point = getDataPointFromEvent(e, chartRef.current);
    if (!point) {
      return;
    }

    setSelection((prev) => ({
      ...prev,
      end: point.index,
      endTimestamp: point.timestamp,
    }));
  };

  // Handle mouse up to finalize selection
  const handleMouseUp = () => {
    if (!enableSelection) {
      return;
    }

    if (selection.start !== undefined && selection.end !== undefined && onSelectionChange) {
      if (selection.startTimestamp !== undefined && selection.endTimestamp !== undefined) {
        const [start, end] = [selection.startTimestamp, selection.endTimestamp].sort(
          (a, b) => a - b,
        );
        onSelectionChange({ start, end });
      }
    }

    isDragging.current = false;
    dragStartData.current = null;
    setSelection({
      start: "",
      end: "",
      startTimestamp: undefined,
      endTimestamp: undefined,
    });
  };

  // Handle mouse leave to cancel selection
  const handleMouseLeave = () => {
    if (isDragging.current) {
      handleMouseUp();
    }
  };

  if (isError) {
    return <ChartError variant="full" labels={labelsWithDefaults} />;
  }
  if (isLoading) {
    return <ChartLoading variant="full" labels={labelsWithDefaults} />;
  }

  // Calculate metrics
  const ranges: Record<string, { min: number; max: number; avg: number }> = {};

  labelsWithDefaults.metrics.forEach((metric) => {
    const values = data.map((d) => d[metric.key] as number);
    const min = data.length > 0 ? Math.min(...values) : 0;
    const max = data.length > 0 ? Math.max(...values) : 0;
    const avg = data.length > 0 ? values.reduce((sum, val) => sum + val, 0) / data.length : 0;

    ranges[metric.key] = { min, max, avg };
  });

  // Get primary metric for range display
  const primaryMetric = labelsWithDefaults.metrics[0];

  return (
    <div
      className="flex flex-col h-full"
      ref={chartRef}
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseLeave}
    >
      <div
        className={cn(
          "pl-5 pt-4 py-3 pr-10 w-full flex justify-between font-sans items-start gap-10",
          labelsWithDefaults.reverse && "flex-row-reverse",
        )}
      >
        <div className="flex flex-col gap-1 max-md:w-full">
          <div className="flex items-center gap-2">
            {labelsWithDefaults.reverse &&
              labelsWithDefaults.metrics.map((metric) => (
                <div
                  key={metric.key}
                  className="rounded h-[10px] w-1"
                  style={{ backgroundColor: metric.color }}
                />
              ))}
            <div className="text-accent-10 text-[11px] leading-4">
              {labelsWithDefaults.rangeLabel}
            </div>
          </div>
          <div className="text-accent-12 text-[18px] font-semibold leading-7">
            {primaryMetric.formatter
              ? `${primaryMetric.formatter(
                  ranges[primaryMetric.key].min,
                )} - ${primaryMetric.formatter(ranges[primaryMetric.key].max)}`
              : `${formatNumber(
                  ranges[primaryMetric.key].min,
                )} - ${formatNumber(ranges[primaryMetric.key].max)}`}
          </div>
        </div>

        {labelsWithDefaults.showRightSide && (
          <div className="flex gap-10 items-center">
            {labelsWithDefaults.metrics.map((metric) => (
              <div key={metric.key} className="flex flex-col gap-1">
                <div className="flex gap-2 items-center">
                  <div className="rounded h-[10px] w-1" style={{ backgroundColor: metric.color }} />
                  <div className="text-accent-10 text-[11px] leading-4">{metric.label}</div>
                </div>
                <div className="text-accent-12 text-[18px] font-semibold leading-7">
                  {metric.formatter
                    ? metric.formatter(ranges[metric.key].avg)
                    : formatNumber(ranges[metric.key].avg)}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="flex-1 min-h-0">
        <ChartContainer config={config} className="w-full h-full aspect-auto">
          <AreaChart
            data={data}
            margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
            // Disable recharts' built-in event handling
            onMouseDown={() => {}}
            onMouseMove={() => {}}
            onMouseUp={() => {}}
            onMouseLeave={() => {}}
          >
            <defs>
              {labelsWithDefaults.metrics.map((metric) => (
                <linearGradient
                  key={`${metric.key}Gradient`}
                  id={`${metric.key}Gradient`}
                  x1="0"
                  y1="0"
                  x2="0"
                  y2="1"
                >
                  <stop offset="5%" stopColor={metric.color} stopOpacity={0.2} />
                  <stop offset="95%" stopColor={metric.color} stopOpacity={0} />
                </linearGradient>
              ))}
            </defs>

            <YAxis domain={["auto", (dataMax: number) => dataMax * 1.1]} hide />
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
                stroke: "hsl(var(--accent-3))",
                strokeWidth: 1,
                strokeDasharray: "5 5",
                strokeOpacity: 0.7,
              }}
              content={({ active, payload, label }) => {
                if (!active || !payload?.length) {
                  return null;
                }
                return (
                  <ChartTooltipContent
                    payload={payload}
                    label={label}
                    active={active}
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

            {labelsWithDefaults.metrics.map((metric) => (
              <Area
                key={metric.key}
                type="monotone"
                dataKey={metric.key}
                stroke={metric.color}
                fill={`url(#${metric.key}Gradient)`}
                fillOpacity={1}
                strokeWidth={2}
              />
            ))}

            {enableSelection && selection.start && selection.end && (
              <ReferenceArea
                x1={Math.min(Number(selection.start), Number(selection.end))}
                x2={Math.max(Number(selection.start), Number(selection.end))}
                fill="hsl(var(--chart-selection))"
                fillOpacity={0.3}
              />
            )}
          </AreaChart>
        </ChartContainer>
      </div>

      <div className="h-max border-t border-b border-gray-4 px-1 py-2 text-accent-9 font-mono text-xxs w-full flex justify-between ">
        {data.length > 0
          ? (() => {
              const lastItem = data.at(-1);
              return calculateTimePoints(
                data[0]?.originalTimestamp ? parseTimestamp(data[0].originalTimestamp) : Date.now(),
                lastItem?.originalTimestamp
                  ? parseTimestamp(lastItem.originalTimestamp)
                  : Date.now(),
              ).map((time, i) => (
                // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
                <div key={i} className="z-10 text-center">
                  {formatTimestampLabel(time)}
                </div>
              ));
            })()
          : null}
      </div>
    </div>
  );
};
