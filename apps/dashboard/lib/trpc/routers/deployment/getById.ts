import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const getById = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      deploymentId: z.string(),
    }),
  )
  .query(async ({ input, ctx }) => {
    try {
      // Get deployment with branch information
      const deployment = await db.query.versions.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
      });

      if (!deployment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Deployment not found",
        });
      }

      // Get the branch for this deployment
      const branch = deployment.branchId ? await db.query.branches.findFirst({
        where: (table, { eq }) => eq(table.id, deployment.branchId!), // eslint-disable-line @typescript-eslint/no-non-null-assertion
      }) : null;

      return {
        id: deployment.id,
        status: deployment.status,
        gitCommitSha: deployment.gitCommitSha,
        gitBranch: deployment.gitBranch,
        branchId: deployment.branchId,
        createdAt: deployment.createdAt,
        updatedAt: deployment.updatedAt,
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
        message: "Failed to fetch deployment",
      });
    }
  });