import { db } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";

export const listProjects = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    return await db.query.projects
      .findMany({
        where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
        columns: {
          id: true,
          name: true,
          slug: true,
          updatedAt: true,
          gitRepositoryUrl: true,
          activeDeploymentId: true,
        },
      })
      .catch((error) => {
        console.error("Error querying projects:", error);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve projects due to an error. If this issue persists, please contact support.",
        });
      });
  });
