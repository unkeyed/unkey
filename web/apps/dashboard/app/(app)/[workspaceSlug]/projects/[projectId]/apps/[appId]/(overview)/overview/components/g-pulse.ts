import type { AreaChartPoint } from "../../deployments/[deploymentId]/network/unkey-flow/components/overlay/node-details-panel/components/chart/area-timeseries-chart";

export type WindowKey = "hour" | "day" | "week";

export type Pulse = {
  windowKey: WindowKey;
  windowLabel: string;
  series: AreaChartPoint[];
  cumulative: number;
  requestsCurrent: number;
  errorsCurrent: number;
  latestTimestamp: number | undefined;
};

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

const MAX_CHART_POINTS = 168;

function downsample(points: AppRpsMetrics["rps"]): AppRpsMetrics["rps"] {
  if (points.length <= MAX_CHART_POINTS) {
    return points;
  }
  const groupSize = Math.ceil(points.length / MAX_CHART_POINTS);
  const grouped: AppRpsMetrics["rps"] = [];
  for (let i = 0; i < points.length; i += groupSize) {
    const group = points.slice(i, i + groupSize);
    grouped.push({
      time: group[0].time,
      requests: group.reduce((sum, p) => sum + p.requests, 0),
      errors: group.reduce((sum, p) => sum + p.errors, 0),
    });
  }
  return grouped;
}

export function buildPulse(data: AppRpsMetrics | undefined): Pulse {
  const windowKey = data?.range ?? "week";
  const series: AreaChartPoint[] = downsample(data?.rps ?? []).map((p) => ({
    originalTimestamp: p.time,
    total: p.requests,
    errors: p.errors,
  }));
  const last = series.at(-1);
  return {
    windowKey,
    windowLabel: WINDOW_LABEL[windowKey],
    series,
    cumulative: data?.totalRequests ?? 0,
    requestsCurrent: last ? Number(last.total) || 0 : 0,
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

export function formatBucketCount(v: number): string {
  if (!Number.isFinite(v) || v <= 0) {
    return "0";
  }
  return v >= 1000 ? `${(v / 1000).toFixed(1).replace(/\.0$/, "")}K` : `${Math.round(v)}`;
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
