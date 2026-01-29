import type { DeploymentNode } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/deployments/[deploymentId]/network/unkey-flow/components/nodes";
import type { HealthStatus } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/deployments/[deploymentId]/network/unkey-flow/components/nodes/types";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

type FlagCode = "us" | "de" | "au" | "jp" | "in" | "br";

function mapDatabaseHealthToUI(
  dbHealth: "unknown" | "paused" | "healthy" | "unhealthy",
): HealthStatus {
  const mapping = {
    healthy: "normal",
    unhealthy: "unhealthy",
    paused: "disabled",
    unknown: "unknown",
  } as const;
  return mapping[dbHealth];
}

function mapInstanceStatusToHealth(
  status: "inactive" | "pending" | "running" | "failed",
): HealthStatus {
  const mapping = {
    running: "normal",
    pending: "health_syncing",
    inactive: "disabled",
    failed: "unhealthy",
  } as const;
  return mapping[status];
}

function calculateSentinelHealth(
  instances: Array<{ status: "inactive" | "pending" | "running" | "failed" }>,
  sentinels: Array<{ health: "unknown" | "paused" | "healthy" | "unhealthy" }>,
): HealthStatus {
  const healthPriority: Record<HealthStatus, number> = {
    unhealthy: 5,
    degraded: 4,
    unstable: 3,
    health_syncing: 2,
    recovering: 2,
    normal: 1,
    unknown: 0,
    disabled: 0,
  };

  const allHealth = [
    ...instances.map((i) => mapInstanceStatusToHealth(i.status)),
    ...sentinels.map((s) => mapDatabaseHealthToUI(s.health)),
  ];

  return allHealth.reduce(
    (worst, current) => (healthPriority[current] > healthPriority[worst] ? current : worst),
    "normal" as HealthStatus,
  );
}

function mapRegionToFlag(region: string): FlagCode {
  if (region.startsWith("us-")) return "us";
  if (region.startsWith("eu-")) return "de";
  if (region.startsWith("ap-southeast")) return "au";
  if (region.startsWith("ap-northeast")) return "jp";
  if (region.startsWith("ap-south")) return "in";
  if (region.startsWith("sa-")) return "br";
  return "us";
}

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

      // Fetch instances by deploymentId
      const instances = await db.query.instances.findMany({
        where: (table, { eq, and }) =>
          and(eq(table.deploymentId, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          id: true,
          region: true,
          cpuMillicores: true,
          memoryMib: true,
          status: true,
        },
      });

      // Fetch sentinels by environmentId
      const sentinels = await db.query.sentinels.findMany({
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
      });

      // Group instances by region (each sentinel manages one region)
      const instancesByRegion = new Map<string, typeof instances>();
      for (const instance of instances) {
        if (!instancesByRegion.has(instance.region)) {
          instancesByRegion.set(instance.region, []);
        }
        instancesByRegion.get(instance.region)!.push(instance);
      }

      // Build tree structure: each sentinel node has instances as children
      const children = sentinels.map((sentinel) => {
        const sentinelInstances = instancesByRegion.get(sentinel.region) || [];

        // Calculate aggregate metrics
        const totalInstanceCpu = sentinelInstances.reduce((sum, i) => sum + i.cpuMillicores, 0);
        const totalInstanceMemory = sentinelInstances.reduce((sum, i) => sum + i.memoryMib, 0);

        // Combined CPU/memory including sentinel
        const totalCpu = sentinel.cpuMillicores + totalInstanceCpu;
        const totalMemory = sentinel.memoryMib + totalInstanceMemory;

        // Calculate health from instances and sentinel
        const sentinelHealth = calculateSentinelHealth(sentinelInstances, [sentinel]);

        return {
          id: sentinel.id,
          label: sentinel.region,
          direction: "vertical" as const,
          metadata: {
            type: "sentinel" as const,
            flagCode: mapRegionToFlag(sentinel.region),
            instances: sentinelInstances.length,
            replicas: sentinel.availableReplicas,
            cpu: totalCpu,
            memory: totalMemory,
            latency: "—",
            health: sentinelHealth,
          },
          children: sentinelInstances.map((instance) => ({
            id: instance.id,
            label: `s-${instance.id.slice(-4)}`,
            metadata: {
              type: "instance" as const,
              description: "Instance replica",
              replicas: sentinel.availableReplicas,
              cpu: instance.cpuMillicores,
              memory: instance.memoryMib,
              latency: "—",
              health: mapInstanceStatusToHealth(instance.status),
            },
          })),
        };
      });

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
