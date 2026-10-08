import { type InferSelectModel, and, db, eq, inArray } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import type { LastExit } from "@/lib/types/deploy";
import {
  appRegionalSettings,
  deployments,
  instances,
  openapiSpecs,
  regions,
} from "@unkey/db/src/schema";
import { z } from "zod";
import { type FlagCode, mapRegionToFlag } from "../network/utils";
import { lastExitsByDeployment, mapInstanceRow } from "./deployment-query-helpers";

type DesiredRegion = {
  region: { id: string; name: string; platform: string };
  flagCode: FlagCode;
};

type DeploymentDetails = Pick<
  InferSelectModel<typeof deployments>,
  "id" | "projectId" | "appId" | "environmentId" | "desiredState" | "triggerReason"
> & {
  instances: ReturnType<typeof mapInstanceRow>[];
  lastExit: LastExit | null;
  desiredInstanceCount: number;
  desiredRegions: DesiredRegion[];
  hasOpenApiSpec: boolean;
};

// The deployments collection reads rows from the public API and merges in what
// the API does not return: parent ids, the desired state, the trigger reason,
// and the runtime details (instances, desired regions, the last container exit,
// and whether an OpenAPI spec was captured)
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

    const [specRows, instanceRows, regionalSettingsRows] = await Promise.all([
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
    const lastExitByDeployment = lastExitsByDeployment(instanceRows);

    const desiredByAppEnv = new Map<
      string,
      { desiredInstanceCount: number; desiredRegions: DesiredRegion[] }
    >();
    for (const row of regionalSettingsRows) {
      const key = `${row.appId}:${row.environmentId}`;
      const regionEntry = {
        region: { id: row.regionId, name: row.regionName, platform: row.regionPlatform },
        flagCode: mapRegionToFlag(row.regionName),
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

    return Object.fromEntries(
      rows.map((row) => {
        const desired = desiredByAppEnv.get(`${row.appId}:${row.environmentId}`);
        return [
          row.id,
          {
            ...row,
            instances: instancesByDeployment.get(row.id) ?? [],
            lastExit: lastExitByDeployment.get(row.id) ?? null,
            desiredInstanceCount: desired?.desiredInstanceCount ?? 0,
            desiredRegions: desired?.desiredRegions ?? [],
            hasOpenApiSpec: specSet.has(row.id),
          },
        ];
      }),
    );
  });
