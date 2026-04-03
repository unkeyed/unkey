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
      where: {
        workspaceId: ctx.workspace.id,
        OR: [
          { RAW: (table, _ops) => sql`${table.name} LIKE ${`%${input.query}%`}` },
          { RAW: (table, _ops) => sql`${table.id} LIKE ${`%${input.query}%`}` },
        ],
        deletedAtM: { isNull: true },
      },
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
