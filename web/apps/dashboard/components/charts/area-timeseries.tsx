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

// Recharts hands the tooltip render callback an unknown `payload` shape; this
// guard narrows it before we touch `originalTimestamp` so a future schema
// drift surfaces as "no tooltip" rather than a runtime read of undefined.
function isAreaChartPoint(value: unknown): value is AreaChartPoint {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as { originalTimestamp?: unknown }).originalTimestamp === "number"
  );
}

// Recharts v3 reports the hovered tick as `activeTooltipIndex`, not `activePayload`,
// so look the point up by index.
function activePointFromState(state: unknown, data: AreaChartPoint[]): AreaChartPoint | null {
  if (typeof state !== "object" || state === null) {
    return null;
  }
  const s = state as { activeTooltipIndex?: unknown; activeIndex?: unknown };
  const raw = s.activeTooltipIndex ?? s.activeIndex;
  const idx =
    typeof raw === "number" ? raw : typeof raw === "string" ? Number.parseInt(raw, 10) : Number.NaN;
  if (!Number.isInteger(idx) || idx < 0 || idx >= data.length) {
    return null;
  }
  const candidate = data[idx];
  return isAreaChartPoint(candidate) ? candidate : null;
}

// Fixed horizontal gutter on the left of every chart. When the Y-axis is
// shown this is its reserved width; when hidden, the AreaChart's left
// margin consumes the same amount. Keeping this symmetric across charts
// aligns plot-area left edges vertically down the stack — otherwise the
// CPU row (hideYAxis) starts further left than mem/disk/network. 60
// comfortably fits the longest byte tick ("999MB" / "9.9GB") at 10px.
const Y_GUTTER_PX = 36;

// Tooltip row value. `value` is the primary (bold) number, `unit` the
// immediate suffix ("MiB"), and `hint` a muted trailing annotation used
// for secondary context like "(17%)" next to memory values.
export type ValueParts = { value: string; unit?: string; hint?: string };

