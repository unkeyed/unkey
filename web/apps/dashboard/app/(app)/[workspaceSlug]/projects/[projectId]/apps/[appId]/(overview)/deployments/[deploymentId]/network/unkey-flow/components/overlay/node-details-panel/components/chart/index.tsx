"use client";

import { ChartEmpty } from "@/components/logs/chart/chart-states";
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
import { useId, useMemo } from "react";
import { Bar, BarChart } from "recharts";
import type { ValueParts } from "./area-timeseries-chart";
import { ChartError } from "./components/chart-error";
import { ChartWaveLoading } from "./components/chart-wave-loading";

export type TooltipValueParts = ValueParts;

type LogsTimeseriesBarChartProps = {
  data?: TimeseriesData[];
  config: ChartConfig;
  height?: number;
  isLoading?: boolean;
  isError?: boolean;
  chartContainerClassname?: string;
  valueFormatter?: (value: number) => string;
  formatTooltipValue?: (value: number) => TooltipValueParts;
  showDateInTooltip?: boolean;
  // Fill bars with a flat color instead of the default top-down gradient.
  solidFill?: boolean;
};

export function LogsTimeseriesBarChart({
  data,
  config,
  height = 50,
  isLoading,
  isError,
  chartContainerClassname,
  valueFormatter,
  formatTooltipValue,
  showDateInTooltip,
  solidFill,
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

  const chartId = useId().replace(/:/g, "");

  if (isError) {
    return <ChartError height={height} />;
  }

  const configKeys = Object.keys(config);
  const isEmpty =
    !data || data.length === 0 || data.every((p) => configKeys.every((k) => !(Number(p[k]) > 0)));

  const firstKey = configKeys[0];
  const sectionColor = config[firstKey]?.color;

  if (isLoading) {
    return <ChartWaveLoading height={height} color={sectionColor} />;
  }

  if (isEmpty) {
    return (
      <ChartEmpty variant="wave" color={sectionColor} height={height} message="No activity yet" />
    );
  }

  return (
    <ChartContainer
      config={config}
      className={cn(
        "!flex-col aspect-auto w-full border-b border-grayA-4 !overflow-visible [&_.recharts-responsive-container]:!overflow-visible [&_.recharts-wrapper]:!overflow-visible",
        chartContainerClassname,
      )}
      style={{ height, width: "100%" }}
    >
      <BarChart data={data} margin={{ top: 0, right: 0, bottom: 0, left: 0 }} barCategoryGap={0.5}>
        <defs>
          {configKeys.map((key) => (
            <linearGradient
              key={`${chartId}-${key}`}
              id={`${chartId}-bar-${key}`}
              x1="0"
              y1="0"
              x2="0"
              y2="1"
            >
              <stop offset="0%" stopColor={config[key].color} stopOpacity={0.9} />
              <stop offset="100%" stopColor={config[key].color} stopOpacity={0.3} />
            </linearGradient>
          ))}
        </defs>
        <ChartTooltip
          offset={15}
          allowEscapeViewBox={{ x: true, y: true }}
          wrapperStyle={{ zIndex: 1000, pointerEvents: "none", overflow: "visible" }}
          cursor={{ fill: "hsl(var(--accent-3))", fillOpacity: 0.4 }}
          content={({ active, payload, label }) => {
            if (!active || !payload?.length || payload?.[0]?.payload.total === 0) {
              return null;
            }
            const allZero = payload.every((p) => typeof p?.value === "number" && p.value === 0);
            if (allZero) {
              return null;
            }
            const tooltipFormatter = formatTooltipValue ?? valueFormatter;
            if (tooltipFormatter) {
              const payloadTimestamp = parseTimestamp(payload?.[0]?.payload?.originalTimestamp);
              const labelText = formatCompactInterval(payloadTimestamp, data, showDateInTooltip);
              return (
                <div
                  role="tooltip"
                  className="grid items-start gap-1.5 rounded-xl border border-gray-4/50 bg-gray-1/80 backdrop-blur-md px-3 py-2.5 text-xs shadow-2xl select-none w-max max-w-[240px] animate-in fade-in-0 zoom-in-95 duration-150"
                >
                  <div className="font-medium text-[11px] text-accent-11">{labelText}</div>
                  <div className="grid gap-1">
                    {payload.map((item, i) => {
                      const dataKey = String(item?.dataKey ?? i);
                      const itemConfig = config[dataKey];
                      const seriesLabel = itemConfig?.label ?? item?.name ?? dataKey;
                      const raw = typeof item?.value === "number" ? item.value : 0;
                      const result = tooltipFormatter(raw);
                      const isStructured = typeof result === "object";
                      return (
                        <div key={dataKey} className="flex items-center gap-2">
                          <div
                            className="shrink-0 rounded-[2px] h-2 w-2"
                            style={{ backgroundColor: itemConfig?.color }}
                          />
                          <span className="text-accent-12">{seriesLabel}</span>
                          <span className="font-mono tabular-nums text-accent-12 ml-auto">
                            {isStructured ? (
                              <>
                                {result.value}
                                {result.unit && ` ${result.unit}`}
                                {result.hint && (
                                  <span className="text-grayA-9 ml-1 font-normal">
                                    {result.hint}
                                  </span>
                                )}
                              </>
                            ) : (
                              result
                            )}
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
                color={config[configKeys[0]]?.color}
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
        {configKeys.map((key) => (
          <Bar
            key={key}
            dataKey={key}
            stackId="a"
            fill={solidFill ? config[key].color : `url(#${chartId}-bar-${key})`}
            radius={[2, 2, 0, 0]}
          />
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
