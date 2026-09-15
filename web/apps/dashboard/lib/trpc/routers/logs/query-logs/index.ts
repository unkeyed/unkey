import { clickhouse } from "@/lib/clickhouse";
import {
  type LogsResponseSchema,
  logsRequestSchema,
  logsResponseSchema,
} from "@/lib/schemas/logs.schema";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import type { RequestLogsResponse } from "@unkey/clickhouse/src/frontline";
import type { Log } from "@unkey/clickhouse/src/logs";
import { transformFilters } from "./utils";

function toApiLog(frontlineLog: RequestLogsResponse, workspaceId: string): Log {
  return {
    request_id: frontlineLog.request_id,
    time: frontlineLog.time,
    workspace_id: workspaceId,
    host: frontlineLog.host,
    method: frontlineLog.method,
    path: frontlineLog.path,
    request_headers: frontlineLog.request_headers,
    request_body: frontlineLog.request_body,
    response_status: frontlineLog.response_status,
    response_headers: frontlineLog.response_headers,
    response_body: frontlineLog.response_body,
    error: "",
    service_latency: frontlineLog.total_latency,
  };
}

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

    let logs = logsResult.val;

    if (logs.length === 0 && isSingleRequestIdLookup) {
      const requestId = transformedInputs.requestIds?.[0];
      if (requestId) {
        const frontlineResult = await clickhouse.frontline.requestById({
          workspaceId: ctx.workspace.id,
          requestId,
        });
        if (!frontlineResult.err && frontlineResult.val.length > 0) {
          logs = [toApiLog(frontlineResult.val[0], ctx.workspace.id)];
        }
      }
    }

    // Prepare the response with pagination info
    const response: LogsResponseSchema = {
      logs,
      hasMore: logs.length === input.limit,
      total: isSingleRequestIdLookup ? logs.length : countResult.val[0].total_count,
      nextCursor: logs.length > 0 ? logs[logs.length - 1].time : undefined,
    };

    return response;
  });
