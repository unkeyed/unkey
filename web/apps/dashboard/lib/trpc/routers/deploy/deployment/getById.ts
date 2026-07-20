import { and, db, eq } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import {
  appRegionalSettings,
  deployments,
  instances,
  openapiSpecs,
  regions,
} from "@unkey/db/src/schema";
import { z } from "zod";
import { mapRegionToFlag } from "../network/utils";
import {
  computeLastExit,
  deploymentSelectFields,
  mapInstanceRow,
  normalizeDeploymentRow,
} from "./deployment-query-helpers";

export const getById = workspaceProcedure
  .input(
    z.object({
      deploymentId: z.string(),
      projectId: z.string(),
    }),
  )
  .query(async ({ input, ctx }) => {
    try {
      const [deployment] = await db
        .select({
          ...deploymentSelectFields,
          appId: deployments.appId,
        })
        .from(deployments)
        .where(
          and(
            eq(deployments.id, input.deploymentId),
            eq(deployments.workspaceId, ctx.workspace.id),
            eq(deployments.projectId, input.projectId),
          ),
        )
        .limit(1);

      if (!deployment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Deployment not found",
        });
      }

      const [specRows, instanceRows, regionalSettingsRows] = await Promise.all([
        db
          .select({ deploymentId: openapiSpecs.deploymentId })
          .from(openapiSpecs)
          .where(eq(openapiSpecs.deploymentId, deployment.id)),
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
          .where(eq(instances.deploymentId, deployment.id)),
        db
          .select({
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
              eq(appRegionalSettings.appId, deployment.appId),
              eq(appRegionalSettings.environmentId, deployment.environmentId),
            ),
          ),
      ]);

      let desiredInstanceCount = 0;
      const desiredRegions: {
        region: { id: string; name: string; platform: string };
        flagCode: ReturnType<typeof mapRegionToFlag>;
      }[] = [];
      for (const row of regionalSettingsRows) {
        desiredInstanceCount += row.replicas;
        desiredRegions.push({
          region: { id: row.regionId, name: row.regionName, platform: row.regionPlatform },
          flagCode: mapRegionToFlag(row.regionName),
        });
      }

      return {
        ...deployment,
        ...normalizeDeploymentRow(deployment),
        instances: instanceRows.map(mapInstanceRow),
        lastExit: computeLastExit(instanceRows),
        desiredInstanceCount,
        desiredRegions,
        hasOpenApiSpec: specRows.length > 0,
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployment",
      });
    }
  });
