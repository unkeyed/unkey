import { clickhouse } from "@/lib/clickhouse";
import {
  type LogsResponseSchema,
  logsRequestSchema,
  logsResponseSchema,
} from "@/lib/schemas/logs.schema";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { transformFilters } from "./utils";

export const queryLogs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(logsRequestSchema)
  .output(logsResponseSchema)
  .query(async ({ ctx, input }) => {
    const transformedInputs = transformFilters(input);
    const { logsQuery, totalQuery } = await clickhouse.api.logs({
      ...transformedInputs,
      cursorTime: input.cursor ?? null,
      workspaceId: ctx.workspace.id,
    });

    // Skip the expensive total count query when looking up a single log by requestId.
    // The count is unnecessary for detail fetches and avoids an extra ClickHouse round-trip
    // per hover/click in the key-details and identity-details log tables.
    const isSingleRequestIdLookup =
      (transformedInputs.requestIds?.length ?? 0) === 1 && input.limit === 1;

    const [countResult, logsResult] = await Promise.all([
      isSingleRequestIdLookup
        ? Promise.resolve({ val: [{ total_count: 0 }], err: null })
        : totalQuery,
      logsQuery,
    ]);

    if (countResult.err || logsResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const logs = logsResult.val;

    // Prepare the response with pagination info
    const response: LogsResponseSchema = {
      logs,
      hasMore: logs.length === input.limit,
      total: isSingleRequestIdLookup ? logs.length : countResult.val[0].total_count,
      nextCursor: logs.length > 0 ? logs[logs.length - 1].time : undefined,
    };

    return response;
  });
