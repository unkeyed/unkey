import { db } from "@/lib/db";
import { requireUser, requireWorkspace, t } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const healthStatusSchema = z.enum([
  "normal",
  "unstable",
  "degraded",
  "unhealthy",
  "recovering",
  "health_syncing",
  "unknown",
  "disabled",
]);

const generatorConfigSchema = z.object({
  deploymentId: z.string(),
  regions: z.number().min(1).max(7),
  instancesPerRegion: z.object({
    min: z.number().min(1).max(20),
    max: z.number().min(1).max(20),
  }),
  healthDistribution: z.record(healthStatusSchema, z.number().min(0).max(100)).optional(),
  regionDirection: z.enum(["vertical", "horizontal"]).optional(),
  instanceDirection: z.enum(["vertical", "horizontal"]).optional(),
});

type HealthStatus = z.infer<typeof healthStatusSchema>;

type RegionMetadata = {
  type: "region";
  flagCode: "us" | "de" | "au" | "jp" | "in" | "br";
  zones: number;
  instances: number;
  replicas: number;
  rps?: number;
  cpu?: number;
  memory?: number;
  storage?: number;
  latency: string;
  health: HealthStatus;
};

type InstanceMetadata = {
  type: "gateway";
  description: string;
  replicas: number;
  rps?: number;
  cpu?: number;
  memory?: number;
  latency: string;
  health: HealthStatus;
};

export type DeploymentNode = {
  id: string;
  label: string;
  direction?: "vertical" | "horizontal";
  metadata: { type: "origin" } | RegionMetadata | InstanceMetadata;
  children?: DeploymentNode[];
};

export const generateDeploymentTree = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(generatorConfigSchema)
  .mutation(async ({ ctx, input }) => {
    try {
      // Verify deployment belongs to workspace
      const deployment = await db.query.deployments.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
        columns: { id: true },
      });

      if (!deployment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Deployment not found",
        });
      }

      const healthStatuses: HealthStatus[] = [
        "normal",
        "unstable",
        "degraded",
        "unhealthy",
        "recovering",
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

      const flags: Record<string, RegionMetadata["flagCode"]> = {
        "us-east-1": "us",
        "eu-central-1": "de",
        "ap-southeast-2": "au",
        "ap-northeast-1": "jp",
        "ap-south-1": "in",
        "sa-east-1": "br",
      };

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

      const selectedRegions = regions.slice(0, input.regions);

      const tree: DeploymentNode = {
        id: "internet",
        label: "INTERNET",
        metadata: { type: "origin" },
        children: selectedRegions.map((regionId) => {
          const instanceCount = getRandomInt(
            input.instancesPerRegion.min,
            input.instancesPerRegion.max,
          );
          const totalInstances = getRandomInt(20, 40);
          const regionHealth = getRandomHealth();

          const regionMetadata: RegionMetadata = {
            type: "region",
            flagCode: flags[regionId],
            zones: getRandomInt(1, 3),
            instances: totalInstances,
            replicas: 2,
            rps: getRandomInt(1000, 5000),
            cpu: getRandomInt(30, 80),
            memory: getRandomInt(40, 85),
            storage: getRandomInt(512, 1024),
            latency: `${(Math.random() * 5 + 1).toFixed(1)}ms`,
            health: regionHealth,
          };

          return {
            id: regionId,
            label: regionId,
            direction: input.instanceDirection ?? "vertical",
            metadata: regionMetadata,
            children: Array.from({ length: instanceCount }, (_, i) => {
              const instanceId = Math.random().toString(36).substring(2, 6);
              const instanceMetadata: InstanceMetadata = {
                type: "gateway",
                description: "Instance replica",
                replicas: 2,
                rps: getRandomInt(100, 500),
                cpu: getRandomInt(20, 70),
                memory: getRandomInt(30, 75),
                latency: `${(Math.random() * 8 + 2).toFixed(1)}ms`,
                health: getRandomHealth(),
              };

              return {
                id: `${regionId}-gw-${instanceId}-${i + 1}`,
                label: `gw-${instanceId}`,
                metadata: instanceMetadata,
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
