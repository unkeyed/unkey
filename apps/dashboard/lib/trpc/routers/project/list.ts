import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const listProjects = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .query(async ({ ctx }) => {
    try {
      const projects = await db.query.projects.findMany({
        where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
        orderBy: (table, { desc }) => [desc(table.createdAt)],
      });

      return {
        projects: projects.map((project) => ({
          id: project.id,
          name: project.name,
          slug: project.slug,
          gitRepositoryUrl: project.gitRepositoryUrl,
          createdAt: project.createdAt,
          updatedAt: project.updatedAt,
        })),
      };
    } catch (_error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch projects",
      });
    }
  });
