import type { RuntimeLogsRequestSchema } from "@/lib/schemas/runtime-logs.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { RuntimeLogsRequest } from "@unkey/clickhouse/src/runtime-logs";

export function transformFilters(
  params: RuntimeLogsRequestSchema,
): Omit<RuntimeLogsRequest, "workspaceId"> {
  const severity = params.severity?.filters?.map((f) => f.value) || [];

  let startTime = params.startTime;
  let endTime = params.endTime;

  const hasRelativeTime = params.since !== "";
  if (hasRelativeTime) {
    startTime = getTimestampFromRelative(params.since);
    endTime = Date.now();
  }

  return {
    projectId: params.projectId,
    deploymentId: params.deploymentId,
    environmentId: params.environmentId,
    limit: params.limit,
    startTime,
    endTime,
    severity,
    searchText: params.searchText,
    cursorTime: params.cursor ?? null,
  };
}
