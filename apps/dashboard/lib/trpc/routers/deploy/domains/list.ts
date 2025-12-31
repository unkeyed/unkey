import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const listDomains = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string() }))
  .query(async ({ ctx, input }) => {
    const project = await db.query.projects
      .findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.projectId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          id: true,
        },
        with: {
          frontlineRoutes: {
            limit: 500,
            orderBy: (table, { desc }) => desc(table.updatedAt),
          },
        },
      })
      .catch((error) => {
        console.error("Error querying domains:", error);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve domains due to an error. If this issue persists, please contact support.",
        });
      });

    return project?.frontlineRoutes ?? [];
  });
