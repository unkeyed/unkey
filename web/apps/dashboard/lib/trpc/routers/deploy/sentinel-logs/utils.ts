import type { SentinelLogsRequest } from "@unkey/clickhouse/src/sentinel";

export function transformSentinelLogsFilters(params: Omit<SentinelLogsRequest, "workspaceId">) {
  return {
    projectId: params.projectId,
    deploymentId: params.deploymentId,
    environmentId: params.environmentId,
    limit: params.limit,
  };
}
