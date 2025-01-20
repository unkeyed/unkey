import type { Log } from "@unkey/clickhouse/src/logs";

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

class ResponseBodyParseError extends Error {
  constructor(
    message: string,
    public readonly context?: unknown,
  ) {
    super(message);
    this.name = "ResponseBodyParseError";
  }
}

export const extractResponseField = <K extends keyof ResponseBody>(
  log: Log,
  fieldName: K,
): ResponseBody[K] | null => {
  if (!log?.response_body) {
    console.error("Invalid log or missing response_body");
    return null;
  }

  try {
    const parsedBody = JSON.parse(log.response_body) as ResponseBody;

    if (typeof parsedBody !== "object" || parsedBody === null) {
      throw new ResponseBodyParseError("Parsed response body is not an object", parsedBody);
    }

    if (!(fieldName in parsedBody)) {
      throw new ResponseBodyParseError(`Field "${String(fieldName)}" not found in response body`, {
        availableFields: Object.keys(parsedBody),
      });
    }

    return parsedBody[fieldName];
  } catch (error) {
    if (error instanceof ResponseBodyParseError) {
      console.error(`Error parsing response body or accessing field: ${error.message}`, {
        context: error.context,
        fieldName,
        logId: log.request_id,
      });
    } else {
      console.error("An unknown error occurred while parsing response body");
    }
    return null;
  }
};

export const getRequestHeader = (log: Log, headerName: string): string | null => {
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

export const HOUR_IN_MS = 60 * 60 * 1000;
const DAY_IN_MS = 24 * HOUR_IN_MS;
const WEEK_IN_MS = 7 * DAY_IN_MS;

export type TimeseriesGranularity = "perMinute" | "perHour" | "perDay";
type TimeseriesConfig = {
  granularity: TimeseriesGranularity;
  startTime: number;
  endTime: number;
};

export const getTimeseriesGranularity = (
  startTime?: number | null,
  endTime?: number | null,
): TimeseriesConfig => {
  const now = Date.now();

  // If both of them are missing fallback to perMinute and fetch lastHour to show latest
  if (!startTime && !endTime) {
    return {
      granularity: "perMinute",
      startTime: now - HOUR_IN_MS,
      endTime: now,
    };
  }

  // Set default end time if missing
  const effectiveEndTime = endTime ?? now;
  // Set default start time if missing (last hour)
  const effectiveStartTime = startTime ?? effectiveEndTime - HOUR_IN_MS;
  const timeRange = effectiveEndTime - effectiveStartTime;
  let granularity: TimeseriesGranularity;

  if (timeRange > WEEK_IN_MS) {
    // > 7 days
    granularity = "perDay";
  } else if (timeRange > HOUR_IN_MS) {
    // > 1 hour
    granularity = "perHour";
  } else {
    // <= 1 hour
    granularity = "perMinute";
  }

  return {
    granularity,
    startTime: effectiveStartTime,
    endTime: effectiveEndTime,
  };
};
