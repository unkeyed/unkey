import type { DeploymentNode } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/deployments/[deploymentId]/network/unkey-flow/components/nodes";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { calculateSentinelHealth, mapInstanceStatusToHealth, mapRegionToFlag } from "./utils";

export const getDeploymentTree = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(z.object({ deploymentId: z.string() }))
  .query(async ({ ctx, input }) => {
    try {
      // Fetch deployment to get environmentId
      const deployment = await db.query.deployments.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          environmentId: true,
          projectId: true,
        },
      });

      if (!deployment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Deployment not found",
        });
      }

      const [instances, sentinels] = await Promise.all([
        db.query.instances.findMany({
          where: (table, { eq, and }) =>
            and(
              eq(table.deploymentId, input.deploymentId),
              eq(table.projectId, deployment.projectId),
              eq(table.workspaceId, ctx.workspace.id),
            ),
          columns: {
            id: true,
            cpuMillicores: true,
            memoryMib: true,
            status: true,
            message: true,
            failureReason: true,
          },
          with: {
            region: {
              columns: {
                name: true,
              },
            },
          },
        }),
        db.query.sentinels.findMany({
          where: (table, { eq, and }) =>
            and(
              eq(table.environmentId, deployment.environmentId),
              eq(table.workspaceId, ctx.workspace.id),
            ),
          columns: {
            id: true,
            regionId: true,
            health: true,
            availableReplicas: true,
            cpuMillicores: true,
            memoryMib: true,
          },
          with: { region: true },
        }),
      ]);

      // Group instances by region
      const instancesByRegion = Object.groupBy(instances, ({ region }) => region?.name ?? "");

      // Build tree structure: each sentinel node has instances as children
      const children = sentinels.map(
        ({ id, region, availableReplicas, cpuMillicores, memoryMib, health }) => {
          const sentinelInstances = instancesByRegion[region.name] ?? [];

          return {
            id,
            label: region.name,
            direction: "vertical" as const,
            metadata: {
              type: "sentinel" as const,
              flagCode: mapRegionToFlag(region.name),
              instances: sentinelInstances.length,
              replicas: availableReplicas,
              cpu: cpuMillicores,
              memory: memoryMib,
              latency: "—",
              health: calculateSentinelHealth(sentinelInstances, health),
            },
            children: sentinelInstances.map(({ id, status, cpuMillicores, memoryMib, message, failureReason }) => ({
              id,
              label: id,
              metadata: {
                type: "instance" as const,
                description: "Instance replica",
                replicas: availableReplicas,
                cpu: cpuMillicores,
                memory: memoryMib,
                latency: "—",
                health: mapInstanceStatusToHealth(status),
                message: message ?? undefined,
                failureReason: failureReason ?? undefined,
              },
            })),
          };
        },
      );

      const tree: DeploymentNode = {
        id: "internet",
        label: "INTERNET",
        direction: "horizontal",
        metadata: { type: "origin" },
        children,
      };

      return tree;
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployment tree",
      });
    }
  });
