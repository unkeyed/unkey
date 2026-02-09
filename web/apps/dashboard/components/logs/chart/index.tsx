// GenericTimeseriesChart.tsx
"use client";

import { ChartError, ChartLoading } from "@/components/logs/chart/chart-states";
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
import { Grid } from "@unkey/icons";
import { useEffect, useMemo, useRef, useState } from "react";
import { Bar, BarChart, ReferenceArea, YAxis } from "recharts";
import { parseTimestamp } from "../parse-timestamp";
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

type LogsTimeseriesBarChartProps = {
  data?: TimeseriesData[];
  config: ChartConfig;
  height?: number;
  onMount?: (distanceToTop: number) => void;
  onSelectionChange?: (selection: { start: number; end: number }) => void;
  isLoading?: boolean;
  isError?: boolean;
  enableSelection?: boolean;
  granularity?: TimeseriesGranularity;
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

  // Track if we're currently dragging for selection
  const isDragging = useRef(false);
  const dragStartData = useRef<{ index: number; timestamp: number } | null>(null);

  // Ref to track latest selection for closure-safe access in handleMouseUp
  const selectionRef = useRef(selection);
  useEffect(() => {
    selectionRef.current = selection;
  }, [selection]);

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

    const currentSelection = selectionRef.current;
    if (
      currentSelection.start !== undefined &&
      currentSelection.end !== undefined &&
      onSelectionChange
    ) {
      if (
        currentSelection.startTimestamp !== undefined &&
        currentSelection.endTimestamp !== undefined
      ) {
        const [start, end] = [currentSelection.startTimestamp, currentSelection.endTimestamp].sort(
          (a, b) => a - b,
        );
        onSelectionChange({ start, end });
      }
    }

    isDragging.current = false;
    dragStartData.current = null;
    setSelection({
      start: undefined,
      end: undefined,
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
    return <ChartError variant="compact" height={height} message="Could not retrieve logs" />;
  }

  if (isLoading) {
    return <ChartLoading variant="compact" height={height} dataPoints={300} />;
  }

  return (
    <div
      ref={chartRef}
      className="w-full relative"
      style={{ height: `${height}px` }}
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseLeave}
    >
      <div className="px-2 text-accent-11 font-mono absolute top-0 text-xxs w-full flex justify-between pointer-events-none z-10">
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
      <ChartContainer
        config={config}
        className="w-full aspect-auto border-b border-gray-4"
        style={{ height: `${height}px` }}
      >
        <BarChart
          data={data}
          margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
          barCategoryGap={0.5}
          // Disable recharts' built-in event handling by using empty handlers
          onMouseDown={() => {}}
          onMouseMove={() => {}}
          onMouseUp={() => {}}
          onMouseLeave={() => {}}
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
              x1={Math.min(selection.start, selection.end)}
              x2={Math.max(selection.start, selection.end)}
              fill="hsl(var(--chart-selection))"
              radius={[4, 4, 0, 0]}
            />
          )}
        </BarChart>
      </ChartContainer>
    </div>
  );
}
