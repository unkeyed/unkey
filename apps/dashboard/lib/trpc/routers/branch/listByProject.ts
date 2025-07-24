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

      // Get all branches for this project
      const branches = await db.query.branches.findMany({
        where: (table, { eq }) => eq(table.projectId, input.projectId),
        orderBy: (table, { desc }) => [desc(table.createdAt)],
      });

      return {
        branches: branches.map((branch) => ({
          id: branch.id,
          name: branch.name,
          projectId: branch.projectId,
          isProduction: branch.name === 'main' || branch.name === 'production', // Simple heuristic
          createdAt: branch.createdAt,
          updatedAt: branch.updatedAt,
        })),
      };
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch branches",
      });
    }
  });