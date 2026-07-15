import type { RatelimitQueryLogsPayload } from "@/components/ratelimit-logs-table/schema/query-logs.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { RatelimitLogsParams } from "@unkey/clickhouse/src/ratelimits";

export function transformFilters(
  params: RatelimitQueryLogsPayload,
): Omit<RatelimitLogsParams, "workspaceId" | "namespaceId"> {
  let startTime = params.startTime;
  let endTime = params.endTime;

  const page = params.page ?? 1;
  const offset = (page - 1) * params.limit;

  // If we have relativeTime filter `since`, ignore other time params
  if (params.since) {
    startTime = getTimestampFromRelative(params.since);
    endTime = Date.now();
  }

  // Transform identifier filters
  const identifiers =
    params.identifiers?.filters.map((f) => ({
      operator: f.operator,
      value: f.value,
    })) ?? [];

  const status =
    params.status?.filters.map((f) => ({
      operator: "is" as const,
      value: f.value,
    })) ?? [];

  return {
    limit: params.limit,
    startTime,
    endTime,
    identifiers,
    requestIds: params.requestIds?.filters.map((f) => f.value) || [],
    status,
    sorts: params.sorts ?? null,
    offset,
  };
}
