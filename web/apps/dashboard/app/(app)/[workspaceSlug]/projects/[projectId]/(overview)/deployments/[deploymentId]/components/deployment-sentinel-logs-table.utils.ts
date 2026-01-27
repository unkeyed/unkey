import type { SentinelResponse } from "@unkey/clickhouse/src/sentinel";

type ResponseBody = {
  keyId: string;
  valid: boolean;
  meta: Record<string, unknown>;
  enabled: boolean;
  permissions: string[];
  code:
  | "VALID"
  | "RATE_LIMITED"
  | "EXPIRED"
  | "USAGE_EXCEEDED"
  | "DISABLED"
  | "FORBIDDEN"
  | "INSUFFICIENT_PERMISSIONS";
};


export const extractResponseField = <K extends keyof ResponseBody>(
  log: SentinelResponse,
  fieldName: K,
): ResponseBody[K] | null => {
  if (!log?.response_body) {
    return null;
  }

  try {
    const parsedBody = JSON.parse(log.response_body) as ResponseBody;
    return parsedBody[fieldName];
  } catch {
    return null;
  }
};

/**
 * Format latency in milliseconds
 */
export const formatLatency = (latency: number): string => {
  return `${latency}ms`;
};

/**
 * Get CSS classes for latency badge based on performance thresholds
 * - >500ms: error (red)
 * - >200ms: warning (yellow)
 * - Otherwise: default (gray)
 */
export const getLatencyStyle = (latency: number): string => {
  if (latency > 500) {
    return "bg-error-4 text-error-11";
  }
  if (latency > 200) {
    return "bg-warning-4 text-warning-11";
  }
  return "bg-grayA-3 text-grayA-11 group-hover:bg-grayA-5";
};
