import type { AlertMetric } from "./types";

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

const quantityFormatter = new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 });
const compactFormatter = new Intl.NumberFormat("en-US", {
  notation: "compact",
  maximumFractionDigits: 1,
});

export function isAlertMetric(value: string): value is AlertMetric {
  return alertMetricOptions.some((option) => option.value === value);
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
    case "egress_bytes":
      return formatBytes(value);
    case "cpu_seconds":
      return `${formatDecimal(value, 2)} s`;
    case "memory_utilization":
      return `${formatDecimal(value * 100, 1)}%`;
    case "error_5xx":
    case "error_4xx":
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
    case "egress_bytes":
      return formatBytes(value);
    case "memory_utilization":
      return `${formatDecimal(value * 100, 0)}%`;
    case "cpu_seconds":
      return `${compactFormatter.format(value)} s`;
    case "error_5xx":
    case "error_4xx":
    case "requests":
    case "requests_drop":
    case "oom_killed":
    case "crash_loop":
      return compactFormatter.format(value);
    default:
      return metric satisfies never;
  }
}

export function formatSigma(observed: number, mean: number, stddev: number): string {
  if (stddev <= 0) {
    return "No variance";
  }
  const sigma = (observed - mean) / stddev;
  return `${sigma >= 0 ? "+" : ""}${sigma.toFixed(1)}σ`;
}

export function hasFixedAlertThreshold(metric: AlertMetric): boolean {
  return metric === "memory_utilization" || metric === "oom_killed" || metric === "crash_loop";
}

export function formatAlertDistance(
  metric: AlertMetric,
  observed: number,
  mean: number,
  stddev: number,
): string {
  switch (metric) {
    case "memory_utilization":
      return `${formatAlertValue(metric, observed)} · limit 90%`;
    case "oom_killed":
    case "crash_loop":
      return `${formatAlertValue(metric, observed)} events · limit 1`;
    default:
      return formatSigma(observed, mean, stddev);
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
