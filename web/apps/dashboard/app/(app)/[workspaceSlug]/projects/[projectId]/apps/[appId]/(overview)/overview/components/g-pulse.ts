import type { AreaChartPoint } from "../../deployments/[deploymentId]/network/unkey-flow/components/overlay/node-details-panel/components/chart/area-timeseries-chart";

export type WindowKey = "hour" | "day" | "week";

export type Pulse = {
  windowKey: WindowKey;
  windowLabel: string;
  series: AreaChartPoint[];
  cumulative: number;
  rpsCurrent: number;
  errorsCurrent: number;
  latestTimestamp: number | undefined;
};

const BUCKET_SECONDS = 15 * 60;

const WINDOW_LABEL: Record<WindowKey, string> = {
  hour: "this hour",
  day: "today",
  week: "this week",
};

type AppRpsMetrics = {
  range: WindowKey;
  totalRequests: number;
  rps: { time: number; requests: number; errors: number }[];
};

export function buildPulse(data: AppRpsMetrics | undefined): Pulse {
  const windowKey = data?.range ?? "week";
  const series: AreaChartPoint[] = (data?.rps ?? []).map((p) => ({
    originalTimestamp: p.time,
    total: p.requests / BUCKET_SECONDS,
    errors: p.errors / BUCKET_SECONDS,
  }));
  const last = series.at(-1);
  return {
    windowKey,
    windowLabel: WINDOW_LABEL[windowKey],
    series,
    cumulative: data?.totalRequests ?? 0,
    rpsCurrent: last ? Number(last.total) || 0 : 0,
    errorsCurrent: last ? Number(last.errors) || 0 : 0,
    latestTimestamp: last?.originalTimestamp,
  };
}

export function formatCount(n: number): string {
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(2).replace(/\.?0+$/, "")}M`;
  }
  if (n >= 1_000) {
    return `${(n / 1_000).toFixed(1).replace(/\.0$/, "")}K`;
  }
  return `${n}`;
}

export function formatRps(v: number): string {
  return `${v >= 10 ? Math.round(v) : Math.round(v * 10) / 10}/s`;
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

export function formatStamp(
  ts: number | undefined,
  windowKey: WindowKey,
  hovering: boolean,
): string {
  if (ts === undefined) {
    return "";
  }
  const d = new Date(ts);
  const time = `${String(d.getUTCHours()).padStart(2, "0")}:${String(d.getUTCMinutes()).padStart(2, "0")} UTC`;
  if (windowKey !== "week") {
    return time;
  }
  const date = `${d.getUTCDate()} ${MONTHS[d.getUTCMonth()]}`;
  return hovering ? `${date}, ${time}` : date;
}
