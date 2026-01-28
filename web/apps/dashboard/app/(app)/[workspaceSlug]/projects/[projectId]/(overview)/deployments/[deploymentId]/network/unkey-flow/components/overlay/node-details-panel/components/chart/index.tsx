"use client";

import type { TimeseriesData } from "@/components/logs/overview-charts/types";
import { parseTimestamp } from "@/components/logs/parse-timestamp";
import { formatTooltipInterval } from "@/components/logs/utils";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { cn } from "@unkey/ui/src/lib/utils";
import { useMemo } from "react";
import { Bar, BarChart } from "recharts";
import { LogsChartError } from "./components/logs-chart-error";
import { LogsChartLoading } from "./components/logs-chart-loading";

type LogsTimeseriesBarChartProps = {
  data?: TimeseriesData[];
  config: ChartConfig;
  height?: number;
  isLoading?: boolean;
  isError?: boolean;
  chartContainerClassname?: string;
};

export function LogsTimeseriesBarChart({
  data,
  config,
  height = 50,
  isLoading,
  isError,
  chartContainerClassname,
}: LogsTimeseriesBarChartProps) {
  // // Precompute timestamp-to-index mapping for O(1) lookup
  const timestampToIndexMap = useMemo(() => {
    const map = new Map<number, number>();
    if (data?.length) {
      data.forEach((item, index) => {
        if (item?.originalTimestamp) {
          const normalizedTimestamp = parseTimestamp(item.originalTimestamp);
          if (Number.isFinite(normalizedTimestamp)) {
            map.set(normalizedTimestamp, index);
          }
        }
      });
    }
    return map;
  }, [data]);

  if (isError) {
    return <LogsChartError />;
  }

  if (isLoading) {
    return <LogsChartLoading />;
  }

  return (
    <ChartContainer
      config={config}
      className={cn("aspect-auto w-full border-b border-grayA-4", chartContainerClassname)}
      style={{ height }}
    >
      <BarChart data={data} margin={{ top: 0, right: 0, bottom: 0, left: 0 }} barCategoryGap={0.5}>
        <ChartTooltip
          position={{ y: 50 }}
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
                className="rounded-lg shadow-lg border border-gray-4"
                labelFormatter={(_, tooltipPayload) => {
                  const payloadTimestamp = tooltipPayload?.[0]?.payload?.originalTimestamp;
                  return formatTooltipInterval(
                    payloadTimestamp,
                    data || [],
                    undefined,
                    timestampToIndexMap,
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
  );
}
