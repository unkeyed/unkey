"use client";

import { ChartError } from "@/components/charts/components/chart-error";
import { ChartWaveLoading } from "@/components/charts/components/chart-wave-loading";
import { ChartEmpty } from "@/components/logs/chart/chart-states";
import { type ChartConfig, ChartContainer, ChartTooltip } from "@/components/ui/chart";
import { cn } from "@unkey/ui/src/lib/utils";
import { useId } from "react";
import {
  Area,
  Bar,
  CartesianGrid,
  ComposedChart,
  Line,
  ReferenceLine,
  XAxis,
  YAxis,
} from "recharts";
import { type SeriesKey, type SeriesPoint, formatBucketLabel, formatTickTime } from "../lib/series";

export type DeployMarker = { id: string; x: number; label: string; title?: string };

type Props = {
  points: SeriesPoint[];
  series: SeriesKey[];
  kind: "area" | "bar" | "line";
  stacked?: boolean;
  height?: number;
  domain: [number, number];
  bucketSeconds: number;
  markers?: DeployMarker[];
  showMarkerLabels?: boolean;
  formatValue: (v: number) => string;
  formatTick?: (v: number) => string;
  isLoading?: boolean;
  isError?: boolean;
  showAxes?: boolean;
  yMax?: number;
  className?: string;
  emptyMessage?: string;
  // Charts sharing a syncId show the hover cursor and tooltip at the same
  // time across all of them. Sparklines pass none.
  syncId?: string;
};

const Y_GUTTER_PX = 40;

