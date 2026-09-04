import type { AppMetricsRange, AppMetricsSeriesPoint, AppMetricsWindow } from "@unkey/clickhouse";

export type SeriesPoint = { x: number } & Record<string, number>;

export type SeriesKey = { key: string; label: string; color: string };

export const WINDOW_LABELS: Record<AppMetricsWindow, { short: string; long: string }> = {
  "1h": { short: "1h", long: "Last hour" },
  "6h": { short: "6h", long: "Last 6 hours" },
  "1d": { short: "24h", long: "Last 24 hours" },
  "1w": { short: "7d", long: "Last 7 days" },
  "30d": { short: "30d", long: "Last 30 days" },
};

export const STATUS_SERIES: SeriesKey[] = [
  { key: "2xx", label: "2xx", color: "hsl(var(--success-9))" },
  { key: "3xx", label: "3xx", color: "hsl(var(--info-9))" },
  { key: "4xx", label: "4xx", color: "hsl(var(--warning-9))" },
  { key: "5xx", label: "5xx", color: "hsl(var(--error-9))" },
];

export const PERCENTILE_SERIES: SeriesKey[] = [
  { key: "p50", label: "p50", color: "hsl(var(--bronze-8))" },
  { key: "p95", label: "p95", color: "hsl(var(--bronze-10))" },
  { key: "p99", label: "p99", color: "hsl(var(--bronze-12))" },
];

export const METRIC_COLORS = {
  cpu: "hsl(var(--feature-9))",
  memory: "hsl(var(--info-9))",
  disk: "hsl(var(--success-9))",
  egress: "hsl(var(--info-9))",
  ingress: "hsl(var(--warning-9))",
  requests: "hsl(var(--success-9))",
  errors: "hsl(var(--error-9))",
  latency: "hsl(var(--bronze-9))",
};

const SPLIT_PALETTE = [
  "hsl(var(--info-9))",
  "hsl(var(--feature-9))",
  "hsl(var(--warning-9))",
  "hsl(var(--success-9))",
  "hsl(var(--bronze-9))",
  "hsl(var(--error-9))",
  "hsl(var(--accent-9))",
];

export function paletteColor(index: number): string {
  return SPLIT_PALETTE[index % SPLIT_PALETTE.length];
}

export function bucketsOf(range: AppMetricsRange): number[] {
  const step = range.bucketSeconds * 1000;
  const out: number[] = [];
  for (let x = range.startMs; x < range.endMs; x += step) {
    out.push(x);
  }
  return out;
}

// Distinct series names in order of first appearance, so a deployment that
// went live earlier lists before its successor.
export function seriesKeysOf(points: AppMetricsSeriesPoint[]): string[] {
  const seen = new Set<string>();
  const keys: string[] = [];
  for (const p of points) {
    if (!seen.has(p.series)) {
      seen.add(p.series);
      keys.push(p.series);
    }
  }
  return keys;
}

// Pivots {x, series, y} rows into one dense row per bucket with a column per
// series, zero-filled. Optional `scale` converts the stored unit (e.g. cpu
// microseconds per bucket) into the displayed one.
export function pivot(
  points: AppMetricsSeriesPoint[],
  range: AppMetricsRange,
  keys: string[],
  scale: (y: number) => number = (y) => y,
): SeriesPoint[] {
  const byX = new Map<number, SeriesPoint>();
  for (const x of bucketsOf(range)) {
    const row: SeriesPoint = { x };
    for (const k of keys) {
      row[k] = 0;
    }
    byX.set(x, row);
  }
  for (const p of points) {
    const row = byX.get(p.x);
    if (row && keys.includes(p.series)) {
      row[p.series] = scale(p.y);
    }
  }
  return [...byX.values()];
}

export function sumSeries(points: SeriesPoint[], key: string): number {
  let total = 0;
  for (const p of points) {
    total += p[key] ?? 0;
  }
  return total;
}

export function maxSeries(points: SeriesPoint[], key: string): number {
  let max = 0;
  for (const p of points) {
    max = Math.max(max, p[key] ?? 0);
  }
  return max;
}

export function meanSeries(points: SeriesPoint[], key: string): number {
  const nonZero = points.filter((p) => (p[key] ?? 0) > 0);
  if (nonZero.length === 0) {
    return 0;
  }
  return sumSeries(nonZero, key) / nonZero.length;
}

// Adds a `total` column across the given keys.
export function withTotal(points: SeriesPoint[], keys: string[]): SeriesPoint[] {
  return points.map((p) => {
    let total = 0;
    for (const k of keys) {
      total += p[k] ?? 0;
    }
    return { ...p, total };
  });
}

// ─── Formatting ──────────────────────────────────────────────────────

