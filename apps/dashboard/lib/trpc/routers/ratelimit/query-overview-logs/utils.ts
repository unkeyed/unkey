import type { RatelimitQueryOverviewLogsPayload } from "@/app/(app)/ratelimits/[namespaceId]/_overview/components/table/query-logs.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { RatelimitOverviewLogsParams } from "@unkey/clickhouse/src/ratelimits";

export function transformFilters(
  params: RatelimitQueryOverviewLogsPayload,
): Omit<RatelimitOverviewLogsParams, "workspaceId" | "namespaceId"> {
  let startTime = params.startTime;
  let endTime = params.endTime;

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

  return {
    limit: params.limit,
    startTime,
    endTime,
    identifiers,
    cursorTime: params.cursor?.time ?? null,
    cursorRequestId: params.cursor?.requestId ?? null,
  };
}
