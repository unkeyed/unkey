"use client";

import { ChartEmpty } from "@/components/logs/chart/chart-states";
import { type ChartConfig, ChartContainer, ChartTooltip } from "@/components/ui/chart";
import { formatBytesPerSecondParts } from "@/lib/utils/deployment-formatters";
import { cn } from "@unkey/ui/src/lib/utils";
import { useEffect, useId, useState } from "react";
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { resolveXAxisDomain } from "./chart-domain";
import { ChartError } from "./components/chart-error";
import { ChartWaveLoading } from "./components/chart-wave-loading";
import { formatBucketInterval } from "./format-interval";

export type AreaChartPoint = { originalTimestamp: number } & {
  [k: string]: number | undefined;
};

function isAreaChartPoint(value: unknown): value is AreaChartPoint {
  return (
    typeof value === "object" &&
    value !== null &&
    "originalTimestamp" in value &&
    typeof value.originalTimestamp === "number"
  );
}

function activePointFromState(state: unknown, data: AreaChartPoint[]): AreaChartPoint | null {
  if (typeof state !== "object" || state === null) {
    return null;
  }
  const raw =
    "activeTooltipIndex" in state
      ? state.activeTooltipIndex
      : "activeIndex" in state
        ? state.activeIndex
        : undefined;
  let idx = Number.NaN;
  if (typeof raw === "number") {
    idx = raw;
  } else if (typeof raw === "string") {
    idx = Number.parseInt(raw, 10);
  }
  if (!Number.isInteger(idx) || idx < 0 || idx >= data.length) {
    return null;
  }
  const candidate = data[idx];
  return isAreaChartPoint(candidate) ? candidate : null;
}

const Y_GUTTER_PX = 36;

export type ValueParts = { value: string; unit?: string; hint?: string };

export type AreaTimeseriesAxisOptions = {
  visible?: boolean;
  x?: {
    domain?: [number, number];
    contractOnSparseData?: boolean;
    utc?: boolean;
  };
  y?: {
    floor?: number;
    formatTick?: (value: number) => string;
  };
};

type Props = {
  data: AreaChartPoint[];
  config: ChartConfig;
  height?: number;
  isLoading?: boolean;
  isError?: boolean;
  chartContainerClassname?: string;
  showDateInTooltip?: boolean;
  formatTooltipValue?: (value: number) => ValueParts;
  axis?: AreaTimeseriesAxisOptions | null;
  paleFill?: boolean;
  fillColors?: Record<string, string>;
  onActiveChange?: (point: AreaChartPoint | null) => void;
  hideTooltip?: boolean;
  showZeroLine?: boolean;
};

