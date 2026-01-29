import type { Log } from "@unkey/clickhouse/src/logs";
import type { RatelimitLog } from "@unkey/clickhouse/src/ratelimits";

export type ResponseBody = {
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
  log: Log | RatelimitLog,
  fieldName: K,
): ResponseBody[K] | null => {
  if (!log?.response_body) {
    console.error("Invalid log or missing response_body");
    return null;
  }

  try {
    const parsedBody = JSON.parse(log.response_body) as ResponseBody;

    return parsedBody[fieldName];
  } catch {
    return null;
  }
};

export const getRequestHeader = (log: Log | RatelimitLog, headerName: string): string | null => {
  if (!headerName.trim()) {
    console.error("Invalid header name provided");
    return null;
  }

  if (!Array.isArray(log.request_headers)) {
    console.error("request_headers is not an array");
    return null;
  }

  const lowerHeaderName = headerName.toLowerCase();
  const header = log.request_headers.find((h) => h.toLowerCase().startsWith(`${lowerHeaderName}:`));

  if (!header) {
    console.warn(`Header "${headerName}" not found in request headers`);
    return null;
  }

  const [, value] = header.split(":", 2);
  return value ? value.trim() : null;
};

export const safeParseJson = (jsonString?: string | null) => {
  if (!jsonString) {
    return null;
  }

  try {
    return JSON.parse(jsonString);
  } catch {
    console.error("Cannot parse JSON:", jsonString);
    return "Invalid JSON format";
  }
};
