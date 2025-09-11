import { db } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";

import { TRPCError } from "@trpc/server";

export const listRatelimitNamespaces = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    try {
      return await db.query.ratelimitNamespaces.findMany({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.workspaceId, ctx.workspace.id), isNull(table.deletedAtM)),
        columns: {
          id: true,
          name: true,
        },
      });
    } catch (error) {
      console.error(
        "Something went wrong when fetching ratelimit namespaces",
        JSON.stringify(error),
      );
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch ratelimit namespaces",
      });
    }
  });
