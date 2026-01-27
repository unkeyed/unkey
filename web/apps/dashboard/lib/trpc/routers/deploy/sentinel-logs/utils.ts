import type { SentinelLogsRequestSchema } from "@/lib/schemas/sentinel-logs";
import type { SentinelRequest } from "@unkey/clickhouse/src/sentinel";

export function transformSentinelLogsFilters(
  params: SentinelLogsRequestSchema,
): Omit<SentinelRequest, "workspaceId"> {
  return {
    projectId: params.projectId,
    deploymentId: params.deploymentId,
    limit: params.limit,
    startTime: params.startTime,
    endTime: params.endTime,
  };
}
