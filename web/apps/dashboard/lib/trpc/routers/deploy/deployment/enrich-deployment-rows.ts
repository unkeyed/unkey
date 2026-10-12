import { and, db, eq, inArray, sql } from "@/lib/db";
import type { LastExit } from "@/lib/types/deploy";
import {
  appRegionalSettings,
  deploymentSteps,
  instances,
  openapiSpecs,
  regions,
} from "@unkey/db/src/schema";
import { type FlagCode, mapRegionToFlag } from "../network/utils";
import {
  type DeploymentListSelection,
  lastExitsByDeployment,
  mapInstanceRow,
  normalizeDeploymentRow,
} from "./deployment-query-helpers";

export async function enrichDeploymentRows(
  workspaceId: string,
  deploymentRows: DeploymentListSelection[],
) {
  const details = await loadDeploymentDetails(workspaceId, deploymentRows);
  return deploymentRows.map((deployment) => ({
    ...deployment,
    ...normalizeDeploymentRow(deployment),
    ...(details.get(deployment.id) ?? EMPTY_DETAILS),
  }));
}

export type DeploymentDetails = {
  instances: ReturnType<typeof mapInstanceRow>[];
  buildEndedAt: number | null;
  lastExit: LastExit | null;
  desiredInstanceCount: number;
  desiredRegions: {
    region: { id: string; name: string; platform: string };
    flagCode: FlagCode;
  }[];
  hasOpenApiSpec: boolean;
};

const EMPTY_DETAILS: DeploymentDetails = {
  instances: [],
  buildEndedAt: null,
  lastExit: null,
  desiredInstanceCount: 0,
  desiredRegions: [],
  hasOpenApiSpec: false,
};

// The runtime state of deployments that the public API does not expose:
// instances, desired regions, the last container exit, build timing, and
// whether an OpenAPI spec was captured
export async function loadDeploymentDetails(
  workspaceId: string,
  deploymentRows: { id: string; appId: string; environmentId: string }[],
): Promise<Map<string, DeploymentDetails>> {
  if (deploymentRows.length === 0) {
    return new Map();
  }

  const deploymentIds = deploymentRows.map((d) => d.id);

  const appIds = [...new Set(deploymentRows.map((d) => d.appId))];
  const environmentIds = [...new Set(deploymentRows.map((d) => d.environmentId))];

  const [specRows, instanceRows, regionalSettingsRows, stepTimingRows] = await Promise.all([
    db
      .select({ deploymentId: openapiSpecs.deploymentId })
      .from(openapiSpecs)
      .where(inArray(openapiSpecs.deploymentId, deploymentIds)),
    db
      .select({
        id: instances.id,
        deploymentId: instances.deploymentId,
        regionId: regions.id,
        regionName: regions.name,
        regionPlatform: regions.platform,
        status: instances.status,
        containerStatus: instances.containerStatus,
      })
      .from(instances)
      .innerJoin(regions, eq(regions.id, instances.regionId))
      .where(inArray(instances.deploymentId, deploymentIds)),
    db
      .select({
        appId: appRegionalSettings.appId,
        environmentId: appRegionalSettings.environmentId,
        regionId: regions.id,
        regionName: regions.name,
        regionPlatform: regions.platform,
        replicas: appRegionalSettings.replicas,
      })
      .from(appRegionalSettings)
      .innerJoin(regions, eq(regions.id, appRegionalSettings.regionId))
      .where(
        and(
          eq(appRegionalSettings.workspaceId, workspaceId),
          inArray(appRegionalSettings.appId, appIds),
          inArray(appRegionalSettings.environmentId, environmentIds),
        ),
      ),
    // Build/deploy timing comes from deployment_steps, the only timestamps
    // stop/wake never mutate. openSteps counts steps still running so we
    // can tell an in-progress build (tick live) from a finished one.
    db
      .select({
        deploymentId: deploymentSteps.deploymentId,
        maxEndedAt: sql<number | null>`max(${deploymentSteps.endedAt})`,
        openSteps: sql<number>`sum(case when ${deploymentSteps.endedAt} is null then 1 else 0 end)`,
      })
      .from(deploymentSteps)
      .where(inArray(deploymentSteps.deploymentId, deploymentIds))
      .groupBy(deploymentSteps.deploymentId),
  ]);

  // buildEndedAt is the moment the pipeline finished: the latest step end,
  // but null while any step is still open so the row ticks live instead of
  // freezing a partial duration. Null when a deployment has no steps (old
  // or prebuilt-image rows) — the row then shows no duration.
  const buildEndedAtByDeployment = new Map<string, number | null>();
  for (const row of stepTimingRows) {
    const openSteps = Number(row.openSteps ?? 0);
    const maxEndedAt = row.maxEndedAt == null ? null : Number(row.maxEndedAt);
    buildEndedAtByDeployment.set(row.deploymentId, openSteps > 0 ? null : maxEndedAt);
  }

  const specSet = new Set(specRows.map((s) => s.deploymentId));
  const instancesByDeployment = new Map<string, ReturnType<typeof mapInstanceRow>[]>();
  for (const row of instanceRows) {
    const entry = mapInstanceRow(row);
    const list = instancesByDeployment.get(row.deploymentId);
    if (list) {
      list.push(entry);
    } else {
      instancesByDeployment.set(row.deploymentId, [entry]);
    }
  }
  const lastExitByDeployment = lastExitsByDeployment(instanceRows);

  const desiredStateByAppEnv = new Map<
    string,
    {
      desiredInstanceCount: number;
      desiredRegions: {
        region: { id: string; name: string; platform: string };
        flagCode: FlagCode;
      }[];
    }
  >();
  for (const row of regionalSettingsRows) {
    const key = `${row.appId}:${row.environmentId}`;
    const regionEntry = {
      region: { id: row.regionId, name: row.regionName, platform: row.regionPlatform },
      flagCode: mapRegionToFlag(row.regionName),
    };
    const replicaCount = row.replicas;
    const existing = desiredStateByAppEnv.get(key);
    if (existing) {
      existing.desiredInstanceCount += replicaCount;
      existing.desiredRegions.push(regionEntry);
    } else {
      desiredStateByAppEnv.set(key, {
        desiredInstanceCount: replicaCount,
        desiredRegions: [regionEntry],
      });
    }
  }

  return new Map(
    deploymentRows.map((deployment): [string, DeploymentDetails] => {
      const desired = desiredStateByAppEnv.get(`${deployment.appId}:${deployment.environmentId}`);
      return [
        deployment.id,
        {
          instances: instancesByDeployment.get(deployment.id) ?? [],
          buildEndedAt: buildEndedAtByDeployment.get(deployment.id) ?? null,
          lastExit: lastExitByDeployment.get(deployment.id) ?? null,
          desiredInstanceCount: desired?.desiredInstanceCount ?? 0,
          desiredRegions: desired?.desiredRegions ?? [],
          hasOpenApiSpec: specSet.has(deployment.id),
        },
      ];
    }),
  );
}
