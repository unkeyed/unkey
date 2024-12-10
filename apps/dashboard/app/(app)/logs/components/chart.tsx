"use client";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import type { LogsTimeseriesDataPoint } from "@unkey/clickhouse/src/logs";
import { addMinutes, format } from "date-fns";
import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";

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

const formatTimestamp = (value: string | number) => {
  const date = new Date(value);
  const offset = new Date().getTimezoneOffset() * -1;
  const localDate = addMinutes(date, offset);
  return format(localDate, "dd MMM HH:mm:ss");
};

export function LogsChart({
  initialTimeseries,
}: {
  initialTimeseries: LogsTimeseriesDataPoint[];
}) {
  const transformedData = initialTimeseries.map((data) => ({
    displayX: formatTimestamp(data.x),
    originalTimestamp: data.x,
    ...data.y,
  }));

  return (
    <ChartContainer config={chartConfig} className="h-[125px] w-full">
      <BarChart accessibilityLayer data={transformedData}>
        <XAxis dataKey="displayX" tickLine={false} axisLine={false} dx={-40} />
        <CartesianGrid
          strokeDasharray="2"
          stroke={"hsl(var(--cartesian-grid-stroke))"}
          vertical={false}
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              className="w-[200px]"
              nameKey="views"
              labelFormatter={(_, payload) => {
                const originalTimestamp = payload[0]?.payload?.originalTimestamp;
                return originalTimestamp ? formatTimestamp(originalTimestamp) : "";
              }}
            />
          }
        />
        <Bar dataKey="success" stackId="a" fill="var(--color-success)" radius={3} maxBarSize={20} />
        <Bar dataKey="warning" stackId="a" fill="var(--color-warning)" radius={3} maxBarSize={20} />
        <Bar dataKey="error" stackId="a" fill="var(--color-error)" radius={3} maxBarSize={20} />
      </BarChart>
    </ChartContainer>
  );
}
