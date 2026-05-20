import { and, db, eq, inArray, schema } from "@/lib/db";
import type { RuntimeLogsRequestSchema } from "@/lib/schemas/runtime-logs.schema";
import { getTimestampFromRelative } from "@/lib/utils";
import type { RuntimeLogsRequest } from "@unkey/clickhouse/src/runtime-logs";

export type K8sRegionEntry = { k8sPodName: string; region: string };

export const toInstanceKey = (k8sPodName: string, region: string): string =>
  `${k8sPodName}::${region}`;

export function uniqueK8sRegionEntries(
  logs: ReadonlyArray<{ k8s_pod_name: string; region: string }>,
  exclude?: ReadonlyMap<string, string>,
): K8sRegionEntry[] {
  const seen = new Set<string>();
  const entries: K8sRegionEntry[] = [];
  for (const log of logs) {
    const key = toInstanceKey(log.k8s_pod_name, log.region);
    if (!seen.has(key) && !exclude?.has(key)) {
      seen.add(key);
      entries.push({ k8sPodName: log.k8s_pod_name, region: log.region });
    }
  }
  return entries;
}

export function transformFilters(
  params: RuntimeLogsRequestSchema,
): Omit<
  RuntimeLogsRequest,
  "workspaceId" | "projectId" | "deploymentId" | "appId" | "k8sPodNames"
> {
  const severity = params.severity?.filters?.map((f) => f.value) || [];
  const region = params.region?.filters?.map((f) => f.value) || [];
  const environmentId = params.environmentId?.filters?.map((f) => f.value) || [];

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
    environmentId,
    message: params.message,
    cursorTime: params.cursor ?? null,
  };
}

export async function resolveK8sNamesToInstanceIds(
  entries: K8sRegionEntry[],
  workspaceId: string,
): Promise<Map<string, string>> {
  const mapping = new Map<string, string>();
  if (entries.length === 0) {
    return mapping;
  }

  const k8sPodNames = [...new Set(entries.map((e) => e.k8sPodName))];
  const instances = await db.query.instances.findMany({
    where: and(
      inArray(schema.instances.k8sName, k8sPodNames),
      eq(schema.instances.workspaceId, workspaceId),
    ),
    columns: { id: true, k8sName: true },
    with: { region: { columns: { name: true } } },
  });
  for (const inst of instances) {
    mapping.set(toInstanceKey(inst.k8sName, inst.region.name), inst.id);
  }
  return mapping;
}
