import { db } from "@/lib/db";
import { requireUser, requireWorkspace, t } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const listDeployments = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(z.object({ projectId: z.string() }))
  .query(async ({ ctx, input }) => {
    try {
      // Get all deployments for this workspace and specific project
      const deployments = await db.query.deployments.findMany({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, ctx.workspace.id), eq(table.projectId, input.projectId)),
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
          runtimeConfig: true,
          status: true,
          createdAt: true,
        },
        limit: 500,
      });

      return deployments.map((deployment) => ({
        ...deployment,
        gitBranch: deployment.gitBranch ?? "main",
        gitCommitAuthorAvatarUrl:
          deployment.gitCommitAuthorAvatarUrl ?? "https://github.com/identicons/dummy-user.png",
        gitCommitTimestamp: deployment.gitCommitTimestamp,
      }));
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments",
      });
    }
  });
