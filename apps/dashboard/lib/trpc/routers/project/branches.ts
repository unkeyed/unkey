import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const listProjectBranches = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      projectId: z.string(),
    }),
  )
  .query(async ({ ctx, input }) => {
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

      // Get all branches for this project
      const branches = await db.query.branches.findMany({
        where: (table, { eq }) => eq(table.projectId, input.projectId),
        orderBy: (table, { desc }) => [desc(table.createdAt)],
      });

      return {
        project: {
          id: project.id,
          name: project.name,
          slug: project.slug,
        },
        branches: branches.map((branch) => ({
          id: branch.id,
          name: branch.name,
          createdAt: branch.createdAt,
          updatedAt: branch.updatedAt,
        })),
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch branches",
      });
    }
  });