type Props = {
  data: AreaChartPoint[];
  config: ChartConfig;
  height?: number;
  isLoading?: boolean;
  isError?: boolean;
  chartContainerClassname?: string;
  showDateInTooltip?: boolean;
  // Renders a tooltip row value, e.g. `24.01` + ` MiB/s`. Defaults to
  // bytes/s for network rate charts.
  formatTooltipValue?: (value: number) => ValueParts;
  // Renders a compact Y-axis tick, e.g. `24M`. Defaults to the bytes/s
  // single-letter scale.
  formatYTick?: (value: number) => string;
  // Minimum top-of-axis when data is tiny / idle, so baselines aren't
  // compressed to nothing. Defaults to 1 KiB for the network rate chart.
  axisFloor?: number;
  // Force the X-axis to span this explicit [start, end] range (millis).
  // Used to anchor the axis to the selected window (e.g. "Past day"
  // always reads as 24h of x-axis even when the deployment is only
  // 1h old), so the tick labels change visibly between windows.
  xAxisDomain?: [number, number];
  // Opt in to contracting the x-axis to the non-zero data extent when that
  // data sparsely covers `xAxisDomain` (see [resolveXAxisDomain]). Off by
  // default: an explicit window is honored as-is, so a header that commits to
  // it (e.g. "requests this week") never disagrees with the axis. The
  // deployment resource panels turn this on, where a brand-new deployment with
  // minutes of data on a multi-day axis reads as broken.
  contractOnSparseData?: boolean;
  hideAxes?: boolean;
  paleFill?: boolean;
  fillColors?: Record<string, string>;
  xAxisUTC?: boolean;
  onActiveChange?: (point: AreaChartPoint | null) => void;
  hideTooltip?: boolean;
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
  formatYTick = formatYAxisCompactBytes,
  axisFloor = 1024,
  xAxisDomain,
  contractOnSparseData,
  hideAxes,
  paleFill,
  fillColors,
  xAxisUTC,
  onActiveChange,
  hideTooltip,
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
  const isEmpty = !data.length || data.every((p) => configKeys.every((k) => !(Number(p[k]) > 0)));
  if (isEmpty) {
    return (
      <ChartEmpty variant="wave" color={sectionColor} height={height} message="No activity yet" />
    );
  }

  // Evenly-spaced Y-axis ticks at 0 / ⅓ / ⅔ / top. We compute them ourselves
  // because recharts' auto-picker chooses "nice round" values (300/600/1.0K)
  // that aren't evenly distributed on the pixel axis. Explicit ticks keep
  // the dashed grid lines visually even regardless of data range.
  const dataMax = data.reduce(
    (m, p) => configKeys.reduce((mm, k) => Math.max(mm, Number(p[k]) || 0), m),
    0,
  );
  const top = niceCeil(Math.max(dataMax, axisFloor));
  const yTicks = [0, top / 3, (2 * top) / 3, top];
  const yDomain: [number, number] = [0, top];

  // Bounds of the non-zero samples (not the raw data extent). resolveXAxisDomain
  // needs these to decide whether the data sparsely covers the window; we scan
  // for non-zero because ClickHouse's WITH FILL pads every bucket, so the raw
  // extent would always look full.
  let firstNonZeroTs: number | undefined;
  let lastNonZeroTs: number | undefined;
  for (const p of data) {
    if (configKeys.some((k) => Number(p[k]) > 0)) {
      firstNonZeroTs ??= p.originalTimestamp;
      lastNonZeroTs = p.originalTimestamp;
    }
  }
  const { effectiveDomain, spanMs } = resolveXAxisDomain({
    xAxisDomain,
    contractOnSparseData,
    firstNonZeroTs,
    lastNonZeroTs,
  });
  const xTickFormatter = (v: number) => formatXAxisTick(v, spanMs, xAxisUTC);

  // Explicit ticks at 0% / 33% / 66% / 100% of the anchored domain so the
  // user always sees the window's start and end labeled. Without this,
  // recharts' auto-picker can place all ticks inside the data extent,
  // leaving the leading empty portion of the axis unlabeled — which
  // reads as "chart starts at 5:17" when the axis actually spans back
  // to 5:00.
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
          hideAxes
            ? { top: 4, right: 0, bottom: 0, left: 0 }
            : { top: 16, right: 8, bottom: 0, left: 0 }
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
        {!hideAxes && (
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
          tick={hideAxes ? false : { fill: "hsl(var(--gray-10))", fontSize: 10 }}
          tickLine={false}
          axisLine={false}
          ticks={xTicks}
          minTickGap={48}
          hide={hideAxes}
        />
        {!hideAxes && (
          <YAxis
            width={Y_GUTTER_PX}
            tickLine={false}
            axisLine={false}
            tickFormatter={formatYTick}
            tick={{ fill: "hsl(var(--gray-10))", fontSize: 10 }}
            ticks={yTicks}
            domain={yDomain}
          />
        )}
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
            const candidate = payload[0]?.payload;
            if (!isAreaChartPoint(candidate)) {
              return null;
            }
            const point = candidate;
            const labelText = formatCompactInterval(
              point.originalTimestamp,
              data,
              showDateInTooltip,
            );
            // Sort rows by value descending so whichever series is dominating
            // shows first — matches where the user's eye goes on a spike.
            // Ties preserve declared order.
            const rows = configKeys
              .map((key) => ({ key, value: Number(point[key]) || 0 }))
              .sort((a, b) => b.value - a.value);
            return (
              <div
                role="tooltip"
                className="grid items-start gap-1.5 rounded-xl border border-gray-4/50 bg-gray-1/80 backdrop-blur-md px-3 py-2.5 text-xs shadow-2xl select-none w-max max-w-[240px] animate-in fade-in-0 zoom-in-95 duration-150"
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

// Y-axis ticks use two-letter byte-scale suffixes (B / KB / MB / GB).
// Short enough to keep the axis narrow, long enough to read as units at
// a glance — a single "M" looks like a variable name, "MB" reads as size.
// Full IEC units ("MiB/s") live in the tooltip.
export function formatYAxisCompactBytes(v: number): string {
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

// Rate variant used by the Network chart. The `/s` suffix on every tick
// keeps the y-axis unit visually consistent with the headline stat
// ("8.40 MiB/s") and prevents the user from pairing a rate tick with the
// separate "total MiB" aggregate in the header.
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

// One decimal only when it carries information: "1.5K" yes, "7.0M" no.
// niceCeil picks ticks that divide cleanly by the unit, so in practice
// this nearly always renders an integer.
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

// niceCeil rounds `max` up to a multiple of 3 × unit (B/K/M/G) so the
// intermediate evenly-spaced ticks (top/3, 2*top/3) land on readable
// values — e.g. max=5000 → top=6144 giving 0 / 2K / 4K / 6K instead of
// 0 / 341 / 682 / 1024.
function niceCeil(max: number): number {
  const kib = 1024;
  const mib = kib * 1024;
  const gib = mib * 1024;
  const unit = max >= gib ? gib : max >= mib ? mib : max >= kib ? kib : 1;
  const step = 3 * unit;
  return Math.max(step, Math.ceil(max / step) * step);
}

// X-axis tick format: "HH:MM" for sub-2-day spans, "MMM d" once we're
// looking at a multi-day window. Inferred from the data span so the chart
// doesn't need the caller to pass the selected window explicitly.
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
