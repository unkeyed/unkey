import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const queryDistinctPaths = rateLimitedProcedure(ratelimit.update)
  .input(z.object({ currentDate: z.number() }))
  .query(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve distinct paths due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    const fromDate = input.currentDate - 12 * 60 * 60 * 1000; // 12 hours ago
    const result = await clickhouse.querier.query({
      query: `
         SELECT DISTINCT path 
         FROM metrics.raw_api_requests_v1 
         WHERE workspace_id = {workspaceId: String}
         AND time >= {fromDate: UInt64}`,
      schema: z.object({ path: z.string() }),
      params: z.object({
        workspaceId: z.string(),
        fromDate: z.number(),
      }),
    })({
      workspaceId: workspace.id,
      fromDate,
    });

    return result.val?.map((i) => i.path) ?? [];
  });
