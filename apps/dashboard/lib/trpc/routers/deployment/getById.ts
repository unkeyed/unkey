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
      // Get deployment information
      const deployment = await db.query.deployments.findFirst({
        with: {
          environment: true,
        },
        where: (table, { eq, and }) =>
          and(eq(table.id, input.deploymentId), eq(table.workspaceId, ctx.workspace.id)),
      });

      if (!deployment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Deployment not found",
        });
      }

      return {
        id: deployment.id,
        status: deployment.status,
        gitCommitSha: deployment.gitCommitSha,
        gitBranch: deployment.gitBranch,
        environment: deployment.environment?.slug ?? "",
        createdAt: deployment.createdAt,
        updatedAt: deployment.updatedAt,
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
