import { apiItemsWithKeyCounts } from "@/app/(app)/apis/actions";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { z } from "zod";

export const overviewApiSearch = rateLimitedProcedure(ratelimit.read)
  .input(z.object({ query: z.string() }))
  .mutation(async ({ ctx, input }) => {
    const apis = await db.query.apis.findMany({
      where: (table, { isNull, and, like, eq, or }) =>
        and(
          eq(table.workspaceId, ctx.workspace.id),
          or(like(table.name, `%${input.query}%`), like(table.id, `%${input.query}%`)),
          isNull(table.deletedAtM),
        ),
    });

    const apiList = apiItemsWithKeyCounts(apis);
    return apiList;
  });
