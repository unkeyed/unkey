"use client";

import { type ChartConfig, ChartContainer, ChartTooltip } from "@/components/ui/chart";
import { formatBytesPerSecondParts } from "@/lib/utils/deployment-formatters";
import { cn } from "@unkey/ui/src/lib/utils";
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { LogsChartEmpty } from "./components/logs-chart-empty";
import { LogsChartError } from "./components/logs-chart-error";
import { LogsChartLoading } from "./components/logs-chart-loading";

// Generic timeseries area chart. Drives both the 2-series network chart
// (egress + ingress, overlapping — intentionally NOT stacked; stacking
// makes a big-ingress / tiny-egress spike read as "egress went up" since
// egress sits on top) and the single-series disk chart.
//
// Zero-valued buckets stay in the data so the line renders flat along the
// baseline during idle stretches instead of disappearing. We rely on
// `dot={false}` / `activeDot={false}` to suppress the per-sample markers
// that would otherwise make each zero bucket read as a distinct "point".

export type AreaChartPoint = { originalTimestamp: number } & {
  [k: string]: number | undefined;
};

// When the observed data covers less than this fraction of the caller's
// window, we stop anchoring the x-axis and let it contract to the data's
// extent. 7 days of axis with 20 minutes of data looks broken — better
// to show that 20 minutes clearly and let the axis labels indicate the
// real span. 0.15 = 15% is the gut-check minimum for "this looks full
// enough to anchor".
const SPARSE_COVERAGE_THRESHOLD = 0.15;

// Fixed horizontal gutter on the left of every chart. When the Y-axis is
// shown this is its reserved width; when hidden, the AreaChart's left
// margin consumes the same amount. Keeping this symmetric across charts
// aligns plot-area left edges vertically down the stack — otherwise the
// CPU row (hideYAxis) starts further left than mem/disk/network. 60
// comfortably fits the longest byte tick ("999MB" / "9.9GB") at 10px.
const Y_GUTTER_PX = 60;

// Tooltip row value. `value` is the primary (bold) number, `unit` the
// immediate suffix ("MiB"), and `hint` a muted trailing annotation used
// for secondary context like "(17%)" next to memory values.
type ValueParts = { value: string; unit?: string; hint?: string };

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
  // Hide the Y-axis entirely. Used for metrics where the axis number
  // would be more confusing than informative (e.g. CPU in millicores,
  // where the tooltip already shows a friendlier utilization %).
  hideYAxis?: boolean;
  // Force the X-axis to span this explicit [start, end] range (millis).
  // Used to anchor the axis to the selected window (e.g. "Past day"
  // always reads as 24h of x-axis even when the deployment is only
  // 1h old), so the tick labels change visibly between windows.
  xAxisDomain?: [number, number];
};

