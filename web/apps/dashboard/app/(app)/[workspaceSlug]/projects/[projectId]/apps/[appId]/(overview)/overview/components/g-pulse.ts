import type { AreaChartPoint } from "@/components/charts/area-timeseries";

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

function formatWindowLabel(data: AppRpsMetrics | undefined): string {
  return data ? WINDOW_LABEL[data.range] : "";
}

export function buildPulse(data: AppRpsMetrics | undefined): Pulse {
  const windowKey = data?.range ?? "week";
  const series: AreaChartPoint[] = (data?.rps ?? []).map((p) => ({
    originalTimestamp: p.time,
    total: p.requests,
    errors: p.errors,
  }));
  const last = series.at(-1);
  return {
    windowKey,
    windowLabel: formatWindowLabel(data),
    series,
    cumulative: data?.totalRequests ?? 0,
    requestsCurrent: last ? Number(last.total) || 0 : 0,
    errorsCurrent: last ? Number(last.errors) || 0 : 0,
    latestTimestamp: last?.originalTimestamp,
  };
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
