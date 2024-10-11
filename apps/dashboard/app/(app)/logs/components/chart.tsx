"use client";

import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { format } from "date-fns";
import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";

export type Log = {
  request_id: string;
  time: number;
  workspace_id: string;
  host: string;
  method: string;
  path: string;
  request_headers: string[];
  request_body: string;
  response_status: number;
  response_headers: string[];
  response_body: string;
  error: string;
  service_latency: number;
};

const chartConfig = {
  success: {
    label: "Success",
    color: "hsl(var(--chart-1))",
  },
  warning: {
    label: "Warning",
    color: "hsl(var(--chart-2))",
  },
  error: {
    label: "Error",
    color: "hsl(var(--chart-3))",
  },
} satisfies ChartConfig;

function aggregateData(data: Log[]) {
  const aggregatedData: {
    date: string;
    success: number;
    warning: number;
    error: number;
  }[] = [];
  const intervalMs = 60 * 1000 * 10;

  if (data.length === 0) {
    return aggregatedData;
  }

  const startOfDay = new Date(data[0].time).setHours(0, 0, 0, 0);
  const endOfDay = startOfDay + 24 * 60 * 60 * 1000;

  for (
    let timestamp = startOfDay;
    timestamp < endOfDay;
    timestamp += intervalMs
  ) {
    const filteredLogs = data.filter(
      (d) => d.time >= timestamp && d.time < timestamp + intervalMs
    );

    const success = filteredLogs.filter(
      (log) => log.response_status >= 200 && log.response_status < 300
    ).length;
    const warning = filteredLogs.filter(
      (log) => log.response_status >= 400 && log.response_status < 500
    ).length;
    const error = filteredLogs.filter(
      (log) => log.response_status >= 500
    ).length;

    aggregatedData.push({
      date: format(timestamp, "yyyy-MM-dd'T'HH:mm:ss"),
      success,
      warning,
      error,
    });
  }

  return aggregatedData;
}

export function LogsChart({ logs }: { logs: Log[] }) {
  const data = aggregateData(logs);

  return (
    <ChartContainer config={chartConfig} className="h-[150px] w-full">
      <BarChart
        accessibilityLayer
        data={data}
        margin={{
          left: 12,
          right: 12,
        }}
      >
        <XAxis
          dataKey="date"
          tickLine={false}
          axisLine={false}
          dx={-25}
          tickFormatter={(value) => {
            const date = new Date(value);
            return date.toLocaleDateString("en-US", {
              hour: "2-digit",
              minute: "2-digit",
              hour12: true,
            });
          }}
        />
        <CartesianGrid strokeDasharray="1" stroke="#DFE2E6" vertical={false} />
        <ChartTooltip
          content={
            <ChartTooltipContent
              className="w-[200px]"
              nameKey="views"
              labelFormatter={(value) => {
                return new Date(value).toLocaleDateString("en-US", {
                  month: "short",
                  day: "numeric",
                  year: "numeric",
                  hour: "2-digit",
                  minute: "2-digit",
                  hour12: true,
                });
              }}
            />
          }
        />
        <Bar dataKey="success" stackId="a" fill="var(--color-success)" />
        <Bar dataKey="warning" stackId="a" fill="var(--color-warning)" />
        <Bar dataKey="error" stackId="a" fill="var(--color-error)" />
      </BarChart>
    </ChartContainer>
  );
}
