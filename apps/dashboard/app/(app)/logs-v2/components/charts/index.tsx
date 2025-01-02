"use client";

import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { addMinutes, format, subHours } from "date-fns";
import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";
import { generateMockLogsData } from "./util";
import { start } from "repl";
import { LogsTimeseriesDataPoint } from "@unkey/clickhouse/src/logs";

const chartConfig = {
  success: {
    label: "Success",
    color: "#0009321F",
  },
  warning: {
    label: "Warning",
    color: "#FFC53D",
  },
  error: {
    label: "Error",
    color: "#E54D2E",
  },
} satisfies ChartConfig;

const formatTimestampTooltip = (value: string | number) => {
  const date = new Date(value);
  const offset = new Date().getTimezoneOffset() * -1;
  const localDate = addMinutes(date, offset);
  return format(localDate, "dd MMM HH:mm:ss");
};

const formatTimestampLabel = (timestamp: string | number | Date) => {
  const date = new Date(timestamp);
  return format(date, "MMM dd, h:mma").toUpperCase();
};

type Timeseries = {
  x: number;
  displayX: string;
  originalTimestamp: string;
  success: number;
  error: number;
  warning: number;
  total: number;
};

const calculateTimePoints = (timeseries: Timeseries[]) => {
  const startTime = timeseries[0].x;
  const endTime = timeseries.at(-1)?.x;
  const timeRange = endTime ?? 0 - startTime;
  const timePoints = Array.from({ length: 5 }, (_, i) => {
    return new Date(startTime + (timeRange * i) / 4);
  });
  return timePoints;
};

const timeseries = generateMockLogsData(24, 10);

export function LogsChart() {
  return (
    <div className="w-full relative bg-white">
      <div className="px-2 text-[#00083045] font-mono absolute top-0 text-xxs w-full flex justify-between">
        {calculateTimePoints(timeseries).map((time, i) => (
          <div key={i}>{formatTimestampLabel(time)}</div>
        ))}
      </div>
      <ResponsiveContainer
        width="100%"
        height={45}
        className="border-b border-gray-4"
      >
        <ChartContainer config={chartConfig}>
          <BarChart
            data={timeseries}
            barGap={2}
            margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
          >
            <YAxis domain={[0, (dataMax: number) => dataMax * 1.5]} hide />
            <ChartTooltip
              cursor={false}
              content={
                <ChartTooltipContent
                  className="rounded-lg shadow-lg border border-border z-50"
                  labelFormatter={(_, payload) => {
                    const originalTimestamp =
                      payload[0]?.payload?.originalTimestamp;
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
            {["success", "warning", "error"].map((key) => (
              <Bar
                key={key}
                dataKey={key}
                stackId="a"
                fill={`var(--color-${key})`}
              />
            ))}
          </BarChart>
        </ChartContainer>
      </ResponsiveContainer>
    </div>
  );
}
