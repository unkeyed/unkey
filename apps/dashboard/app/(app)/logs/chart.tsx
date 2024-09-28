"use client";

import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";

export const description = "An interactive bar chart showing API requests";

const generateHourlyData = () => {
  const data = [];
  const baseDate = new Date(); // Using the current date
  for (let i = 0; i < 24 * 12; i++) {
    // 24 hours * 12 (5-minute intervals) = 288 data points
    const date = new Date(baseDate.getTime() + i * (5 * 60000)); // Add 5 minutes
    data.push({
      date: date.getTime(), // Store as milliseconds
      getRequests: Math.floor(Math.random() * 2000) + 10,
    });
  }
  return data;
};

const chartData = generateHourlyData();

const chartConfig = {
  requests: {
    label: "API Requests",
  },
  getRequests: {
    label: "GET",
    color: "hsl(var(--chart-3))",
  },
} satisfies ChartConfig;

export function ChartsComp() {
  return (
    <ChartContainer
      config={chartConfig}
      className="aspect-auto h-[100px] w-full"
    >
      <BarChart accessibilityLayer data={chartData}>
        <CartesianGrid vertical={false} />
        <XAxis
          dataKey="date"
          tickLine={false}
          axisLine={false}
          tickMargin={8}
          minTickGap={32}
          tickSize={12}
          tickFormatter={(value) => {
            const date = new Date(value);
            return date.toLocaleTimeString([], {
              hour: "2-digit",
              minute: "2-digit",
              hour12: true,
            });
          }}
        />
        <ChartTooltip
          content={
            <ChartTooltipContent
              className="w-[200px]"
              nameKey="requests"
              labelFormatter={(value) => {
                return new Date(value).toLocaleDateString("en-US", {
                  month: "short",
                  day: "numeric",
                  year: "numeric",
                });
              }}
            />
          }
        />
        <Bar
          dataKey="getRequests"
          fill="var(--color-getRequests})"
          radius={[2, 2, 0, 0]}
        />
      </BarChart>
    </ChartContainer>
  );
}
