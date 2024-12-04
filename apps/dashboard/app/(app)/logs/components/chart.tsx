"use client";

import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { format } from "date-fns";
import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";
import { useLogSearchParams } from "../query-state";

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

export function LogsChart({ logs }: { logs: Log[] }) {
  const { searchParams } = useLogSearchParams();
  const data = aggregateData(logs, searchParams.startTime, searchParams.endTime);

  return (
    <ChartContainer config={chartConfig} className="h-[125px] w-full">
      <BarChart accessibilityLayer data={data}>
        <XAxis
          dataKey="date"
          tickLine={false}
          axisLine={false}
          dx={-40}
          tickFormatter={(value) => {
            const date = new Date(value);
            return date.toLocaleDateString("en-US", {
              month: "short",
              day: "2-digit",
              hour: "2-digit",
              minute: "2-digit",
              hour12: true,
            });
          }}
        />
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
        <Bar dataKey="success" stackId="a" fill="var(--color-success)" radius={3} />
        <Bar dataKey="warning" stackId="a" fill="var(--color-warning)" radius={3} />
        <Bar dataKey="error" stackId="a" fill="var(--color-error)" radius={3} />
      </BarChart>
    </ChartContainer>
  );
}

function aggregateData(data: Log[], startTime: number, endTime: number) {
  const aggregatedData: {
    date: string;
    success: number;
    warning: number;
    error: number;
  }[] = [];

  const intervalMs = 60 * 1000 * 10; // 10 minutes

  if (data.length === 0) {
    return aggregatedData;
  }

  const buckets = new Map();

  // Create a bucket for each 10 minute interval
  for (let timestamp = startTime; timestamp < endTime; timestamp += intervalMs) {
    buckets.set(timestamp, {
      date: format(timestamp, "yyyy-MM-dd'T'HH:mm:ss"),
      success: 0,
      warning: 0,
      error: 0,
    });
  }

  // For each log, find its bucket then increment the appropriate counter
  for (const log of data) {
    const bucketIndex = Math.floor((log.time - startTime) / intervalMs);
    const bucket = buckets.get(startTime + bucketIndex * intervalMs);

    if (bucket) {
      const status = log.response_status;
      if (status >= 200 && status < 300) {
        bucket.success++;
      } else if (status >= 400 && status < 500) {
        bucket.warning++;
      } else if (status >= 500) {
        bucket.error++;
      }
    }
  }

  return Array.from(buckets.values());
}