export function AreaTimeseriesChart({
  data,
  config,
  height = 160,
  isLoading,
  isError,
  chartContainerClassname,
  showDateInTooltip,
  formatTooltipValue = formatBytesPerSecondParts,
  formatYTick = formatYAxisCompactBytes,
  axisFloor = 1024,
  hideYAxis = false,
  xAxisDomain,
}: Props) {
  if (isError) {
    return <LogsChartError />;
  }
  if (isLoading) {
    // Match the chart's own height so the loading flash doesn't collapse
    // the enclosing panel by ~110px on every window change.
    return <LogsChartLoading height={height} />;
  }
  const configKeys = Object.keys(config);
  const isEmpty = !data.length || data.every((p) => configKeys.every((k) => !(Number(p[k]) > 0)));
  if (isEmpty) {
    return <LogsChartEmpty config={config} height={height} />;
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

  // Use the caller's xAxisDomain only when non-zero data covers enough of
  // it to read as a real trend ("Past week" with 5 days of non-zero data
  // → anchor to the week). When non-zero coverage is too thin ("Past
  // week" with 20 minutes of non-zero data on a new deployment) fall
  // back to the non-zero extent so the handful of real points span the
  // visible width instead of being swallowed by the WITH FILL padding.
  //
  // We specifically look at non-zero samples because ClickHouse's WITH
  // FILL pads every bucket in the window; using total data extent here
  // would always report 100% coverage.
  let firstNonZeroTs: number | undefined;
  let lastNonZeroTs: number | undefined;
  for (const p of data) {
    if (configKeys.some((k) => Number(p[k]) > 0)) {
      firstNonZeroTs ??= p.originalTimestamp;
      lastNonZeroTs = p.originalTimestamp;
    }
  }
  const nonZeroSpanMs =
    firstNonZeroTs !== undefined && lastNonZeroTs !== undefined
      ? Math.max(0, lastNonZeroTs - firstNonZeroTs)
      : 0;
  const windowSpanMs = xAxisDomain ? xAxisDomain[1] - xAxisDomain[0] : 0;
  const coverage = windowSpanMs > 0 ? nonZeroSpanMs / windowSpanMs : 0;
  // Stay anchored in two cases: genuine coverage above the threshold, or
  // a degenerate non-zero span (single bucket, or no non-zero data) that
  // would otherwise contract to a zero-width domain. Keeping the window
  // axis in that case gives the single dot a full frame to sit in
  // instead of disappearing.
  const useAnchoredDomain =
    xAxisDomain && (coverage >= SPARSE_COVERAGE_THRESHOLD || nonZeroSpanMs < 60_000);
  // When we fall back to non-zero extent, we explicitly use the non-zero
  // bounds rather than ["dataMin", "dataMax"] — otherwise recharts
  // widens the axis to include the zero-filled padding rows and the
  // sparse data collapses into the right-hand sliver again.
  const effectiveDomain: [number, number] | undefined = useAnchoredDomain
    ? xAxisDomain
    : firstNonZeroTs !== undefined && lastNonZeroTs !== undefined
      ? [firstNonZeroTs, lastNonZeroTs]
      : undefined;
  const spanMs = useAnchoredDomain ? windowSpanMs : nonZeroSpanMs;
  const xTickFormatter = (v: number) => formatXAxisTick(v, spanMs);

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
        // Reserve the Y-axis gutter via either the YAxis (when shown) or
        // margin.left (when hidden). Same total left offset (`Y_GUTTER_PX`)
        // for every chart so plot-area left edges align vertically across
        // the stack — otherwise the CPU row (hideYAxis) starts further
        // left than memory/disk/network, which reads as different chart
        // sizes.
        margin={{ top: 16, right: 8, bottom: 0, left: hideYAxis ? Y_GUTTER_PX : 0 }}
      >
        <CartesianGrid vertical={false} stroke="hsl(var(--gray-4))" strokeDasharray="3 3" />
        <XAxis
          dataKey="originalTimestamp"
          type="number"
          // Anchor to the caller-provided window range when we have
          // enough data to draw something readable there. Otherwise let
          // the axis contract to the data's extent so a handful of
          // points don't drown in an empty week-wide axis.
          domain={effectiveDomain ?? ["dataMin", "dataMax"]}
          allowDataOverflow={Boolean(effectiveDomain)}
          scale="time"
          tickFormatter={xTickFormatter}
          tick={{ fill: "hsl(var(--gray-10))", fontSize: 10 }}
          tickLine={false}
          axisLine={false}
          ticks={xTicks}
          // minTickGap is only meaningful when we let recharts pick ticks;
          // when we force explicit ticks it has no effect. Keep it as a
          // fallback for the unanchored path.
          minTickGap={48}
        />
        <YAxis
          // Compact single-letter suffix (24M / 1.2K / 600B — no "/s", the
          // rate is implied by context) keeps the axis narrow so the chart
          // doesn't eat the panel. Full units live in the tooltip. When
          // `hideYAxis` is set we still mount YAxis (recharts needs one to
          // compute data extent and range the areas correctly) but strip
          // all visible chrome — no ticks, no labels, zero width.
          width={hideYAxis ? 0 : Y_GUTTER_PX}
          tickLine={false}
          axisLine={false}
          tickFormatter={formatYTick}
          tick={hideYAxis ? false : { fill: "hsl(var(--gray-10))", fontSize: 10 }}
          ticks={hideYAxis ? undefined : yTicks}
          domain={yDomain}
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
            const point = payload[0]?.payload as AreaChartPoint | undefined;
            if (!point) {
              return null;
            }
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
                className="grid items-start gap-1 rounded-lg border border-gray-4 bg-gray-1 px-3 py-2 text-xs shadow-2xl select-none w-max max-w-[240px]"
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
        {configKeys.map((key) => (
          <Area
            key={key}
            dataKey={key}
            type="monotone"
            stroke={config[key].color}
            strokeWidth={1.5}
            fill={config[key].color}
            fillOpacity={0.25}
            isAnimationActive={false}
            dot={false}
            activeDot={false}
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
    return "0";
  }
  const kib = 1024;
  const mib = kib * 1024;
  const gib = mib * 1024;
  if (v >= gib) {
    return `${trim(v / gib)}GB`;
  }
  if (v >= mib) {
    return `${trim(v / mib)}MB`;
  }
  if (v >= kib) {
    return `${trim(v / kib)}KB`;
  }
  return `${Math.round(v)}B`;
}

// Rate variant used by the Network chart. The `/s` suffix on every tick
// keeps the y-axis unit visually consistent with the headline stat
// ("8.40 MiB/s") and prevents the user from pairing a rate tick with the
// separate "total MiB" aggregate in the header.
export function formatYAxisCompactBytesPerSecond(v: number): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "0";
  }
  return `${formatYAxisCompactBytes(v)}/s`;
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
function formatXAxisTick(v: number, spanMs: number): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "";
  }
  const d = new Date(v);
  if (spanMs >= TWO_DAYS_MS) {
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  }
  return d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
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
