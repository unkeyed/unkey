import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const getById = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      versionId: z.string(),
    }),
  )
  .query(async ({ input, ctx }) => {
    try {
      // Get version with branch information
      const version = await db.query.versions.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.versionId), eq(table.workspaceId, ctx.workspace.id)),
      });

      if (!version) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Version not found",
        });
      }

      // Get the branch for this version
      const branch = await db.query.branches.findFirst({
        where: (table, { eq }) => eq(table.id, version.branchId),
      });

      return {
        id: version.id,
        status: version.status,
        gitCommitSha: version.gitCommitSha,
        gitBranch: version.gitBranch,
        branchId: version.branchId,
        createdAt: version.createdAt,
        updatedAt: version.updatedAt,
        branch: branch ? {
          id: branch.id,
          name: branch.name,
        } : null,
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch version",
      });
    }
  });