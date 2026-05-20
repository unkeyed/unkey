import { DEFAULT_LOGS_SINCE, getTimestampFromRelative } from "@/lib/utils";
import type { SentinelLogsRequest } from "@unkey/clickhouse/src/sentinel";

export function transformSentinelLogsFilters(params: Omit<SentinelLogsRequest, "workspaceId">) {
  const hasAbsoluteRange = params.startTime !== 0 && params.endTime !== 0;
  const since = params.since !== "" ? params.since : hasAbsoluteRange ? "" : DEFAULT_LOGS_SINCE;

  let startTime = params.startTime;
  let endTime = params.endTime;

  if (since !== "") {
    startTime = getTimestampFromRelative(since);
    endTime = Date.now();
  }

  return {
    projectId: params.projectId,
    deploymentId: params.deploymentId ?? null,
    environmentId: params.environmentId ?? [],
    limit: params.limit,
    startTime,
    endTime,
    since,
    statusCodes: params.statusCodes ?? [],
    methods: params.methods ?? [],
    paths: params.paths ?? [],
    cursor: params.cursor ?? null,
  };
}
