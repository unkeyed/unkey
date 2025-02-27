import { apiItemsWithKeyCounts } from "@/app/(app)/apis/actions";
import { db, sql } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { z } from "zod";

export const overviewApiSearch = rateLimitedProcedure(ratelimit.read)
  .input(
    z.object({
      query: z.string().trim().max(100),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const apis = await db.query.apis.findMany({
      where: (table, { isNull, and, eq, or }) =>
        and(
          eq(table.workspaceId, ctx.workspace.id),
          or(
            sql`${table.name} LIKE ${`%${input.query}%`}`,
            sql`${table.id} LIKE ${`%${input.query}%`}`,
          ),
          isNull(table.deletedAtM),
        ),
    });

    const apiList = apiItemsWithKeyCounts(apis);
    return apiList;
  });
