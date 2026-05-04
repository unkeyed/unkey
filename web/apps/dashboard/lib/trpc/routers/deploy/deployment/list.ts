import { and, db, desc, eq, gte, inArray, lte } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { deployments, instances, openapiSpecs, regions } from "@unkey/db/src/schema";
import { z } from "zod";
import {
  deploymentSelectFields,
  mapInstanceRow,
  normalizeDeploymentRow,
} from "./deployment-query-helpers";

export const listDeployments = workspaceProcedure
  .input(
    z.object({
      projectId: z.string(),
      startTime: z.number().int().optional(),
      endTime: z.number().int().optional(),
    }),
  )
  .query(async ({ ctx, input }) => {
    try {
      const deploymentRows = await db
        .select(deploymentSelectFields)
        .from(deployments)
        .where(
          and(
            eq(deployments.workspaceId, ctx.workspace.id),
            eq(deployments.projectId, input.projectId),
            input.startTime ? gte(deployments.createdAt, input.startTime) : undefined,
            input.endTime ? lte(deployments.createdAt, input.endTime) : undefined,
          ),
        )
        .orderBy(desc(deployments.createdAt), desc(deployments.id))
        .limit(100);

      if (deploymentRows.length === 0) {
        return [];
      }

      const deploymentIds = deploymentRows.map((d) => d.id);
      const [specRows, instanceRows] = await Promise.all([
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
          })
          .from(instances)
          .innerJoin(regions, eq(regions.id, instances.regionId))
          .where(inArray(instances.deploymentId, deploymentIds)),
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

      return deploymentRows.map((deployment) => ({
        ...deployment,
        ...normalizeDeploymentRow(deployment),
        instances: instancesByDeployment.get(deployment.id) ?? [],
        hasOpenApiSpec: specSet.has(deployment.id),
      }));
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments",
      });
    }
  });
