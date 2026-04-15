import { and, db, desc, eq, inArray } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { deployments, instances, openapiSpecs, regions } from "@unkey/db/src/schema";
import { z } from "zod";
import { mapRegionToFlag } from "../network/utils";

export const listDeployments = workspaceProcedure
  .input(z.object({ projectId: z.string() }))
  .query(async ({ ctx, input }) => {
    try {
      const deploymentRows = await db
        .select({
          id: deployments.id,
          projectId: deployments.projectId,
          environmentId: deployments.environmentId,
          gitCommitSha: deployments.gitCommitSha,
          gitBranch: deployments.gitBranch,
          gitCommitMessage: deployments.gitCommitMessage,
          gitCommitAuthorHandle: deployments.gitCommitAuthorHandle,
          gitCommitAuthorAvatarUrl: deployments.gitCommitAuthorAvatarUrl,
          gitCommitTimestamp: deployments.gitCommitTimestamp,
          prNumber: deployments.prNumber,
          forkRepositoryFullName: deployments.forkRepositoryFullName,
          status: deployments.status,
          cpuMillicores: deployments.cpuMillicores,
          memoryMib: deployments.memoryMib,
          storageMib: deployments.storageMib,
          createdAt: deployments.createdAt,
        })
        .from(deployments)
        .where(
          and(
            eq(deployments.workspaceId, ctx.workspace.id),
            eq(deployments.projectId, input.projectId),
          ),
        )
        .orderBy(desc(deployments.createdAt))
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
      const instancesByDeployment = new Map<
        string,
        {
          id: string;
          region: { id: string; name: string; platform: string };
          flagCode: ReturnType<typeof mapRegionToFlag>;
        }[]
      >();
      for (const row of instanceRows) {
        const list = instancesByDeployment.get(row.deploymentId);
        const entry = {
          id: row.id,
          region: { id: row.regionId, name: row.regionName, platform: row.regionPlatform },
          flagCode: mapRegionToFlag(row.regionName),
        };
        if (list) {
          list.push(entry);
        } else {
          instancesByDeployment.set(row.deploymentId, [entry]);
        }
      }

      return deploymentRows.map((deployment) => ({
        ...deployment,
        instances: instancesByDeployment.get(deployment.id) ?? [],
        gitBranch: deployment.gitBranch ?? "",
        prNumber: deployment.prNumber ?? null,
        forkRepositoryFullName: deployment.forkRepositoryFullName ?? null,
        gitCommitAuthorAvatarUrl:
          deployment.gitCommitAuthorAvatarUrl ?? "https://github.com/identicons/dummy-user.png",
        hasOpenApiSpec: specSet.has(deployment.id),
        gitCommitTimestamp: deployment.gitCommitTimestamp,
      }));
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments",
      });
    }
  });
