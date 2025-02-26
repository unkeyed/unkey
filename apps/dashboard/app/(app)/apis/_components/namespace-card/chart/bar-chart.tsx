"use client";

import { formatTimestampTooltip } from "@/components/logs/chart/utils/format-timestamp";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { Grid } from "@unkey/icons";
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, YAxis } from "recharts";
import { LogsChartError } from "./components/logs-chart-error";
import { LogsChartLoading } from "./components/logs-chart-loading";

type VerificationTimeseriesData = {
  displayX: string;
  originalTimestamp: number;
  valid: number;
  insufficient_permissions: number;
  rate_limited: number;
  forbidden: number;
  disabled: number;
  expired: number;
  usage_exceeded: number;
  total: number;
  success?: number; // Alias for valid
  error?: number; // Sum of error outcomes
};

type VerificationTimeseriesBarChartProps = {
  data?: VerificationTimeseriesData[];
  config: ChartConfig;
  isLoading?: boolean;
  isError?: boolean;
};

export function VerificationTimeseriesBarChart({
  data,
  config,
  isError,
  isLoading,
}: VerificationTimeseriesBarChartProps) {
  if (isError) {
    return <LogsChartError />;
  }
  if (isLoading) {
    return <LogsChartLoading />;
  }

  return (
    <ResponsiveContainer width="100%" height="100%">
      <ChartContainer config={config}>
        <BarChart data={data} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
          <YAxis domain={["auto", (dataMax: number) => dataMax * 1.3]} hide />
          <CartesianGrid
            horizontal
            vertical={false}
            strokeDasharray="3 3"
            stroke="hsl(var(--gray-6))"
            strokeOpacity={0.3}
            strokeWidth={1}
            fill="hsl(var(--gray-1))"
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
                              {payload[0]?.payload?.total}
                            </span>
                          </div>
                        </div>
                      </div>
                    </div>
                  }
                  className="rounded-lg shadow-lg border border-gray-4"
                  labelFormatter={(_, tooltipPayload) => {
                    const originalTimestamp = tooltipPayload[0]?.payload?.originalTimestamp;
                    return originalTimestamp ? (
                      <div>
                        <span className="font-mono text-accent-9 text-xs px-4">
                          {formatTimestampTooltip(originalTimestamp)}
                        </span>
                      </div>
                    ) : (
                      ""
                    );
                  }}
                />
              );
            }}
          />
          {Object.keys(config).map((key) => (
            <Bar key={key} dataKey={key} stackId="a" fill={config[key].color} />
          ))}
        </BarChart>
      </ChartContainer>
    </ResponsiveContainer>
  );
}
