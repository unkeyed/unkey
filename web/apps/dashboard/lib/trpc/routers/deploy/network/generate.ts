import type {
  DeploymentNode,
  HealthStatus,
  InstanceNode,
  SentinelNode,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/deployments/[deploymentId]/network/unkey-flow/components/nodes/types";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { mapRegionToFlag } from "./utils";

const healthStatusSchema = z.enum(["normal", "unhealthy", "health_syncing", "unknown", "disabled"]);

const generatorConfigSchema = z.object({
  sentinels: z.number().min(1).max(7),
  instancesPerSentinel: z.object({
    min: z.number().min(0).max(20),
    max: z.number().min(0).max(20),
  }),
  healthDistribution: z.record(healthStatusSchema, z.number().min(0).max(100)).optional(),
  regionDirection: z.enum(["vertical", "horizontal"]).optional(),
  instanceDirection: z.enum(["vertical", "horizontal"]).optional(),
});

export const generateDeploymentTree = workspaceProcedure
  .input(generatorConfigSchema)
  .mutation(async ({ input }): Promise<DeploymentNode> => {
    try {
      const healthStatuses: HealthStatus[] = [
        "normal",
        "unhealthy",
        "health_syncing",
        "unknown",
        "disabled",
      ];

      const regions = [
        "us-east-1",
        "eu-central-1",
        "ap-southeast-2",
        "ap-northeast-1",
        "ap-south-1",
        "sa-east-1",
      ] as const;

      const getRandomHealth = (): HealthStatus => {
        if (input.healthDistribution) {
          const rand = Math.random() * 100;
          let cumulative = 0;
          for (const [status, percentage] of Object.entries(input.healthDistribution)) {
            cumulative += percentage;
            if (rand <= cumulative) {
              return status as HealthStatus;
            }
          }
        }
        return healthStatuses[Math.floor(Math.random() * healthStatuses.length)];
      };

      const getRandomInt = (min: number, max: number) =>
        Math.floor(Math.random() * (max - min + 1)) + min;

      const selectedRegions = regions.slice(0, input.sentinels);

      const tree: DeploymentNode = {
        id: "internet",
        label: "INTERNET",
        direction: input.regionDirection ?? "horizontal",
        metadata: { type: "origin" },
        children: selectedRegions.map((regionId): SentinelNode => {
          const instanceCount = getRandomInt(
            input.instancesPerSentinel.min,
            input.instancesPerSentinel.max,
          );
          const totalInstances = getRandomInt(20, 40);
          const regionHealth = getRandomHealth();

          return {
            id: regionId,
            label: regionId,
            direction: input.instanceDirection ?? "horizontal",
            metadata: {
              type: "sentinel",
              flagCode: mapRegionToFlag(regionId),
              instances: totalInstances,
              replicas: 2,
              cpu: getRandomInt(200, 2000),
              memory: getRandomInt(256, 2048),
              latency: "—",
              health: regionHealth,
            },
            children: Array.from({ length: instanceCount }, (_, i): InstanceNode => {
              const instanceId = Math.random().toString(36).substring(2, 6);

              return {
                id: `${regionId}-s-${instanceId}-${i + 1}`,
                label: `s-${instanceId}`,
                metadata: {
                  type: "instance",
                  description: "Instance replica",
                  replicas: 2,
                  cpu: getRandomInt(100, 1000),
                  memory: getRandomInt(128, 1024),
                  latency: "—",
                  health: getRandomHealth(),
                },
              };
            }),
          };
        }),
      };

      return tree;
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to generate deployment tree",
      });
    }
  });
