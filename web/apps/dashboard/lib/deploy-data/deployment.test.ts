import type { Deployment as ApiDeployment } from "@unkey/api/models/components";
import { describe, expect, it, vi } from "vitest";
import { deploymentSchema, withDeploymentDetails } from "./deployment";

const { listDetails } = vi.hoisted(() => ({ listDetails: vi.fn() }));

vi.mock("@/lib/trpc/client", () => ({
  trpcClient: { deploy: { deployment: { listDetails: { query: listDetails } } } },
}));

describe("withDeploymentDetails", () => {
  it.each([1_700_000_060_000, 0, null, undefined])(
    "preserves firstReadyAt %s without substituting other lifecycle timestamps",
    async (firstReadyAt) => {
      const projectId = `proj_${crypto.randomUUID()}`;
      const row: ApiDeployment = {
        id: `dep_${crypto.randomUUID()}`,
        status: "stopped",
        isCurrent: false,
        environment: "production",
        app: "api",
        project: "project",
        trigger: { via: "api" },
        availableActions: [],
        regions: [],
        issues: [],
        runtime: {
          vCpus: 1,
          memoryMib: 256,
          storageMib: 0,
          port: 8080,
          command: [],
          shutdownSignal: "SIGTERM",
          upstreamProtocol: "http1",
        },
        createdAt: 1_700_000_000_000,
        finishedAt: 1_700_000_070_000,
        updatedAt: 1_700_000_120_000,
      };
      listDetails.mockResolvedValueOnce({
        [row.id]: {
          id: row.id,
          projectId,
          appId: `app_${crypto.randomUUID()}`,
          environmentId: `env_${crypto.randomUUID()}`,
          desiredState: "stopped",
          triggerReason: null,
          hasOpenApiSpec: false,
          instances: [],
          desiredInstanceCount: 0,
          desiredRegions: [],
          ...(firstReadyAt === undefined ? {} : { firstReadyAt }),
        },
      });

      const deployments = await withDeploymentDetails([row], projectId, () => undefined);

      expect(deployments).toHaveLength(1);
      expect(deployments[0]).toMatchObject({
        firstReadyAt: firstReadyAt ?? null,
        createdAt: row.createdAt,
        updatedAt: row.updatedAt,
        buildEndedAt: row.finishedAt,
      });
      expect(deploymentSchema.safeParse(deployments[0]).success).toBe(true);
    },
  );
});
