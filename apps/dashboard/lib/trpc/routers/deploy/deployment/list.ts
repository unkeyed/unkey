import { db } from "@/lib/db";
import { requireUser, requireWorkspace, t } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";

export const listDeployments = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .query(async ({ ctx }) => {
    try {
      // Get all deployments for this workspace with project info
      const deployments = await db.query.deployments.findMany({
        where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
        columns: {
          id: true,
          projectId: true,
          environmentId: true,
          gitCommitSha: true,
          gitBranch: true,
          gitCommitMessage: true,
          gitCommitAuthorName: true,
          gitCommitAuthorEmail: true,
          gitCommitAuthorUsername: true,
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
        // Replace NULL git fields with dummy data that clearly indicates it's fake
        gitCommitSha: deployment.gitCommitSha ?? "abc123ef456789012345678901234567890abcdef",
        gitBranch: deployment.gitBranch ?? "main",
        gitCommitMessage: deployment.gitCommitMessage ?? "[DUMMY] Initial commit",
        gitCommitAuthorName: deployment.gitCommitAuthorName ?? "[DUMMY] Unknown Author",
        gitCommitAuthorEmail: deployment.gitCommitAuthorEmail ?? "dummy@example.com",
        gitCommitAuthorUsername: deployment.gitCommitAuthorUsername ?? "dummy-user",
        gitCommitAuthorAvatarUrl:
          deployment.gitCommitAuthorAvatarUrl ?? "https://github.com/identicons/dummy-user.png",
        gitCommitTimestamp: deployment.gitCommitTimestamp ?? Date.now() - 86400000,
      }));
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments",
      });
    }
  });
