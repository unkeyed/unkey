import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const listByProject = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      projectId: z.string(),
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

      // Get all deployments for this project
      const deployments = await db.query.deployments.findMany({
        where: (table, { eq }) => eq(table.projectId, input.projectId),
        orderBy: (table, { desc }) => [desc(table.createdAt)],
        with: {
          environment: true,
          project: true,
        },
      });

      return {
        project: {
          id: project.id,
          name: project.name,
          slug: project.slug,
          gitRepositoryUrl: project.gitRepositoryUrl,
          createdAt: project.createdAt,
          updatedAt: project.updatedAt,
        },
        deployments: deployments.map((deployment) => ({
          id: deployment.id,
          status: deployment.status,
          gitCommitSha: deployment.gitCommitSha,
          gitBranch: deployment.gitBranch,
          environment: deployment.environment.slug,
          createdAt: deployment.createdAt,
          updatedAt: deployment.updatedAt,
          project: deployment.project
            ? {
                id: deployment.project.id,
                name: deployment.project.name,
                slug: deployment.project.slug,
              }
            : null,
        })),
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments for project",
      });
    }
  });
