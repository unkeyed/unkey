"use client";

import { calculateTimePoints } from "@/components/logs/chart/utils/calculate-timepoints";
import { formatTimestampLabel } from "@/components/logs/chart/utils/format-timestamp";
import { formatTooltipTimestamp } from "@/components/logs/utils";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { formatNumber } from "@/lib/fmt";
import {
  type CompoundTimeseriesGranularity,
  getTimeBufferForGranularity,
} from "@/lib/trpc/routers/utils/granularity";
import { cn } from "@/lib/utils";
import { useState } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ReferenceArea,
  ResponsiveContainer,
  YAxis,
} from "recharts";

import { OverviewAreaChartError } from "./overview-area-chart-error";
import { OverviewAreaChartLoader } from "./overview-area-chart-loader";
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

export type Granularity =
  | "perMinute"
  | "per5Minutes"
  | "per15Minutes"
  | "per30Minutes"
  | "perHour"
  | "per2Hours"
  | "per4Hours"
  | "per6Hours"
  | "per12Hours"
  | "perDay"
  | "per3Days"
  | "perWeek"
  | "perMonth"
  | "perYear"
  | undefined;

// Helper function to safely convert local Granularity to CompoundTimeseriesGranularity
const getGranularityBuffer = (granularity: Granularity): number => {
  if (!granularity || granularity === "perYear") {
    return 60000; // 1 minute fallback
  }
  return getTimeBufferForGranularity(granularity as CompoundTimeseriesGranularity);
};

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
  const [selection, setSelection] = useState<Selection>({ start: "", end: "" });

  const labelsWithDefaults = {
    ...labels,
    showRightSide: labels.showRightSide !== undefined ? labels.showRightSide : true,
    reverse: labels.reverse !== undefined ? labels.reverse : false,
  };

  // biome-ignore lint/suspicious/noExplicitAny: <explanation>
  const handleMouseDown = (e: any) => {
    if (!enableSelection) {
      return;
    }
    const timestamp = e?.activePayload?.[0]?.payload?.originalTimestamp;
    setSelection({
      start: e.activeLabel,
      end: e.activeLabel,
      startTimestamp: timestamp,
      endTimestamp: timestamp,
    });
  };

  // biome-ignore lint/suspicious/noExplicitAny: <explanation>
  const handleMouseMove = (e: any) => {
    if (!enableSelection || !selection.start) {
      return;
    }
    const timestamp = e?.activePayload?.[0]?.payload?.originalTimestamp;
    setSelection((prev) => ({
      ...prev,
      end: e.activeLabel,
      endTimestamp: timestamp,
    }));
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
    return <OverviewAreaChartError labels={labelsWithDefaults} />;
  }
  if (isLoading) {
    return <OverviewAreaChartLoader labels={labelsWithDefaults} />;
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
    <div className="flex flex-col h-full">
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
        <ResponsiveContainer width="100%" height="100%">
          <ChartContainer config={config}>
            <AreaChart
              data={data}
              margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
              onMouseDown={handleMouseDown}
              onMouseMove={handleMouseMove}
              onMouseUp={handleMouseUp}
              onMouseLeave={handleMouseUp}
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
                        if (!tooltipPayload?.[0]?.payload?.originalTimestamp) {
                          return "";
                        }

                        const currentPayload = tooltipPayload[0].payload;
                        const currentTimestamp = currentPayload.originalTimestamp;

                        // Handle missing data
                        if (!data?.length) {
                          return (
                            <div className="px-4">
                              <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
                                {formatTooltipTimestamp(currentTimestamp, granularity, data)}
                              </span>
                            </div>
                          );
                        }

                        // Handle single timestamp case
                        if (data.length === 1) {
                          return (
                            <div className="px-4">
                              <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
                                {formatTooltipTimestamp(currentTimestamp, granularity, data)}
                              </span>
                            </div>
                          );
                        }

                        // Normalize timestamps to numeric for robust comparison
                        const currentTimestampNumeric =
                          typeof currentTimestamp === "number"
                            ? currentTimestamp
                            : +new Date(currentTimestamp);

                        // Find position in the data array using numeric comparison
                        const currentIndex = data.findIndex((item) => {
                          if (!item?.originalTimestamp) {
                            return false;
                          }
                          const itemTimestamp =
                            typeof item.originalTimestamp === "number"
                              ? item.originalTimestamp
                              : +new Date(item.originalTimestamp);
                          return itemTimestamp === currentTimestampNumeric;
                        });

                        // If not found, fallback to single timestamp display
                        if (currentIndex === -1) {
                          return (
                            <div className="px-4">
                              <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
                                {formatTooltipTimestamp(currentTimestampNumeric, granularity, data)}
                              </span>
                            </div>
                          );
                        }

                        // Compute interval end timestamp
                        let intervalEndTimestamp: number;

                        // If this is the last item, compute interval end using granularity
                        if (currentIndex >= data.length - 1) {
                          const inferredGranularityMs = granularity
                            ? getGranularityBuffer(granularity)
                            : data.length > 1
                              ? Math.abs(
                                  (typeof data[1].originalTimestamp === "number"
                                    ? data[1].originalTimestamp
                                    : +new Date(data[1].originalTimestamp)) -
                                    (typeof data[0].originalTimestamp === "number"
                                      ? data[0].originalTimestamp
                                      : +new Date(data[0].originalTimestamp)),
                                )
                              : 60000; // 1 minute fallback
                          intervalEndTimestamp = currentTimestampNumeric + inferredGranularityMs;
                        } else {
                          // Use next data point's timestamp
                          const nextPoint = data[currentIndex + 1];
                          if (!nextPoint?.originalTimestamp) {
                            // Fallback to single timestamp if next point is invalid
                            return (
                              <div className="px-4">
                                <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
                                  {formatTooltipTimestamp(
                                    currentTimestampNumeric,
                                    granularity,
                                    data,
                                  )}
                                </span>
                              </div>
                            );
                          }
                          intervalEndTimestamp =
                            typeof nextPoint.originalTimestamp === "number"
                              ? nextPoint.originalTimestamp
                              : +new Date(nextPoint.originalTimestamp);
                        }

                        // Format both timestamps using normalized numeric values
                        const formattedCurrentTimestamp = formatTooltipTimestamp(
                          currentTimestampNumeric,
                          granularity,
                          data,
                        );
                        const formattedNextTimestamp = formatTooltipTimestamp(
                          intervalEndTimestamp,
                          granularity,
                          data,
                        );

                        // Get timezone abbreviation from the actual point date for correct DST handling
                        const pointDate = new Date(currentTimestampNumeric);
                        const timezone =
                          new Intl.DateTimeFormat("en-US", {
                            timeZoneName: "short",
                          })
                            .formatToParts(pointDate)
                            .find((part) => part.type === "timeZoneName")?.value || "";

                        // Return formatted interval with timezone info
                        return (
                          <div className="px-4">
                            <span className="font-mono text-accent-9 text-xs whitespace-nowrap">
                              {formattedCurrentTimestamp} - {formattedNextTimestamp} ({timezone})
                            </span>
                          </div>
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
        </ResponsiveContainer>
      </div>

      <div className="h-max border-t border-b border-gray-4 px-1 py-2 text-accent-9 font-mono text-xxs w-full flex justify-between ">
        {data.length > 0
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
};
