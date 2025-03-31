import { db, sql } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";

export const searchNamespace = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ query: z.string() }))
  .mutation(async ({ ctx, input }) => {
    return await db.query.ratelimitNamespaces.findMany({
      where: (table, { isNull, and, eq }) =>
        and(
          eq(table.workspaceId, ctx.workspace.id),
          sql`${table.name} LIKE ${`%${input.query}%`}`,
          isNull(table.deletedAtM),
        ),
      columns: {
        id: true,
        name: true,
      },
    });
  });
