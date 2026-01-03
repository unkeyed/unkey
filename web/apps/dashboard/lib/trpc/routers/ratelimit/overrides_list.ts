import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

import { TRPCError } from "@trpc/server";

export const listRatelimitOverrides = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }) => {
    try {
      return await db.query.ratelimitOverrides.findMany({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.workspaceId, ctx.workspace.id), isNull(table.deletedAtM)),
        columns: {
          id: true,
          identifier: true,
          limit: true,
          duration: true,
          namespaceId: true,
        },
      });
    } catch (error) {
      console.error(
        "Something went wrong when fetching ratelimit overrides",
        JSON.stringify(error),
      );
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch ratelimit overrides",
      });
    }
  });
