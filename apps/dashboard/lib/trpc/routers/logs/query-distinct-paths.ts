import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { z } from "zod";

export const queryDistinctPaths = rateLimitedProcedure(ratelimit.update).query(async ({ ctx }) => {
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
    workspaceId: ctx.workspace.id,
    fromDate: Date.now() - 12 * 60 * 60 * 1000, // 12 hours ag,
  });

  return result.val?.map((i) => i.path) ?? [];
});
