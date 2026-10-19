import { getTimestampFromRelative } from "@/lib/duration";
import type { LogsFilterValue } from "@/lib/schemas/logs.filter.schema";
import { getUnkeyClient } from "@/lib/unkey-client";
import {
  type ApiRequestLog,
  HttpMethod,
  type IntFilter,
  type TextFilter,
  type V2LogsListApiRequestsFilters,
  type V2LogsListApiRequestsRequestBody,
} from "@unkey/api/models/components";
/** One row of the /logs table and its details panel, in the table's column names */
export type Log = {
  request_id: string;
  time: number;
  host: string;
  method: string;
  path: string;
  request_headers: string[];
  request_body: string;
  response_status: number;
  response_headers: string[];
  response_body: string;
  service_latency: number;
};

export type ApiRequestsPage = {
  logs: Log[];
  cursor: string | undefined;
};

export async function listApiRequests(
  body: V2LogsListApiRequestsRequestBody,
  signal?: AbortSignal,
): Promise<ApiRequestsPage> {
  const { result } = await getUnkeyClient().logs.listApiRequests(body, { signal });
  if (result.pagination.hasMore && !result.pagination.cursor) {
    throw new Error("API request logs returned a continuation page without a cursor");
  }

  return {
    logs: result.data.map(toLog),
    cursor: result.pagination.hasMore ? result.pagination.cursor : undefined,
  };
}

function toLog(row: ApiRequestLog): Log {
  return {
    request_id: row.requestId,
    time: row.time,
    host: row.host,
    method: row.method,
    path: row.path,
    request_headers: row.requestHeaders,
    request_body: row.requestBody,
    response_status: row.status,
    response_headers: row.responseHeaders,
    response_body: row.responseBody,
    service_latency: row.latencyMs,
  };
}

/**
 * Builds the API filters for the /logs page. A status family (200, 400, 500)
 * becomes a range, and several families become the range that spans them. The
 * API takes one range per field, so a gap such as 2xx with 5xx widens to
 * 2xx-5xx, and exact codes next to a family are dropped. The status filter
 * picks one family, but saved URLs and AI search can still produce either
 */
export function toApiRequestsFilters(
  filters: LogsFilterValue[],
  window: { startTime: number; endTime: number },
): V2LogsListApiRequestsFilters {
  const time = { gte: window.startTime, lte: window.endTime };
  let path: TextFilter | undefined;
  let method: HttpMethod[] | undefined;
  let requestId: string[] | undefined;
  const statuses: number[] = [];
  let since = "";

  for (const { field, operator, value } of filters) {
    switch (field) {
      case "status":
        statuses.push(Number(value));
        break;
      case "methods":
        if (isHttpMethod(value)) {
          method = [...(method ?? []), value];
        }
        break;
      case "paths":
        path = { ...path, [operator]: [...(path?.[operator] ?? []), String(value)] };
        break;
      case "requestId":
        requestId = [...(requestId ?? []), String(value)];
        break;
      case "startTime":
        time.gte = Number(value);
        break;
      case "endTime":
        time.lte = Number(value);
        break;
      case "since":
        since = String(value);
        break;
    }
  }

  // A relative window wins over explicit bounds and ends at the page's query
  // time, so it never starts after it ends
  if (since) {
    time.gte = getTimestampFromRelative(since, window.endTime);
    time.lte = window.endTime;
  }

  return {
    time,
    path,
    method: method && { is: method },
    requestId: requestId && { is: requestId },
    status: toStatusFilter(statuses),
  };
}

const STATUS_FAMILIES = new Set([200, 400, 500]);

export function toStatusFilter(statuses: number[]): IntFilter | undefined {
  if (statuses.length === 0) {
    return undefined;
  }

  const families = statuses.filter((status) => STATUS_FAMILIES.has(status));
  if (families.length === 0) {
    return { is: statuses };
  }

  return { gte: Math.min(...families), lt: Math.max(...families) + 100 };
}

const HTTP_METHODS: ReadonlySet<string> = new Set(Object.values(HttpMethod));

function isHttpMethod(value: unknown): value is HttpMethod {
  return typeof value === "string" && HTTP_METHODS.has(value);
}
