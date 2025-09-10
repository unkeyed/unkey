import {
  db
} from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";




export const listDomains = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {

    return await db.query.domains.findMany({
      where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
      columns: {
        id: true,
        domain: true,
        projectId: true,
        type: true,
      }
    }).catch(error => {
      console.error("Error querying domains:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve domains due to an error. If this issue persists, please contact support.",
      });
    });
  });
