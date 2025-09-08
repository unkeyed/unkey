import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const listByEnvironment = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      projectId: z.string(),
      environmentId: z.string(),
    }),
  )
  .query(async ({ input, ctx }) => {
    try {
      // First verify the project exists and belongs to this workspace
      const project = await db.query.projects.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
      });

      if (!project) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Project not found",
        });
      }

      // Get all deployments for this project and environment
      const deployments = await db.query.deployments.findMany({
        where: (table, { eq, and }) =>
          and(eq(table.projectId, input.projectId), eq(table.environmentId, input.environmentId)),
        orderBy: (table, { desc }) => [desc(table.createdAt)],
      });

      return {
        deployments: deployments.map((deployment) => ({
          id: deployment.id,
          status: deployment.status,
          gitCommitSha: deployment.gitCommitSha,
          gitBranch: deployment.gitBranch,
          createdAt: deployment.createdAt,
          updatedAt: deployment.updatedAt,
        })),
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments for environment",
      });
    }
  });
