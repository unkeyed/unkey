import type { RuntimeLogsRequestSchema } from "@/lib/schemas/runtime-logs.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { RuntimeLogsRequest } from "@unkey/clickhouse/src/runtime-logs";

export function transformFilters(
  params: RuntimeLogsRequestSchema,
): Omit<
  RuntimeLogsRequest,
  "workspaceId" | "projectId" | "environmentId" | "deploymentId" | "appId" | "k8sPodNames"
> {
  const severity = params.severity?.filters?.map((f) => f.value) || [];
  const region = params.region?.filters?.map((f) => f.value) || [];

  let startTime = params.startTime;
  let endTime = params.endTime;

  const hasRelativeTime = params.since !== "";
  if (hasRelativeTime) {
    startTime = getTimestampFromRelative(params.since);
    endTime = Date.now();
  }

  return {
    limit: params.limit,
    startTime,
    endTime,
    severity,
    region,
    message: params.message,
    cursorTime: params.cursor ?? null,
  };
}
