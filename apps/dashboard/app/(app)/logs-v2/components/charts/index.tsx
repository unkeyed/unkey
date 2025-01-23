"use client";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { Grid } from "@unkey/icons";
import { useEffect, useRef } from "react";
import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";
import { LogsChartError } from "./components/logs-chart-error";
import { LogsChartLoading } from "./components/logs-chart-loading";
import { useFetchTimeseries } from "./hooks/use-fetch-timeseries";
import { calculateTimePoints } from "./utils/calculate-timepoints";
import { formatTimestampLabel, formatTimestampTooltip } from "./utils/format-timestamp";

const chartConfig = {
  success: {
    label: "Success",
    subLabel: "2xx",
    color: "hsl(var(--accent-4))",
  },
  warning: {
    label: "Warning",
    subLabel: "4xx",
    color: "hsl(var(--warning-9))",
  },
  error: {
    label: "Error",
    subLabel: "5xx",
    color: "hsl(var(--error-9))",
  },
} satisfies ChartConfig;

export function LogsChart({
  onMount,
}: {
  onMount: (distanceToTop: number) => void;
}) {
  const chartRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const distanceToTop = chartRef.current?.getBoundingClientRect().top ?? 0;
    onMount(distanceToTop);
  }, [onMount]);

  const { timeseries, isLoading, isError } = useFetchTimeseries();

  if (isError) {
    return <LogsChartError />;
  }

  if (isLoading) {
    return <LogsChartLoading />;
  }

  return (
    <div className="w-full relative" ref={chartRef}>
      <div className="px-2 text-accent-11 font-mono absolute top-0 text-xxs w-full flex justify-between">
        {timeseries
          ? calculateTimePoints(
              timeseries[0].originalTimestamp ?? Date.now(),
              timeseries.at(-1)?.originalTimestamp ?? Date.now(),
            ).map((time, i) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: use of index is acceptable here.
              <div key={i} className="z-10">
                {formatTimestampLabel(time)}
              </div>
            ))
          : null}
      </div>
      <ResponsiveContainer width="100%" height={50} className="border-b border-gray-4">
        <ChartContainer config={chartConfig}>
          <BarChart
            data={timeseries}
            barGap={2}
            barSize={8}
            margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
          >
            <YAxis domain={[0, (dataMax: number) => dataMax * 1.5]} hide />
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
                if (!active || !payload?.length) {
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
            {["success", "error", "warning"].map((key) => (
              <Bar key={key} dataKey={key} stackId="a" fill={`var(--color-${key})`} />
            ))}
          </BarChart>
        </ChartContainer>
      </ResponsiveContainer>
    </div>
  );
}
