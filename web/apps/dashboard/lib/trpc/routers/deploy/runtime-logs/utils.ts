import { db, inArray, schema } from "@/lib/db";
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

export async function resolveK8sNamesToInstanceIds(
  k8sPodNames: string[],
): Promise<Map<string, string>> {
  const mapping = new Map<string, string>();
  if (k8sPodNames.length === 0) {
    return mapping;
  }

  const instances = await db.query.instances.findMany({
    where: inArray(schema.instances.k8sName, k8sPodNames),
    columns: { id: true, k8sName: true },
  });
  for (const inst of instances) {
    mapping.set(inst.k8sName, inst.id);
  }
  return mapping;
}
