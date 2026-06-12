import { DEFAULT_LOGS_SINCE, getTimestampFromRelative } from "@/lib/utils";
import type { SentinelLogsRequest } from "@unkey/clickhouse/src/sentinel";

export function transformSentinelLogsFilters(params: Omit<SentinelLogsRequest, "workspaceId">) {
  let since: string;
  let startTime: number;
  let endTime: number;
  if (params.since !== null && params.since !== "") {
    since = params.since;
    startTime = getTimestampFromRelative(since);
    endTime = Date.now();
  } else if (params.startTime !== null && params.endTime !== null) {
    since = "";
    startTime = params.startTime;
    endTime = params.endTime;
  } else {
    since = DEFAULT_LOGS_SINCE;
    startTime = getTimestampFromRelative(since);
    endTime = Date.now();
  }

  return {
    projectId: params.projectId,
    deploymentId: params.deploymentId ?? [],
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
