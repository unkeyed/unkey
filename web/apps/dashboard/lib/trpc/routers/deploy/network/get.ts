import type {
  DeploymentNode,
  HealthStatus,
  RegionNode,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/deployments/[deploymentId]/network/unkey-flow/components/nodes/types";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { mapInstanceStatusToHealth, mapRegionToFlag } from "./utils";

export const getDeploymentTree = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(z.object({ deploymentId: z.string() }))
  .query(async ({ ctx, input }) => {
    try {
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

      const instances = await db.query.instances.findMany({
        where: (table, { eq, and }) =>
          and(
            eq(table.deploymentId, input.deploymentId),
            eq(table.projectId, deployment.projectId),
            eq(table.workspaceId, ctx.workspace.id),
          ),
        columns: {
          id: true,
          k8sName: true,
          cpuMillicores: true,
          memoryMib: true,
          status: true,
          // Kubelet-detailed status (exit info, waiting reason). Drizzle
          // hands us the typed JSON shape; the instance panel projects
          // out lastTerminationState / waiting so a starting/failed pod
          // shows "OOMKilled · exit=137" inline, not just a generic "is
          // starting up" message.
          containerStatus: true,
        },
        with: {
          region: {
            columns: {
              id: true,
              name: true,
            },
          },
        },
      });

      // Group instances by region. Each region with at least one instance
      // becomes a child of the INTERNET origin; instances within a region
      // are children of that region node.
      const instancesByRegionName = Object.groupBy(instances, ({ region }) => region?.name ?? "");

      const children: RegionNode[] = Object.entries(instancesByRegionName)
        .filter(([regionName]) => regionName !== "")
        .map(([regionName, regionInstances = []]) => {
          // Region health is derived from the worst instance status: any
          // failed instance flips the region red, any pending instance
          // shows syncing, otherwise normal.
          const hasUnhealthy = regionInstances.some((i) => i.status === "failed");
          const hasSyncing = regionInstances.some((i) => i.status === "pending");
          const health: HealthStatus = hasUnhealthy
            ? "unhealthy"
            : hasSyncing
              ? "health_syncing"
              : "normal";

          // Stable id so React can reuse the node across refetches.
          const regionId = regionInstances[0]?.region?.id ?? regionName;

          return {
            id: `region-${regionId}`,
            label: regionName,
            direction: "vertical" as const,
            metadata: {
              type: "region" as const,
              flagCode: mapRegionToFlag(regionName),
              instances: regionInstances.length,
              health,
            },
            children: regionInstances.map((instance) => ({
              id: instance.id,
              label: instance.id,
              metadata: {
                type: "instance" as const,
                description: "Instance replica",
                cpu: instance.cpuMillicores,
                memory: instance.memoryMib,
                latency: "—",
                health: mapInstanceStatusToHealth(instance.status),
                k8sName: instance.k8sName,
                // Forward the denormalized last-exit context so the instance
                // details panel can render it without a follow-up query.
                // Project the JSON's optional sub-fields into the flat
                // shape consumers already expect; null fields stay null
                // and the panel hides empties.
                lastExit:
                  instance.containerStatus.lastTerminationState !== undefined ||
                  instance.containerStatus.waiting !== undefined
                    ? {
                        restartCount: instance.containerStatus.restartCount ?? 0,
                        exitCode: instance.containerStatus.lastTerminationState?.exitCode ?? null,
                        signal: instance.containerStatus.lastTerminationState?.signal ?? null,
                        reason: instance.containerStatus.lastTerminationState?.reason ?? null,
                        statusReason: instance.containerStatus.waiting?.reason ?? null,
                        finishedAt:
                          instance.containerStatus.lastTerminationState?.finishedAt ?? null,
                      }
                    : null,
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
