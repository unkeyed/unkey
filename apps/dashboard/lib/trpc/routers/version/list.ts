import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const listVersions = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .query(async ({ ctx }) => {
    try {
      // Get all versions for this workspace with project and branch info
      const versions = await db.query.versions.findMany({
        where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
        orderBy: (table, { desc }) => [desc(table.createdAt)],
        with: {
          project: true,
          branch: true,
        },
      });

      return {
        versions: versions.map((version) => ({
          id: version.id,
          status: version.status,
          gitCommitSha: version.gitCommitSha,
          gitBranch: version.gitBranch,
          rootfsImageId: version.rootfsImageId,
          buildId: version.buildId,
          createdAt: version.createdAt,
          updatedAt: version.updatedAt,
          project: version.project
            ? {
                id: version.project.id,
                name: version.project.name,
                slug: version.project.slug,
              }
            : null,
          branch: version.branch
            ? {
                id: version.branch.id,
                name: version.branch.name,
              }
            : null,
        })),
      };
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch versions",
      });
    }
  });
