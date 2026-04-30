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
import { LogsChartEmpty } from "./components/logs-chart-empty";
import { LogsChartError } from "./components/logs-chart-error";
import { LogsChartLoading } from "./components/logs-chart-loading";

type LogsTimeseriesBarChartProps = {
  data?: TimeseriesData[];
  config: ChartConfig;
  height?: number;
  isLoading?: boolean;
  isError?: boolean;
  chartContainerClassname?: string;
  // Optional per-value tooltip formatter. Receives the raw numeric value for
  // the hovered bar and returns the string shown after the series name.
  // Lets callers render "67m (27%)" instead of the raw number.
  valueFormatter?: (value: number) => string;
  // When true, the compact-tooltip label includes the date (e.g. "Apr 17,
  // 4:14 AM"). For multi-day ranges a bare "4:14 AM" is ambiguous.
  showDateInTooltip?: boolean;
};

export function LogsTimeseriesBarChart({
  data,
  config,
  height = 50,
  isLoading,
  isError,
  chartContainerClassname,
  valueFormatter,
  showDateInTooltip,
}: LogsTimeseriesBarChartProps) {
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

  const isEmpty = !data || data.length === 0;

  if (isError) {
    return <LogsChartError />;
  }

  if (isLoading) {
    return <LogsChartLoading />;
  }

  if (isEmpty) {
    return <LogsChartEmpty config={config} height={height} />;
  }

  return (
    <ChartContainer
      config={config}
      // ChartContainer's default classes include `flex justify-center` which
      // can stop the chart from stretching full-width. Force block + full
      // width explicitly so the bars span the entire panel.
      className={cn(
        "!flex-col aspect-auto w-full border-b border-grayA-4",
        chartContainerClassname,
      )}
      style={{ height, width: "100%" }}
    >
      <BarChart data={data} margin={{ top: 0, right: 0, bottom: 0, left: 0 }} barCategoryGap={0.5}>
        <ChartTooltip
          position={{ y: 50 }}
          // y:true lets the tooltip render below the short (48px) chart row
          // without vertical clipping. x stays clamped to the chart width so
          // the tooltip can't spill off the browser viewport when the panel
          // sits flush with the right edge.
          allowEscapeViewBox={{ x: false, y: true }}
          wrapperStyle={{ zIndex: 1000, pointerEvents: "none" }}
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
            // Suppress tooltip on zero-valued bars. These happen when a 15s
            // bucket only got a single heimdall sample (CPU delta = 0) or
            // when the live-tip bucket hasn't received any checkpoints yet.
            // Showing "0%" on these is misleading — there's just no data.
            // Check across every series in the payload, not just the first
            // stack: a stacked chart can have ingress=0 with egress=non-zero
            // (or vice versa) and we still want the tooltip in that case.
            const allZero = payload.every((p) => typeof p?.value === "number" && p.value === 0);
            if (allZero) {
              return null;
            }
            // Custom tooltip body when a valueFormatter is provided so the
            // caller can return non-numeric strings ("0.4%", "43 MiB (17%)").
            // Default path falls back to the shared ChartTooltipContent which
            // only handles numeric values.
            if (valueFormatter) {
              // parseTimestamp normalises string/number inputs to a number
              // the same way timestampToIndexMap does upstream — without
              // this normalisation a string-shaped originalTimestamp falls
              // through formatCompactInterval's `typeof === "number"` guard
              // and the tooltip header goes blank.
              const payloadTimestamp = parseTimestamp(payload?.[0]?.payload?.originalTimestamp);
              const labelText = formatCompactInterval(payloadTimestamp, data, showDateInTooltip);
              return (
                <div
                  role="tooltip"
                  className="grid items-start gap-1 rounded-lg border border-gray-4 bg-gray-1 px-3 py-2 text-xs shadow-2xl select-none w-max max-w-[240px]"
                >
                  <div className="font-medium text-[11px] text-accent-11">{labelText}</div>
                  <div className="grid gap-1">
                    {payload.map((item, i) => {
                      const dataKey = String(item?.dataKey ?? i);
                      const itemConfig = config[dataKey];
                      const seriesLabel = itemConfig?.label ?? item?.name ?? dataKey;
                      const raw = typeof item?.value === "number" ? item.value : 0;
                      const color = item?.color ?? itemConfig?.color;
                      return (
                        <div key={dataKey} className="flex items-center gap-2">
                          <div
                            className="shrink-0 rounded-[2px] h-2 w-2"
                            style={{ backgroundColor: color }}
                          />
                          <span className="text-accent-12">{seriesLabel}</span>
                          <span className="font-mono tabular-nums text-accent-12 ml-auto">
                            {valueFormatter(raw)}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </div>
              );
            }
            return (
              <ChartTooltipContent
                payload={payload}
                label={label}
                active={active}
                className="rounded-lg shadow-lg border border-gray-4"
                labelFormatter={(_, tooltipPayload) => {
                  const payloadTimestamp = parseTimestamp(
                    tooltipPayload?.[0]?.payload?.originalTimestamp,
                  );
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

// Compact label like "4:14 – 4:15 AM" — about a third as wide as the
// original "Apr 17, 4:14 AM - Apr 17, 4:15 AM (GMT+2)" so the tooltip
// doesn't overflow a narrow panel. For multi-day ranges the bucket start
// also shows the date ("Apr 17, 4:14 – 5:14 AM") so a bare time isn't
// ambiguous across days.
function formatCompactInterval(
  startTs: unknown,
  data: TimeseriesData[] | undefined,
  withDate?: boolean,
): string {
  if (typeof startTs !== "number" || !Number.isFinite(startTs)) {
    return "";
  }
  const start = new Date(startTs);
  const nextTs = findNextTimestamp(startTs, data);
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

function findNextTimestamp(
  startTs: number,
  data: TimeseriesData[] | undefined,
): number | undefined {
  if (!data?.length) {
    return undefined;
  }
  for (let i = 0; i < data.length; i++) {
    const ts = data[i]?.originalTimestamp;
    if (typeof ts === "number" && ts === startTs) {
      const next = data[i + 1]?.originalTimestamp;
      return typeof next === "number" ? next : undefined;
    }
  }
  return undefined;
}
