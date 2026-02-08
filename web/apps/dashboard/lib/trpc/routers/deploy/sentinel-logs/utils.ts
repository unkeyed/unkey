import { getTimestampFromRelative } from "@/lib/utils";
import type { SentinelLogsRequest } from "@unkey/clickhouse/src/sentinel";

export function transformSentinelLogsFilters(params: Omit<SentinelLogsRequest, "workspaceId">) {
  // Handle time ranges
  let startTime = params.startTime;
  let endTime = params.endTime;

  const hasRelativeTime = params.since !== "";
  if (hasRelativeTime) {
    startTime = getTimestampFromRelative(params.since);
    endTime = Date.now();
  }

  return {
    projectId: params.projectId,
    deploymentId: params.deploymentId ?? null,
    environmentId: params.environmentId ?? null,
    limit: params.limit,
    startTime,
    endTime,
    since: params.since,
    statusCodes: params.statusCodes ?? [],
    methods: params.methods ?? [],
    paths: params.paths ?? [],
    cursor: params.cursor ?? null,
  };
}