const KIB = 1024;
const MIB = KIB * 1024;
const GIB = MIB * 1024;
const TIB = GIB * 1024;

export function formatBytes(bytes: number, digits = 1): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "0 B";
  }
  if (bytes >= TIB) {
    return `${(bytes / TIB).toFixed(2)} TiB`;
  }
  if (bytes >= GIB) {
    return `${(bytes / GIB).toFixed(digits)} GiB`;
  }
  if (bytes >= MIB) {
    return `${(bytes / MIB).toFixed(digits)} MiB`;
  }
  if (bytes >= KIB) {
    return `${(bytes / KIB).toFixed(0)} KiB`;
  }
  return `${Math.round(bytes)} B`;
}

export function formatBytesRate(bytesPerSecond: number): string {
  if (!Number.isFinite(bytesPerSecond) || bytesPerSecond <= 0) {
    return "0 B/s";
  }
  return `${formatBytes(bytesPerSecond)}/s`;
}

export function formatVcpu(millicores: number): string {
  if (!Number.isFinite(millicores) || millicores <= 0) {
    return "0 vCPU";
  }
  if (millicores < 100) {
    return `${millicores.toFixed(1)} mCPU`;
  }
  return `${(millicores / 1000).toFixed(2)} vCPU`;
}

export function formatCpuTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return "0s";
  }
  if (seconds < 60) {
    return `${Math.round(seconds)}s`;
  }
  if (seconds < 3600) {
    return `${(seconds / 60).toFixed(1)}m`;
  }
  return `${(seconds / 3600).toFixed(1)}h`;
}

export function formatCount(n: number): string {
  if (!Number.isFinite(n) || n <= 0) {
    return "0";
  }
  if (n >= 1_000_000_000) {
    return `${(n / 1_000_000_000).toFixed(1)}B`;
  }
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(1)}M`;
  }
  if (n >= 10_000) {
    return `${Math.round(n / 1000)}K`;
  }
  if (n >= 1000) {
    return `${(n / 1000).toFixed(1)}K`;
  }
  return `${Math.round(n)}`;
}

export function formatRequestRate(perSecond: number): string {
  if (!Number.isFinite(perSecond) || perSecond <= 0) {
    return "0 req/s";
  }
  if (perSecond < 10) {
    return `${perSecond.toFixed(2)} req/s`;
  }
  return `${formatCount(perSecond)} req/s`;
}

export function formatMs(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) {
    return "0 ms";
  }
  if (ms < 1000) {
    return `${ms < 10 ? ms.toFixed(1) : Math.round(ms)} ms`;
  }
  return `${(ms / 1000).toFixed(2)} s`;
}

export function formatPercent(p: number): string {
  if (!Number.isFinite(p) || p <= 0) {
    return "0%";
  }
  if (p < 0.1) {
    return `${p.toFixed(2)}%`;
  }
  if (p < 10) {
    return `${p.toFixed(1)}%`;
  }
  return `${Math.round(p)}%`;
}

export function shortSha(sha: string | null | undefined): string {
  return sha ? sha.slice(0, 7) : "";
}

export function formatTickTime(x: number, spanMs: number): string {
  const d = new Date(x);
  if (spanMs >= 2 * 24 * 60 * 60 * 1000) {
    return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  }
  return d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
}

export function formatBucketLabel(x: number, bucketSeconds: number): string {
  const start = new Date(x);
  const end = new Date(x + bucketSeconds * 1000);
  const date = start.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  const time = (d: Date) => d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
  if (bucketSeconds >= 24 * 3600) {
    return date;
  }
  return `${date}, ${time(start)} – ${time(end)}`;
}

// Axis ticks drop the unit spacing so they fit the gutter; the tooltip keeps
// the full form.
export function tickBytes(v: number): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "";
  }
  if (v >= GIB) {
    return `${(v / GIB).toFixed(1)}G`;
  }
  if (v >= MIB) {
    return `${(v / MIB).toFixed(0)}M`;
  }
  if (v >= KIB) {
    return `${(v / KIB).toFixed(0)}K`;
  }
  return `${Math.round(v)}B`;
}

export function tickBytesRate(v: number): string {
  const t = tickBytes(v);
  return t ? `${t}/s` : "";
}

export function tickVcpu(v: number): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "";
  }
  return v < 100 ? `${v.toFixed(0)}m` : `${(v / 1000).toFixed(2)}`;
}

export function tickCount(v: number): string {
  return v > 0 ? formatCount(v) : "";
}

export function tickMs(v: number): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "";
  }
  return v < 1000 ? `${Math.round(v)}ms` : `${(v / 1000).toFixed(1)}s`;
}

export function tickPercent(v: number): string {
  return v > 0 ? formatPercent(v) : "";
}
