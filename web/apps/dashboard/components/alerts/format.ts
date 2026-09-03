import type { AlertMetric, AlertSeriesMetric } from "./types";

export const alertMetricOptions: ReadonlyArray<{ value: AlertMetric; label: string }> = [
  { value: "error_5xx", label: "5xx errors" },
  { value: "error_4xx", label: "4xx errors" },
  { value: "requests", label: "Requests" },
  { value: "requests_drop", label: "Traffic drop" },
  { value: "egress_bytes", label: "Egress" },
  { value: "cpu_seconds", label: "CPU" },
  { value: "memory_utilization", label: "Memory" },
  { value: "oom_killed", label: "Out of memory" },
  { value: "crash_loop", label: "Crash loop" },
];

export const alertSeriesMetricOptions: ReadonlyArray<{
  value: AlertSeriesMetric;
  label: string;
}> = [
  { value: "error_5xx", label: "5xx errors" },
  { value: "error_4xx", label: "4xx errors" },
  { value: "requests", label: "Requests" },
  { value: "egress_bytes", label: "Egress" },
  { value: "cpu_seconds", label: "CPU" },
  { value: "memory_utilization", label: "Memory" },
  { value: "health", label: "Health" },
];

const quantityFormatter = new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 });
const compactFormatter = new Intl.NumberFormat("en-US", {
  notation: "compact",
  maximumFractionDigits: 1,
});
const baselineMultipleFormatter = new Intl.NumberFormat("en-US", {
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
});

export function isAlertMetric(value: string): value is AlertMetric {
  return alertMetricOptions.some((option) => option.value === value);
}

export function seriesMetricForAlert(metric: AlertMetric): AlertSeriesMetric {
  switch (metric) {
    case "requests_drop":
      return "requests";
    case "oom_killed":
    case "crash_loop":
      return "health";
    case "error_5xx":
    case "error_4xx":
    case "requests":
    case "egress_bytes":
    case "cpu_seconds":
    case "memory_utilization":
      return metric;
    default:
      return metric satisfies never;
  }
}

export function isAlertSeriesMetric(value: string): value is AlertSeriesMetric {
  return alertSeriesMetricOptions.some((option) => option.value === value);
}

export function alertSeriesMetricLabel(metric: AlertSeriesMetric): string {
  return alertSeriesMetricOptions.find((option) => option.value === metric)?.label ?? metric;
}

export function formatAlertSeriesValue(metric: AlertSeriesMetric, value: number): string {
  return formatAlertValue(metric === "health" ? "oom_killed" : metric, value);
}

export function formatAlertSeriesAxisValue(metric: AlertSeriesMetric, value: number): string {
  return formatAlertAxisValue(metric === "health" ? "oom_killed" : metric, value);
}

export function alertMetricLabel(metric: AlertMetric): string {
  switch (metric) {
    case "error_5xx":
      return "5xx errors";
    case "error_4xx":
      return "4xx errors";
    case "requests":
      return "Requests";
    case "requests_drop":
      return "Traffic drop";
    case "egress_bytes":
      return "Egress";
    case "cpu_seconds":
      return "CPU";
    case "memory_utilization":
      return "Memory";
    case "oom_killed":
      return "Out of memory";
    case "crash_loop":
      return "Crash loop";
    default:
      return metric satisfies never;
  }
}

export function formatAlertValue(metric: AlertMetric, value: number): string {
  switch (metric) {
    case "error_5xx":
    case "error_4xx":
      return `${formatDecimal(value * 100, 1)}%`;
    case "egress_bytes":
      return formatBytes(value);
    case "cpu_seconds":
      return `${formatDecimal(value, 2)} s`;
    case "memory_utilization":
      return `${formatDecimal(value * 100, 1)}%`;
    case "requests":
    case "requests_drop":
    case "oom_killed":
    case "crash_loop":
      return quantityFormatter.format(value);
    default:
      return metric satisfies never;
  }
}

export function formatAlertAxisValue(metric: AlertMetric, value: number): string {
  switch (metric) {
    case "error_5xx":
    case "error_4xx":
      return `${formatDecimal(value * 100, 1)}%`;
    case "egress_bytes":
      return formatBytes(value);
    case "memory_utilization":
      return `${formatDecimal(value * 100, 0)}%`;
    case "cpu_seconds":
      return `${compactFormatter.format(value)} s`;
    case "requests":
    case "requests_drop":
    case "oom_killed":
    case "crash_loop":
      return compactFormatter.format(value);
    default:
      return metric satisfies never;
  }
}

export function formatBaselineMultiple(observed: number, mean: number): string {
  if (mean <= 0) {
    return "no prior traffic";
  }
  const multiple = observed / mean;
  if (multiple < 1) {
    return "below baseline";
  }
  const formatted =
    multiple >= 10
      ? quantityFormatter.format(multiple)
      : baselineMultipleFormatter.format(multiple);
  return `${formatted}× baseline`;
}

export function hasFixedAlertThreshold(metric: AlertMetric): boolean {
  return metric === "memory_utilization" || metric === "oom_killed" || metric === "crash_loop";
}

export function isErrorRateMetric(metric: AlertMetric): boolean {
  return metric === "error_5xx" || metric === "error_4xx";
}

export function formatRequestsDropChange(observed: number, recentMedian: number): string {
  if (recentMedian <= 0) {
    return "No recent traffic";
  }
  const changePercent = (observed / recentMedian - 1) * 100;
  const sign = changePercent < 0 ? "−" : "+";
  return `${sign}${formatDecimal(Math.abs(changePercent), 0)}%`;
}

export function formatAlertDistance(metric: AlertMetric, observed: number, mean: number): string {
  switch (metric) {
    case "memory_utilization":
      return `avg ${formatAlertValue(metric, observed)} · limit 90%`;
    case "oom_killed":
    case "crash_loop":
      return `${formatAlertValue(metric, observed)} events · limit 1`;
    case "requests_drop":
      return formatRequestsDropChange(observed, mean);
    case "error_5xx":
    case "error_4xx":
      return mean <= 0 ? "no prior errors" : formatBaselineMultiple(observed, mean);
    case "requests":
    case "egress_bytes":
    case "cpu_seconds":
      return formatBaselineMultiple(observed, mean);
    default:
      return metric satisfies never;
  }
}

function formatBytes(bytes: number): string {
  const units = ["B", "KB", "MB", "GB", "TB"] as const;
  let value = Math.max(bytes, 0);
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  return `${formatDecimal(value, value >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
}

function formatDecimal(value: number, maximumFractionDigits: number): string {
  return new Intl.NumberFormat("en-US", { maximumFractionDigits }).format(value);
}
