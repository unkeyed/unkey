import { apiItemsWithApproxKeyCounts } from "@/app/(app)/[workspaceSlug]/apis/actions";
import { db, sql } from "@/lib/db";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../trpc";

export const overviewApiSearch = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
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
      with: {
        keyAuth: {
          columns: {
            sizeApprox: true,
          },
        },
      },
    });

    const apiList = await apiItemsWithApproxKeyCounts(apis);
    return apiList;
  });
