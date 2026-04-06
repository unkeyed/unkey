"use client";

import { type ChartConfig, ChartContainer, ChartTooltip } from "@/components/ui/chart";
import { formatBytesPerSecondParts } from "@/lib/utils/deployment-formatters";
import { cn } from "@unkey/ui/src/lib/utils";
import { Area, AreaChart, YAxis } from "recharts";
import { LogsChartEmpty } from "./components/logs-chart-empty";
import { LogsChartError } from "./components/logs-chart-error";
import { LogsChartLoading } from "./components/logs-chart-loading";

// Stacked area chart for per-pod network rate. Single chart showing both
// directions so you can compare bursts at a glance. Y-axis ticks auto-scale
// via formatBytesPerSecondParts so idle pods get a "B/s" axis and busy
// ones get "MiB/s" without the chart jumping units mid-render.

export type NetworkChartPoint = {
  originalTimestamp: number;
  network_egress: number;
  network_ingress: number;
};

type Props = {
  data: NetworkChartPoint[];
  config: ChartConfig;
  height?: number;
  isLoading?: boolean;
  isError?: boolean;
  chartContainerClassname?: string;
  showDateInTooltip?: boolean;
};

export function NetworkAreaChart({
  data,
  config,
  height = 160,
  isLoading,
  isError,
  chartContainerClassname,
  showDateInTooltip,
}: Props) {
  if (isError) {
    return <LogsChartError />;
  }
  if (isLoading) {
    return <LogsChartLoading />;
  }
  const isEmpty = !data.length || data.every((p) => !p.network_egress && !p.network_ingress);
  if (isEmpty) {
    return <LogsChartEmpty config={config} height={height} />;
  }

  return (
    <ChartContainer
      config={config}
      className={cn("!flex-col aspect-auto w-full", chartContainerClassname)}
      style={{ height, width: "100%" }}
    >
      <AreaChart data={data} margin={{ top: 16, right: 8, bottom: 0, left: 0 }}>
        <YAxis
          width={56}
          tickLine={false}
          axisLine={false}
          tickFormatter={formatYAxisRate}
          tick={{ fill: "hsl(var(--gray-10))", fontSize: 10 }}
          // At least show 1 KiB/s so the baseline isn't compressed to nothing
          // on mostly-idle charts.
          domain={[0, (max: number) => Math.max(max, 1024)]}
          // Nice-round tick values (e.g. 2.5 / 5 / 7.5 MiB/s instead of 2.86
          // / 5.69 / 8.52). Recharts picks bounds for us when we don't force
          // them; `allowDecimals` keeps sub-KiB values legible on idle pods.
        />
        <ChartTooltip
          allowEscapeViewBox={{ x: false, y: true }}
          wrapperStyle={{ zIndex: 1000, pointerEvents: "none" }}
          cursor={{
            stroke: "hsl(var(--accent-9))",
            strokeWidth: 1,
            strokeDasharray: "4 4",
            strokeOpacity: 0.7,
          }}
          content={({ active, payload }) => {
            if (!active || !payload?.length) {
              return null;
            }
            // Pull values straight from the data point so we always render
            // both series, even when recharts' payload drops zero-valued
            // items under stacked areas (which it sometimes does).
            const point = payload[0]?.payload as NetworkChartPoint | undefined;
            if (!point) {
              return null;
            }
            const labelText = formatCompactInterval(
              point.originalTimestamp,
              data,
              showDateInTooltip,
            );
            // Egress on top (drawn above ingress in the stack), so egress
            // above ingress in the tooltip reads like the visual order.
            const rows: Array<{ key: keyof NetworkChartPoint; value: number }> = [
              { key: "network_egress", value: point.network_egress ?? 0 },
              { key: "network_ingress", value: point.network_ingress ?? 0 },
            ];
            return (
              <div
                role="tooltip"
                className="grid items-start gap-1 rounded-lg border border-gray-4 bg-gray-1 px-3 py-2 text-xs shadow-2xl select-none w-max max-w-[240px]"
              >
                <div className="font-medium text-[11px] text-accent-11">{labelText}</div>
                <div className="grid gap-1">
                  {rows.map(({ key, value }) => {
                    const itemConfig = config[key];
                    const parts = formatBytesPerSecondParts(value);
                    return (
                      <div key={key} className="flex items-center gap-2">
                        <div
                          className="shrink-0 rounded-[2px] h-2 w-2"
                          style={{ backgroundColor: itemConfig?.color }}
                        />
                        <span className="text-accent-12">{itemConfig?.label ?? key}</span>
                        <span className="font-mono tabular-nums text-accent-12 ml-auto">
                          {parts.value}
                          {parts.unit && ` ${parts.unit}`}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>
            );
          }}
        />
        {Object.keys(config).map((key) => (
          <Area
            key={key}
            dataKey={key}
            type="monotone"
            stackId="a"
            stroke={config[key].color}
            strokeWidth={1.5}
            fill={config[key].color}
            fillOpacity={0.25}
            isAnimationActive={false}
          />
        ))}
      </AreaChart>
    </ChartContainer>
  );
}

function formatYAxisRate(v: number): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "0";
  }
  const parts = formatBytesPerSecondParts(v);
  return `${parts.value}${parts.unit ? ` ${parts.unit}` : ""}`;
}

function formatCompactInterval(
  startTs: number | undefined,
  data: NetworkChartPoint[],
  withDate?: boolean,
): string {
  if (typeof startTs !== "number" || !Number.isFinite(startTs)) {
    return "";
  }
  const idx = data.findIndex((p) => p.originalTimestamp === startTs);
  const nextTs = idx >= 0 ? data[idx + 1]?.originalTimestamp : undefined;
  const start = new Date(startTs);
  const end = typeof nextTs === "number" ? new Date(nextTs) : null;
  const timeFmt: Intl.DateTimeFormatOptions = { hour: "numeric", minute: "2-digit" };
  const dateFmt: Intl.DateTimeFormatOptions = { month: "short", day: "numeric" };
  const startStr = withDate
    ? `${start.toLocaleDateString(undefined, dateFmt)}, ${start.toLocaleTimeString(undefined, timeFmt)}`
    : start.toLocaleTimeString(undefined, timeFmt);
  if (!end) {
    return startStr;
  }
  const sameDay = start.toDateString() === end.toDateString();
  const endStr =
    withDate && !sameDay
      ? `${end.toLocaleDateString(undefined, dateFmt)}, ${end.toLocaleTimeString(undefined, timeFmt)}`
      : end.toLocaleTimeString(undefined, timeFmt);
  return `${startStr} – ${endStr}`;
}
