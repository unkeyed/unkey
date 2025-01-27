import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { getLogsClickhousePayload } from "@unkey/clickhouse/src/logs";

export const queryLogs = rateLimitedProcedure(ratelimit.update)
  .input(getLogsClickhousePayload.omit({ workspaceId: true }))
  .query(async ({ ctx, input }) => {
    const result = await clickhouse.api.logs({
      ...input,
      workspaceId: ctx.workspace.id,
    });
    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }
    return result.val;
  });
