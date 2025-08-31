import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const listDeployments = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .query(async ({ ctx }) => {
    try {
      // Get all deployments for this workspace with project info
      const deployments = await db.query.deployments.findMany({
        where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
        orderBy: (table, { desc }) => [desc(table.createdAt)],
        with: {
          environment: { columns: { slug: true } },
          project: { columns: { id: true, name: true, slug: true } },
        },
      });

      return {
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
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch deployments",
      });
    }
  });
