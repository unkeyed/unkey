import type { RatelimitQueryOverviewLogsPayload } from "@/app/(app)/[workspace]/ratelimits/[namespaceId]/_overview/components/table/query-logs.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { RatelimitOverviewLogsParams } from "@unkey/clickhouse/src/ratelimits";

export function transformFilters(
  params: RatelimitQueryOverviewLogsPayload,
): Omit<RatelimitOverviewLogsParams, "workspaceId" | "namespaceId"> {
  let startTime = params.startTime;
  let endTime = params.endTime;

  if (params.since) {
    startTime = getTimestampFromRelative(params.since);
    endTime = Date.now();
  }

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

  const sorts =
    params.sorts?.map((sort) => ({
      column: sort.column,
      direction: sort.direction,
    })) ?? null;

  return {
    limit: params.limit,
    startTime,
    endTime,
    identifiers,
    cursorTime: params.cursor ?? null,
    status,
    sorts, // Add sorts to the returned params
  };
}
