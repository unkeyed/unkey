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
      // Fetch deployment to get environmentId and resource specs
      const deployment = await db.query.deployments.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          environmentId: true,
          cpuMillicores: true,
          memoryMib: true,
        },
      });

      if (!deployment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Deployment not found",
        });
      }

      // Fetch instances and sentinels in parallel
      const [instances, sentinels] = await Promise.all([
        db.query.instances.findMany({
          where: (table, { eq, and }) =>
            and(
              eq(table.deploymentId, input.deploymentId),
              eq(table.workspaceId, ctx.workspace.id),
            ),
          columns: {
            id: true,
            region: true,
            cpuMillicores: true,
            memoryMib: true,
            status: true,
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
            region: true,
            health: true,
            availableReplicas: true,
            cpuMillicores: true,
            memoryMib: true,
          },
        }),
      ]);

      // Group instances by region
      const instancesByRegion = Object.groupBy(instances, ({ region }) => region);

      // Build tree structure: each sentinel node has instances as children
      const children = sentinels.map(
        ({ id, region, availableReplicas, cpuMillicores, memoryMib, health }) => {
          const sentinelInstances = instancesByRegion[region] ?? [];

          return {
            id,
            label: region,
            direction: "vertical" as const,
            metadata: {
              type: "sentinel" as const,
              flagCode: mapRegionToFlag(region),
              instances: sentinelInstances.length,
              replicas: availableReplicas,
              cpu: cpuMillicores,
              memory: memoryMib,
              latency: "—",
              health: calculateSentinelHealth(sentinelInstances, health),
            },
            children: sentinelInstances.map(({ id, status, cpuMillicores, memoryMib }) => ({
              id,
              label: `s-${id.slice(-4)}`,
              metadata: {
                type: "instance" as const,
                description: "Instance replica",
                replicas: availableReplicas,
                cpu: cpuMillicores,
                memory: memoryMib,
                latency: "—",
                health: mapInstanceStatusToHealth(status),
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
