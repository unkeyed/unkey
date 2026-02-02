import type { RuntimeLog } from "./types";

export function safeParseAttributes(log: RuntimeLog): Record<string, unknown> | null {
  if (!log.attributes) return null;
  if (typeof log.attributes === "object") return log.attributes;
  return null;
}

export function formatTimestamp(timestamp: number): string {
  return new Date(timestamp).toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function getSeverityColorClass(severity: string): string {
  const colors = {
    ERROR: "text-error-11 bg-error-2",
    WARN: "text-warning-11 bg-warning-2",
    INFO: "text-info-11 bg-info-2",
    DEBUG: "text-grayA-9 bg-grayA-2",
  };
  return colors[severity as keyof typeof colors] || colors.DEBUG;
}
