"use client";

import { type ChartConfig, ChartContainer } from "@/components/ui/chart";
import { Empty, Skeleton, Tabs, TabsList, TabsTrigger } from "@unkey/ui";
import { useMemo, useState } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ReferenceArea,
  ReferenceLine,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { alertMetricLabel, formatAlertAxisValue, formatAlertValue } from "../format";
import type { AlertMetric, AlertTimeseriesData } from "../types";

const chartConfig = {
  value: { label: "Observed", color: "hsl(var(--error-9))" },
} satisfies ChartConfig;

const threeHoursMs = 3 * 60 * 60 * 1000;

type ChartRange = "anomaly" | "baseline";

function isChartRange(value: string): value is ChartRange {
  return value === "anomaly" || value === "baseline";
}

export function AlertChart({
  metric,
  data,
  loading,
  error,
}: {
  metric: AlertMetric;
  data: AlertTimeseriesData | undefined;
  loading: boolean;
  error: boolean;
}) {
  const [range, setRange] = useState<ChartRange>("anomaly");
  const buckets = useMemo(() => {
    if (!data?.buckets.length || range === "baseline") {
      return data?.buckets ?? [];
    }
    const availableStart = data.buckets[0]?.time ?? data.windowStart;
    const availableEnd = data.buckets.at(-1)?.time ?? data.windowEnd;
    const rangeStart = Math.max(availableStart, data.windowStart - threeHoursMs);
    const rangeEnd = Math.min(availableEnd, data.windowEnd + threeHoursMs);
    return data.buckets.filter((bucket) => bucket.time >= rangeStart && bucket.time <= rangeEnd);
  }, [data, range]);

  if (loading) {
    return <Skeleton className="h-[360px] w-full rounded-lg" />;
  }
  if (error) {
    return (
      <Empty className="h-[360px] w-full">
        <Empty.Title>Chart unavailable</Empty.Title>
        <Empty.Description>We could not load telemetry for this alert.</Empty.Description>
      </Empty>
    );
  }
  if (!data?.buckets.length) {
    return (
      <Empty className="h-[360px] w-full">
        <Empty.Title>No telemetry</Empty.Title>
        <Empty.Description>No metric buckets remain for this alert's time range.</Empty.Description>
      </Empty>
    );
  }

  const thresholdBound = metric === "requests_drop" ? data.lowerBound : data.upperBound;

  return (
    <div className="overflow-hidden rounded-lg border border-grayA-4 bg-gray-1">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-grayA-4 px-5 py-4">
        <div>
          <h2 className="text-sm font-semibold text-gray-12">{alertMetricLabel(metric)}</h2>
          <p className="text-xs text-gray-9">Five-minute production buckets</p>
        </div>
        <Tabs
          value={range}
          onValueChange={(value) => {
            if (isChartRange(value)) {
              setRange(value);
            }
          }}
        >
          <TabsList aria-label="Chart range">
            <TabsTrigger value="anomaly">Around anomaly</TabsTrigger>
            <TabsTrigger value="baseline">Full baseline</TabsTrigger>
          </TabsList>
        </Tabs>
        <div className="flex flex-wrap items-center gap-4 text-xs text-gray-10">
          <ChartLegend color="bg-error-9" label="Observed" />
          <ChartLegend color="border-t-2 border-dashed border-gray-10" label="Baseline mean" />
          <ChartLegend color="bg-warningA-4" label="Threshold band" />
          <ChartLegend color="bg-errorA-3" label="Anomaly window" />
        </div>
      </div>
      <ChartContainer
        config={chartConfig}
        className="h-[360px] w-full aspect-auto px-3 pt-5 pb-2"
        aria-label={`${alertMetricLabel(metric)} anomaly timeseries`}
      >
        <LineChart
          data={buckets}
          accessibilityLayer
          margin={{ top: 8, right: 16, bottom: 8, left: 8 }}
        >
          <CartesianGrid
            vertical={false}
            stroke="hsl(var(--gray-6))"
            strokeDasharray="3 3"
            strokeOpacity={0.4}
          />
          <XAxis
            dataKey="time"
            type="number"
            scale="time"
            domain={["dataMin", "dataMax"]}
            tickLine={false}
            axisLine={false}
            minTickGap={48}
            tickFormatter={(value) => formatChartTime(Number(value))}
          />
          <YAxis
            width={72}
            tickLine={false}
            axisLine={false}
            tickFormatter={(value) => formatAlertAxisValue(metric, Number(value))}
          />
          <Tooltip
            cursor={{ stroke: "hsl(var(--gray-8))", strokeDasharray: "4 4" }}
            contentStyle={{
              border: "1px solid hsl(var(--gray-6))",
              borderRadius: 8,
              background: "hsl(var(--gray-2))",
              color: "hsl(var(--gray-12))",
              fontSize: 12,
            }}
            labelFormatter={(value) => formatChartTooltipTime(Number(value))}
            formatter={(value) => [formatAlertValue(metric, Number(value ?? 0)), "Observed"]}
          />
          <ReferenceArea
            y1={data.baselineMean}
            y2={thresholdBound}
            fill="hsl(var(--warning-9))"
            fillOpacity={0.12}
            strokeOpacity={0}
          />
          <ReferenceArea
            x1={data.windowStart}
            x2={data.windowEnd}
            fill="hsl(var(--error-9))"
            fillOpacity={0.2}
            stroke="hsl(var(--error-8))"
            strokeOpacity={0.8}
          />
          <ReferenceLine y={data.baselineMean} stroke="hsl(var(--gray-10))" strokeDasharray="5 5" />
          <Line
            type="monotone"
            dataKey="value"
            stroke="var(--color-value)"
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 4, fill: "hsl(var(--error-9))" }}
          />
        </LineChart>
      </ChartContainer>
    </div>
  );
}

function ChartLegend({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={`h-2 w-4 rounded-sm ${color}`} aria-hidden="true" />
      {label}
    </span>
  );
}

function formatChartTime(value: number): string {
  return new Intl.DateTimeFormat("en-US", { hour: "numeric", minute: "2-digit" }).format(value);
}

function formatChartTooltipTime(value: number): string {
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(value);
}
