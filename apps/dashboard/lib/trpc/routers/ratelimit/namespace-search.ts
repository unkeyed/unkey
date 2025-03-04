import { db, sql } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { z } from "zod";

export const searchNamespace = rateLimitedProcedure(ratelimit.update)
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
