import type { SentinelLogsRequestSchema } from "@unkey/clickhouse/src/sentinel";

export function transformSentinelLogsFilters(
  params: Omit<SentinelLogsRequestSchema, "workspaceId">,
) {
  return {
    projectId: params.projectId,
    deploymentId: params.deploymentId,
    environmentId: params.environmentId,
    limit: params.limit,
    startTime: params.startTime,
    endTime: params.endTime,
  };
}
