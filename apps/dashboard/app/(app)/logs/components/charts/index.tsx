"use client";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import type { LogsTimeseriesDataPoint } from "@unkey/clickhouse/src/logs";
import { addMinutes, format } from "date-fns";
import { useEffect, useState } from "react";
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, XAxis, YAxis } from "recharts";
import { useFetchTimeseries } from "./hooks";

const chartConfig = {
  success: {
    label: "Success",
    color: "hsl(var(--chart-3))",
  },
  warning: {
    label: "Warning",
    color: "hsl(var(--chart-4))",
  },
  error: {
    label: "Error",
    color: "hsl(var(--chart-1))",
  },
} satisfies ChartConfig;

const formatTimestampTooltip = (value: string | number) => {
  const date = new Date(value);
  const offset = new Date().getTimezoneOffset() * -1;
  const localDate = addMinutes(date, offset);
  return format(localDate, "dd MMM HH:mm:ss");
};

const calculateTickInterval = (dataLength: number, containerWidth: number) => {
  const pixelsPerTick = 80; // Adjust this value to control density
  const suggestedInterval = Math.ceil((dataLength * pixelsPerTick) / containerWidth);

  const intervals = [1, 2, 5, 10, 15, 30, 60];
  return intervals.find((i) => i >= suggestedInterval) || intervals[intervals.length - 1];
};

export function LogsChart({
  initialTimeseries,
}: {
  initialTimeseries: LogsTimeseriesDataPoint[];
}) {
  const { timeseries } = useFetchTimeseries(initialTimeseries);
  const [tickInterval, setTickInterval] = useState(5);
  const [containerWidth, setContainerWidth] = useState(0);

  useEffect(() => {
    if (containerWidth > 0 && timeseries.length > 0) {
      const newInterval = calculateTickInterval(timeseries.length, containerWidth);
      setTickInterval(newInterval);
    }
  }, [timeseries.length, containerWidth]);

  const maxValue = Math.max(
    ...timeseries.map((item) => (item.success || 0) + (item.warning || 0) + (item.error || 0)),
  );
  const yAxisMax = Math.ceil(maxValue * 1.1);

  return (
    <div className="w-full h-[120px] p-1 rounded-lg bg-card">
      <ResponsiveContainer
        width="100%"
        height="60px"
        onResize={(width) => setContainerWidth(width)}
      >
        <ChartContainer config={chartConfig}>
          <BarChart data={timeseries} barCategoryGap={4} maxBarSize={20}>
            <XAxis
              dataKey="displayX"
              tickLine={false}
              axisLine={false}
              className="font-mono"
              minTickGap={30}
              tick={{
                fontSize: 11,
                fill: "hsl(var(--muted-foreground))",
                dy: 10,
              }}
              interval={tickInterval - 1}
            />
            <CartesianGrid
              strokeDasharray="3 3"
              stroke="hsl(var(--border))"
              vertical={false}
              strokeOpacity={0.4}
            />
            <ChartTooltip
              cursor={false}
              content={
                <ChartTooltipContent
                  className="rounded-lg shadow-lg border border-border"
                  labelFormatter={(_, payload) => {
                    const originalTimestamp = payload[0]?.payload?.originalTimestamp;
                    return originalTimestamp ? (
                      <span className="font-mono text-muted-foreground text-xs">
                        {formatTimestampTooltip(originalTimestamp)}
                      </span>
                    ) : (
                      ""
                    );
                  }}
                />
              }
            />
            <YAxis domain={[0, yAxisMax]} hide />
            {["success", "warning", "error"].map((key, index) => (
              <Bar
                key={key}
                dataKey={key}
                stackId="a"
                fill={`var(--color-${key})`}
                radius={[
                  index === 0 ? 2 : 0,
                  index === 2 ? 2 : 0,
                  index === 2 ? 2 : 0,
                  index === 0 ? 2 : 0,
                ]}
              />
            ))}
          </BarChart>
        </ChartContainer>
      </ResponsiveContainer>
    </div>
  );
}
