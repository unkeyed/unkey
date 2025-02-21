import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { z } from "zod";

export const searchNamespace = rateLimitedProcedure(ratelimit.update)
  .input(z.object({ query: z.string() }))
  .mutation(async ({ ctx, input }) => {
    return await db.query.ratelimitNamespaces.findMany({
      where: (table, { isNull, and, like, eq }) =>
        and(
          eq(table.workspaceId, ctx.workspace.id),
          like(table.name, `%${input.query}%`),
          isNull(table.deletedAt),
        ),
      columns: {
        id: true,
        name: true,
      },
    });
  });
