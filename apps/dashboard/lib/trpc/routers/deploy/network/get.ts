import type { DeploymentNode } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/[deploymentId]/components/unkey-flow/components/nodes";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";

export const getDeploymentTree = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .query(async () => {
    try {
      // TODO: When you have real deployment data, query it from the database
      // For now, return a default tree structure
      const defaultTree: DeploymentNode = {
        id: "internet",
        label: "INTERNET",
        direction: "horizontal",
        metadata: { type: "origin" },
        children: [
          {
            id: "us-east-1",
            label: "us-east-1",
            direction: "vertical",
            metadata: {
              type: "region",
              flagCode: "us",
              zones: 3,
              instances: 2,
              replicas: 2,
              rps: 2400,
              cpu: 45,
              memory: 62,
              storage: 800,
              latency: "2.3ms",
              health: "normal",
            },
            children: [
              {
                id: "us-east-1-s-a1b2-1",
                label: "s-a1b2",
                metadata: {
                  type: "sentinel",
                  description: "Instance replica",
                  replicas: 2,
                  rps: 320,
                  cpu: 42,
                  memory: 58,
                  latency: "3.1ms",
                  health: "normal",
                },
              },
              {
                id: "us-east-1-s-c3d4-2",
                label: "s-c3d4",
                metadata: {
                  type: "sentinel",
                  description: "Instance replica",
                  replicas: 2,
                  rps: 280,
                  cpu: 38,
                  memory: 52,
                  latency: "2.8ms",
                  health: "normal",
                },
              },
            },
            children: [
              {
                id: "eu-central-1-s-e5f6-1",
                label: "s-e5f6",
                metadata: {
                  type: "sentinel",
                  description: "Instance replica",
                  replicas: 2,
                  rps: 240,
                  cpu: 48,
                  memory: 64,
                  latency: "4.2ms",
                  health: "normal",
                },
              },
              {
                id: "eu-central-1-s-g7h8-2",
                label: "s-g7h8",
                metadata: {
                  type: "sentinel",
                  description: "Instance replica",
                  replicas: 2,
                  rps: 260,
                  cpu: 52,
                  memory: 70,
                  latency: "3.8ms",
                  health: "unstable",
                },
              },
              {
                id: "eu-central-1-s-i9j0-3",
                label: "s-i9j0",
                metadata: {
                  type: "sentinel",
                  description: "Instance replica",
                  replicas: 2,
                  rps: 220,
                  cpu: 46,
                  memory: 62,
                  latency: "4.0ms",
                  health: "normal",
                },
              },
            },
            children: [
              {
                id: "ap-southeast-2-s-k1l2-1",
                label: "s-k1l2",
                metadata: {
                  type: "sentinel",
                  description: "Instance replica",
                  replicas: 2,
                  rps: 180,
                  cpu: 35,
                  memory: 52,
                  latency: "4.5ms",
                  health: "normal",
                },
              },
              {
                id: "ap-southeast-2-s-m3n4-2",
                label: "s-m3n4",
                metadata: {
                  type: "sentinel",
                  description: "Instance replica",
                  replicas: 2,
                  rps: 200,
                  cpu: 40,
                  memory: 58,
                  latency: "3.9ms",
                  health: "normal",
                },
              },
            },
          ],
        },
      ],
    };

    return defaultTree;
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
