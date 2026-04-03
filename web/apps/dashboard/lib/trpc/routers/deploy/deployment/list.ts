import { db } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { mapRegionToFlag } from "../network/utils";

export const listDeployments = workspaceProcedure
  .input(z.object({ projectId: z.string() }))
  .query(async ({ ctx, input }) => {
    try {
      const deployments = await db.query.deployments.findMany({
        where: { workspaceId: ctx.workspace.id, projectId: input.projectId },
        columns: {
          id: true,
          projectId: true,
          environmentId: true,
          gitCommitSha: true,
          gitBranch: true,
          gitCommitMessage: true,
          gitCommitAuthorHandle: true,
          gitCommitAuthorAvatarUrl: true,
          gitCommitTimestamp: true,
          prNumber: true,
          forkRepositoryFullName: true,
          status: true,
          cpuMillicores: true,
          memoryMib: true,
          createdAt: true,
        },
        with: {
          openapiSpec: {
            columns: {
              pk: true,
            },
          },
          instances: {
            columns: {
              id: true,
            },
            with: {
              region: {
                columns: {
                  id: true,
                  name: true,
                  platform: true,
                },
              },
            },
          },
        },
        orderBy: { createdAt: "desc" },
        limit: 500,
      });

      return deployments.map(({ openapiSpec, ...deployment }) => ({
        ...deployment,
        instances: deployment.instances.map((i) => ({
          ...i,
          flagCode: mapRegionToFlag(i.region.name),
        })),
        gitBranch: deployment.gitBranch ?? "",
        prNumber: deployment.prNumber ?? null,
        forkRepositoryFullName: deployment.forkRepositoryFullName ?? null,
        gitCommitAuthorAvatarUrl:
          deployment.gitCommitAuthorAvatarUrl ?? "https://github.com/identicons/dummy-user.png",
        hasOpenApiSpec: Boolean(openapiSpec),
        gitCommitTimestamp: deployment.gitCommitTimestamp,
      }));
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments",
      });
    }
  });
