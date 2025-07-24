import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const listByBranch = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      branchId: z.string(),
    }),
  )
  .query(async ({ input, ctx }) => {
    try {
      // First verify the branch exists and belongs to this workspace
      const branch = await db.query.branches.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.branchId), eq(table.workspaceId, ctx.workspace.id)),
      });

      if (!branch) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Branch not found",
        });
      }

      // Get all versions for this branch
      const versions = await db.query.versions.findMany({
        where: (table, { eq }) => eq(table.branchId, input.branchId),
        orderBy: (table, { desc }) => [desc(table.createdAt)],
      });

      return {
        versions: versions.map((version) => ({
          id: version.id,
          status: version.status,
          gitCommitSha: version.gitCommitSha,
          gitBranch: version.gitBranch,
          createdAt: version.createdAt,
          updatedAt: version.updatedAt,
        })),
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch versions for branch",
      });
    }
  });