export function AreaTimeseriesChart({
  data,
  config,
  height = 140,
  isLoading,
  isError,
  chartContainerClassname,
  showDateInTooltip,
  formatTooltipValue = formatBytesPerSecondParts,
  axis,
  paleFill,
  fillColors,
  onActiveChange,
  hideTooltip,
  showZeroLine,
}: Props) {
  const handleActive = onActiveChange
    ? (state: unknown) => onActiveChange(activePointFromState(state, data))
    : undefined;
  const chartId = useId().replace(/:/g, "");
  const [shouldAnimate, setShouldAnimate] = useState(true);
  useEffect(() => {
    const timer = setTimeout(() => setShouldAnimate(false), 600);
    return () => clearTimeout(timer);
  }, []);

  if (isError) {
    return <ChartError height={height} />;
  }

  const configKeys = Object.keys(config);
  const firstKey = configKeys[0];
  const sectionColor = config[firstKey]?.color;

  if (isLoading) {
    return <ChartWaveLoading height={height} color={sectionColor} />;
  }
  const hasPositiveValue = data.some((point) => configKeys.some((key) => Number(point[key]) > 0));
  const isAllZero =
    data.length > 0 && data.every((point) => configKeys.every((key) => Number(point[key]) === 0));
  const renderZeroLine = Boolean(showZeroLine && isAllZero);
  if (!data.length || (!hasPositiveValue && !renderZeroLine)) {
    return (
      <ChartEmpty variant="wave" color={sectionColor} height={height} message="No activity yet" />
    );
  }

  const dataMax = data.reduce(
    (m, p) => configKeys.reduce((mm, k) => Math.max(mm, Number(p[k]) || 0), m),
    0,
  );
  const yAxisFloor = axis === null ? 0 : (axis?.y?.floor ?? 1024);
  const formatYTick = axis?.y?.formatTick ?? formatYAxisCompactBytes;
  const top = renderZeroLine ? Math.max(yAxisFloor, 1) : Math.max(dataMax, yAxisFloor) * 1.1;
  const yTicks = [0, top / 3, (2 * top) / 3, top];
  const yDomain: [number, number] = [0, top];

  let firstNonZeroTs: number | undefined;
  let lastNonZeroTs: number | undefined;
  for (const p of data) {
    if (configKeys.some((k) => Number(p[k]) > 0)) {
      firstNonZeroTs ??= p.originalTimestamp;
      lastNonZeroTs = p.originalTimestamp;
    }
  }
  const { effectiveDomain, spanMs } = resolveXAxisDomain({
    xAxisDomain: axis?.x?.domain,
    contractOnSparseData: axis?.x?.contractOnSparseData,
    firstNonZeroTs,
    lastNonZeroTs,
  });
  const xTickFormatter = (v: number) => formatXAxisTick(v, spanMs, axis?.x?.utc);
  const showAxes = axis !== null && axis?.visible !== false;

  const xTicks = effectiveDomain
    ? [0, 1, 2, 3].map((i) =>
        Math.round(effectiveDomain[0] + (i * (effectiveDomain[1] - effectiveDomain[0])) / 3),
      )
    : undefined;

  return (
    <ChartContainer
      config={config}
      className={cn("!flex-col aspect-auto w-full", chartContainerClassname)}
      style={{ height, width: "100%" }}
    >
      <AreaChart
        data={data}
        onMouseMove={handleActive}
        onMouseLeave={onActiveChange ? () => onActiveChange(null) : undefined}
        margin={
          showAxes
            ? { top: 16, right: 8, bottom: 0, left: 0 }
            : { top: 4, right: 0, bottom: 0, left: 0 }
        }
      >
        <defs>
          {configKeys.map((key) => {
            const fillColor = fillColors?.[key] ?? config[key].color;
            return (
              <linearGradient
                key={`${chartId}-${key}`}
                id={`${chartId}-${key}`}
                x1="0"
                y1="0"
                x2="0"
                y2="1"
              >
                {paleFill ? (
                  <>
                    <stop offset="0%" stopColor={fillColor} stopOpacity={0.85} />
                    <stop offset="90%" stopColor={fillColor} stopOpacity={0.1} />
                  </>
                ) : (
                  <>
                    <stop offset="0%" stopColor={config[key].color} stopOpacity={0.45}>
                      <animate
                        attributeName="stop-opacity"
                        values="0.45;0.3;0.45"
                        dur="6s"
                        repeatCount="indefinite"
                      />
                    </stop>
                    <stop offset="50%" stopColor={config[key].color} stopOpacity={0.2}>
                      <animate
                        attributeName="stop-opacity"
                        values="0.2;0.35;0.2"
                        dur="6s"
                        repeatCount="indefinite"
                      />
                    </stop>
                    <stop offset="85%" stopColor={config[key].color} stopOpacity={0.08}>
                      <animate
                        attributeName="stop-opacity"
                        values="0.08;0.15;0.08"
                        dur="6s"
                        repeatCount="indefinite"
                      />
                    </stop>
                    <stop offset="100%" stopColor={config[key].color} stopOpacity={0.03} />
                  </>
                )}
              </linearGradient>
            );
          })}
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
          dataKey="originalTimestamp"
          type="number"
          domain={effectiveDomain ?? ["dataMin", "dataMax"]}
          allowDataOverflow={Boolean(effectiveDomain)}
          scale="time"
          tickFormatter={xTickFormatter}
          tick={showAxes ? { fill: "hsl(var(--gray-10))", fontSize: 10 } : false}
          tickLine={false}
          axisLine={false}
          ticks={xTicks}
          minTickGap={48}
          hide={!showAxes}
        />
        <YAxis
          width={showAxes ? Y_GUTTER_PX : 0}
          tickLine={false}
          axisLine={false}
          tickFormatter={formatYTick}
          tick={showAxes ? { fill: "hsl(var(--gray-10))", fontSize: 10 } : false}
          ticks={yTicks}
          domain={yDomain}
          hide={!showAxes}
        />
        <ChartTooltip
          allowEscapeViewBox={{ x: false, y: true }}
          wrapperStyle={{ zIndex: 1000, pointerEvents: "none" }}
          cursor={{
            stroke: "hsl(var(--accent-9))",
            strokeWidth: 1,
            strokeDasharray: "3 3",
            strokeOpacity: 0.5,
          }}
          content={({ active, payload }) => {
            if (hideTooltip || !active || !payload?.length) {
              return null;
            }
            const point: unknown = payload[0]?.payload;
            if (!isAreaChartPoint(point)) {
              return null;
            }
            const labelText = formatCompactInterval(
              point.originalTimestamp,
              data,
              showDateInTooltip,
            );
            const rows = configKeys
              .map((key) => ({ key, value: Number(point[key]) || 0 }))
              .sort((a, b) => b.value - a.value);
            return (
              <div
                role="tooltip"
                className="grid w-max max-w-[300px] animate-in items-start gap-1.5 rounded-xl border border-gray-4/50 bg-gray-1/80 px-3 py-2.5 text-xs shadow-2xl backdrop-blur-md duration-150 fade-in-0 zoom-in-95 select-none"
              >
                <div className="font-medium text-[11px] text-accent-11">{labelText}</div>
                <div className="grid gap-1">
                  {rows.map(({ key, value }) => {
                    const itemConfig = config[key];
                    const parts = formatTooltipValue(value);
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
                          {parts.hint && (
                            <span className="text-grayA-9 ml-1 font-normal">{parts.hint}</span>
                          )}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </div>
            );
          }}
        />
        {!paleFill &&
          configKeys.map((key) => (
            <Area
              key={`${key}-glow`}
              dataKey={key}
              type="monotone"
              stroke={config[key].color}
              strokeWidth={3.5}
              strokeOpacity={0.12}
              fill="none"
              isAnimationActive={shouldAnimate}
              animationDuration={500}
              animationEasing="ease-out"
              dot={false}
              activeDot={false}
            />
          ))}
        {configKeys.map((key) => (
          <Area
            key={key}
            dataKey={key}
            type="monotone"
            stroke={config[key].color}
            strokeWidth={1.5}
            fill={`url(#${chartId}-${key})`}
            fillOpacity={1}
            isAnimationActive={shouldAnimate}
            animationDuration={500}
            animationEasing="ease-out"
            dot={false}
            activeDot={(props: { cx?: number; cy?: number }) => {
              const x = props.cx ?? 0;
              const y = props.cy ?? 0;
              return (
                <g>
                  <circle cx={x} cy={y} r={8} fill={config[key].color} opacity={0.12} />
                  <circle cx={x} cy={y} r={5} fill={config[key].color} opacity={0.3} />
                  <circle
                    cx={x}
                    cy={y}
                    r={3}
                    fill={config[key].color}
                    stroke="white"
                    strokeWidth={1.5}
                  />
                </g>
              );
            }}
          />
        ))}
      </AreaChart>
    </ChartContainer>
  );
}

function formatYAxisCompactBytes(v: number): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "";
  }
  const kib = 1024;
  const mib = kib * 1024;
  const gib = mib * 1024;
  if (v >= gib) {
    return `${trim(v / gib)} GiB`;
  }
  if (v >= mib) {
    return `${trim(v / mib)} MiB`;
  }
  if (v >= kib) {
    return `${trim(v / kib)} KiB`;
  }
  return `${Math.round(v)} B`;
}

export function formatYAxisCompactBytesPerSecond(v: number): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "";
  }
  const bytes = formatYAxisCompactBytes(v);
  const spaceIdx = bytes.indexOf(" ");
  if (spaceIdx > 0) {
    return `${bytes.slice(0, spaceIdx)} ${bytes.slice(spaceIdx + 1)}/s`;
  }
  return `${bytes}/s`;
}

function trim(n: number): string {
  if (n >= 10) {
    return `${Math.round(n)}`;
  }
  const rounded = Math.round(n);
  if (Math.abs(n - rounded) < 0.05) {
    return `${rounded}`;
  }
  return n.toFixed(1);
}

const TWO_DAYS_MS = 2 * 24 * 60 * 60 * 1000;
function formatXAxisTick(v: number, spanMs: number, utc?: boolean): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "";
  }
  const tz: Intl.DateTimeFormatOptions = utc ? { timeZone: "UTC" } : {};
  const d = new Date(v);
  if (spanMs >= TWO_DAYS_MS) {
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric", ...tz });
  }
  return d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit", ...tz });
}

function formatCompactInterval(
  startTs: number | undefined,
  data: AreaChartPoint[],
  withDate?: boolean,
): string {
  if (typeof startTs !== "number" || !Number.isFinite(startTs)) {
    return "";
  }
  const idx = data.findIndex((p) => p.originalTimestamp === startTs);
  const nextTs = idx >= 0 ? data[idx + 1]?.originalTimestamp : undefined;
  return formatBucketInterval(startTs, typeof nextTs === "number" ? nextTs : undefined, withDate);
}
