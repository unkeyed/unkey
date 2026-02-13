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
import { Grid } from "@unkey/icons";
import { useEffect, useMemo, useRef, useState } from "react";
import { Bar, BarChart, CartesianGrid, ReferenceArea, YAxis } from "recharts";
import { parseTimestamp } from "../parse-timestamp";

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
  granularity?: TimeseriesGranularity;
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
  granularity,
}: OverviewBarChartProps) {
  const chartRef = useRef<HTMLDivElement>(null);
  const [selection, setSelection] = useState<Selection>({ start: "", end: "" });

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
    return <OverviewChartError labels={labels} />;
  }
  if (isLoading) {
    return <OverviewChartLoader labels={labels} />;
  }

  // Calculate totals based on the provided keys
  const totalCount = (data ?? []).reduce(
    (acc, crr) => acc + (crr[labels.primaryKey] as number) + (crr[labels.secondaryKey] as number),
    0,
  );
  const primaryCount = (data ?? []).reduce(
    (acc, crr) => acc + (crr[labels.primaryKey] as number),
    0,
  );
  const secondaryCount = (data ?? []).reduce(
    (acc, crr) => acc + (crr[labels.secondaryKey] as number),
    0,
  );

  return (
    <div
      className="flex flex-col h-full"
      ref={chartRef}
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseLeave}
    >
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
              <div className="bg-accent-8 rounded-sm h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">{labels.primaryLabel}</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7">
              {formatNumber(primaryCount)}
            </div>
          </div>
          <div className="flex flex-col gap-1">
            <div className="flex gap-2 items-center">
              <div className="bg-orange-9 rounded-sm h-[10px] w-1" />
              <div className="text-accent-10 text-[11px] leading-4">{labels.secondaryLabel}</div>
            </div>
            <div className="text-accent-12 text-[18px] font-semibold leading-7">
              {formatNumber(secondaryCount)}
            </div>
          </div>
        </div>
      </div>
      <div className="flex-1 min-h-0">
        <ChartContainer config={config} className="w-full h-full aspect-auto">
          <BarChart
            data={data}
            margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
            barCategoryGap={0.5}
            // Disable recharts' built-in event handling
            onMouseDown={() => {}}
            onMouseMove={() => {}}
            onMouseUp={() => {}}
            onMouseLeave={() => {}}
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
                                {formatNumber(payload[0]?.payload?.total)}
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
                                  {formatNumber(payload[0]?.payload?.[item.dataKey] ?? 0)}
                                </span>
                              </div>
                            </div>
                          </div>
                        ))}
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
            {enableSelection && selection.start && selection.end && (
              <ReferenceArea
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
      </div>

      <div className="h-max border-t border-b border-gray-4 px-1 py-2 text-accent-9 font-mono text-xxs w-full flex justify-between ">
        {data
          ? calculateTimePoints(
              data[0]?.originalTimestamp ?? Date.now(),
              data.at(-1)?.originalTimestamp ?? Date.now(),
            ).map((time, i) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: <explanation>
              <div key={i} className="z-10 text-center">
                {formatTimestampLabel(time)}
              </div>
            ))
          : null}
      </div>
    </div>
  );
}
