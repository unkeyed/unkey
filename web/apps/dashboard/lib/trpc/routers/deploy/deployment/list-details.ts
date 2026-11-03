import { type InferSelectModel, and, db, eq, inArray, notInArray } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import {
  appRegionalSettings,
  deploymentTopology,
  deployments,
  instances,
  openapiSpecs,
  regions,
} from "@unkey/db/src/schema";
import { z } from "zod";
import { mapInstanceRow } from "./deployment-query-helpers";

type DesiredRegion = {
  region: { id: string; name: string; platform: string };
};

type TargetRegion = DesiredRegion & { replicas: number };

type DeploymentDetails = Pick<
  InferSelectModel<typeof deployments>,
  "id" | "projectId" | "appId" | "environmentId" | "desiredState" | "triggerReason"
> & {
  instances: ReturnType<typeof mapInstanceRow>[];
  desiredInstanceCount: number;
  desiredRegions: DesiredRegion[];
  targetRegions: TargetRegion[];
  hasOpenApiSpec: boolean;
};

// The deployments collection reads rows from the public API and merges in what
// the API does not return: parent ids, the desired state, the trigger reason,
// and the runtime details (instances with their last container exit, desired
// regions, and whether an OpenAPI spec was captured)
export const listDeploymentDetails = workspaceProcedure
  .input(z.object({ deploymentIds: z.array(z.string()).min(1).max(100) }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }): Promise<Record<string, DeploymentDetails>> => {
    const rows = await db
      .select({
        id: deployments.id,
        projectId: deployments.projectId,
        appId: deployments.appId,
        environmentId: deployments.environmentId,
        desiredState: deployments.desiredState,
        triggerReason: deployments.triggerReason,
      })
      .from(deployments)
      .where(
        and(
          eq(deployments.workspaceId, ctx.workspace.id),
          inArray(deployments.id, input.deploymentIds),
        ),
      );
    if (rows.length === 0) {
      return {};
    }

    const deploymentIds = rows.map((d) => d.id);
    const appIds = [...new Set(rows.map((d) => d.appId))];
    const environmentIds = [...new Set(rows.map((d) => d.environmentId))];

    const [specRows, instanceRows, regionalSettingsRows, topologyRows] = await Promise.all([
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
            eq(appRegionalSettings.workspaceId, ctx.workspace.id),
            inArray(appRegionalSettings.appId, appIds),
            inArray(appRegionalSettings.environmentId, environmentIds),
          ),
        ),
      db
        .select({
          deploymentId: deploymentTopology.deploymentId,
          regionId: regions.id,
          regionName: regions.name,
          regionPlatform: regions.platform,
          replicas: deploymentTopology.autoscalingReplicasMin,
        })
        .from(deploymentTopology)
        .innerJoin(deployments, eq(deployments.id, deploymentTopology.deploymentId))
        .innerJoin(regions, eq(regions.id, deploymentTopology.regionId))
        .where(
          and(
            eq(deploymentTopology.workspaceId, ctx.workspace.id),
            inArray(deploymentTopology.deploymentId, deploymentIds),
            eq(deployments.desiredState, "running"),
            notInArray(deployments.status, ["failed", "cancelled", "superseded", "skipped"]),
          ),
        ),
    ]);

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

    const desiredByAppEnv = new Map<
      string,
      { desiredInstanceCount: number; desiredRegions: DesiredRegion[] }
    >();
    for (const row of regionalSettingsRows) {
      const key = `${row.appId}:${row.environmentId}`;
      const regionEntry = {
        region: { id: row.regionId, name: row.regionName, platform: row.regionPlatform },
      };
      const existing = desiredByAppEnv.get(key);
      if (existing) {
        existing.desiredInstanceCount += row.replicas;
        existing.desiredRegions.push(regionEntry);
      } else {
        desiredByAppEnv.set(key, {
          desiredInstanceCount: row.replicas,
          desiredRegions: [regionEntry],
        });
      }
    }

    const targetsByDeployment = Map.groupBy(topologyRows, (row) => row.deploymentId);

    return Object.fromEntries(
      rows.map((row) => {
        const desired = desiredByAppEnv.get(`${row.appId}:${row.environmentId}`);
        return [
          row.id,
          {
            ...row,
            instances: instancesByDeployment.get(row.id) ?? [],
            desiredInstanceCount: desired?.desiredInstanceCount ?? 0,
            desiredRegions: desired?.desiredRegions ?? [],
            targetRegions: (targetsByDeployment.get(row.id) ?? []).map((target) => ({
              region: {
                id: target.regionId,
                name: target.regionName,
                platform: target.regionPlatform,
              },
              replicas: target.replicas,
            })),
            hasOpenApiSpec: specSet.has(row.id),
          },
        ];
      }),
    );
  });
