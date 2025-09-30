import { db } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const listDomains = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ projectId: z.string() }))
  .query(async ({ ctx, input }) => {
    return await db.query.domains
      .findMany({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, ctx.workspace.id), eq(table.projectId, input.projectId)),
        columns: {
          id: true,
          domain: true,
          projectId: true,
          deploymentId: true,
          type: true,
          sticky: true,
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
  });
