import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const listDeployments = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .query(async ({ ctx }) => {
    try {
      // Get all deployments for this workspace with project info
      return await db.query.deployments.findMany({
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
          createdAt: true
        }
      });

    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments",
      });
    }
  });