// One chart for all seven metrics. The kind decides the mark, the series list
// decides colours, and the marker list draws the deploy lines that every chart
// on the page shares, so a spike and the deploy that preceded it line up in
// the same x position across cards.
export function MetricsChart({
  points,
  series,
  kind,
  stacked,
  height = 180,
  domain,
  bucketSeconds,
  markers = [],
  showMarkerLabels = true,
  formatValue,
  formatTick,
  isLoading,
  isError,
  showAxes = true,
  yMax,
  className,
  emptyMessage = "No data in this range",
  syncId,
}: Props) {
  const chartId = useId().replace(/:/g, "");

  if (isError) {
    return <ChartError height={height} />;
  }
  const primaryColor = series[0]?.color;
  if (isLoading) {
    return <ChartWaveLoading height={height} color={primaryColor} />;
  }
  const keys = series.map((s) => s.key);
  const hasData = points.some((p) => keys.some((k) => (p[k] ?? 0) > 0));
  if (!hasData) {
    return (
      <ChartEmpty variant="wave" color={primaryColor} height={height} message={emptyMessage} />
    );
  }

  const config: ChartConfig = Object.fromEntries(
    series.map((s) => [s.key, { label: s.label, color: s.color }]),
  );

  const observedMax = points.reduce((m, p) => {
    if (stacked) {
      return Math.max(
        m,
        keys.reduce((sum, k) => sum + (p[k] ?? 0), 0),
      );
    }
    return Math.max(m, ...keys.map((k) => p[k] ?? 0));
  }, 0);
  const top = yMax ?? observedMax * 1.15;
  const yTicks = [0, top / 3, (2 * top) / 3, top];
  const spanMs = domain[1] - domain[0];
  const xTicks = [0, 1, 2, 3, 4].map((i) => Math.round(domain[0] + (i * spanMs) / 4));
  const tickFormatter = formatTick ?? formatValue;
  const bucketMs = bucketSeconds * 1000;

  return (
    <ChartContainer
      config={config}
      className={cn("!flex-col aspect-auto w-full", className)}
      style={{ height, width: "100%" }}
    >
      <ComposedChart
        data={points}
        syncId={syncId}
        syncMethod="value"
        margin={
          showAxes
            ? { top: 12, right: 8, bottom: 0, left: 0 }
            : { top: 4, right: 0, bottom: 0, left: 0 }
        }
        barCategoryGap={1}
      >
        <defs>
          {series.map((s) => (
            <linearGradient key={s.key} id={`${chartId}-${s.key}`} x1="0" y1="0" x2="0" y2="1">
              <stop
                offset="0%"
                stopColor={s.color}
                stopOpacity={kind === "bar" ? 0.9 : stacked ? 0.55 : 0.35}
              />
              <stop
                offset="100%"
                stopColor={s.color}
                stopOpacity={kind === "bar" ? 0.45 : stacked ? 0.25 : 0.03}
              />
            </linearGradient>
          ))}
        </defs>
        {showAxes && (
          <CartesianGrid
            vertical={false}
            stroke="hsl(var(--gray-4))"
            strokeDasharray="3 3"
            strokeOpacity={0.6}
          />
        )}
        <XAxis
          dataKey="x"
          type="number"
          scale="time"
          domain={[domain[0], domain[1]]}
          allowDataOverflow
          ticks={xTicks}
          tickFormatter={(v: number) => formatTickTime(v, spanMs)}
          tick={showAxes ? { fill: "hsl(var(--gray-10))", fontSize: 10 } : false}
          tickLine={false}
          axisLine={false}
          minTickGap={40}
          hide={!showAxes}
        />
        <YAxis
          width={showAxes ? Y_GUTTER_PX : 0}
          domain={[0, top]}
          ticks={yTicks}
          tickFormatter={(v: number) => (v <= 0 ? "" : tickFormatter(v))}
          tick={showAxes ? { fill: "hsl(var(--gray-10))", fontSize: 10 } : false}
          tickLine={false}
          axisLine={false}
          hide={!showAxes}
        />
        <ChartTooltip
          allowEscapeViewBox={{ x: false, y: true }}
          wrapperStyle={{ zIndex: 1000, pointerEvents: "none" }}
          cursor={
            kind === "bar"
              ? { fill: "hsl(var(--accent-3))", fillOpacity: 0.4 }
              : {
                  stroke: "hsl(var(--accent-9))",
                  strokeWidth: 1,
                  strokeDasharray: "3 3",
                  strokeOpacity: 0.5,
                }
          }
          content={({ active, payload }) => {
            if (!active || !payload?.length) {
              return null;
            }
            const point = payload[0]?.payload as SeriesPoint | undefined;
            if (!point) {
              return null;
            }
            const rows = series
              .map((s) => ({ ...s, value: point[s.key] ?? 0 }))
              .sort((a, b) => b.value - a.value);
            const deploysHere = markers.filter((m) => m.x >= point.x && m.x < point.x + bucketMs);
            return (
              <div
                role="tooltip"
                className="grid items-start gap-1.5 rounded-xl border border-gray-4/50 bg-gray-1/80 backdrop-blur-md px-3 py-2.5 text-xs shadow-2xl select-none w-max max-w-[280px]"
              >
                <div className="font-medium text-[11px] text-accent-11">
                  {formatBucketLabel(point.x, bucketSeconds)}
                </div>
                <div className="grid gap-1">
                  {rows.map((r) => (
                    <div key={r.key} className="flex items-center gap-2">
                      <div
                        className="shrink-0 rounded-[2px] h-2 w-2"
                        style={{ backgroundColor: r.color }}
                      />
                      <span className="text-accent-12 truncate max-w-[150px]">{r.label}</span>
                      <span className="font-mono tabular-nums text-accent-12 ml-auto">
                        {formatValue(r.value)}
                      </span>
                    </div>
                  ))}
                </div>
                {deploysHere.length > 0 && (
                  <div className="border-t border-gray-4 pt-1.5 mt-0.5 grid gap-0.5">
                    {deploysHere.map((m) => (
                      <div key={`${m.x}-${m.label}`} className="flex items-center gap-2">
                        <span className="text-[10px] text-gray-11">Deployed</span>
                        <span className="font-mono text-[11px] text-gray-12">{m.label}</span>
                        {m.title && (
                          <span className="text-[11px] text-gray-11 truncate max-w-[160px]">
                            {m.title}
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            );
          }}
        />
        {series.map((s) => {
          if (kind === "bar") {
            return (
              <Bar
                key={s.key}
                dataKey={s.key}
                stackId={stacked ? "stack" : undefined}
                fill={`url(#${chartId}-${s.key})`}
                stroke={s.color}
                strokeWidth={0}
                isAnimationActive={false}
              />
            );
          }
          if (kind === "line") {
            return (
              <Line
                key={s.key}
                dataKey={s.key}
                type="monotone"
                stroke={s.color}
                strokeWidth={1.5}
                dot={false}
                isAnimationActive={false}
              />
            );
          }
          return (
            <Area
              key={s.key}
              dataKey={s.key}
              type="monotone"
              stackId={stacked ? "stack" : undefined}
              stroke={s.color}
              strokeWidth={stacked ? 0.75 : 1.5}
              fill={`url(#${chartId}-${s.key})`}
              fillOpacity={1}
              dot={false}
              isAnimationActive={false}
            />
          );
        })}
        {markers.map((m) => (
          <ReferenceLine
            key={`${m.x}-${m.label}`}
            x={m.x}
            stroke="hsl(var(--gray-9))"
            strokeDasharray="2 3"
            strokeWidth={1}
            ifOverflow="hidden"
            label={
              showMarkerLabels && showAxes
                ? {
                    value: m.label,
                    position: "insideTopLeft",
                    fill: "hsl(var(--gray-10))",
                    fontSize: 9,
                    fontFamily: "ui-monospace, monospace",
                    offset: 6,
                  }
                : undefined
            }
          />
        ))}
      </ComposedChart>
    </ChartContainer>
  );
}
