import { queryLogsPayload } from "@/app/(app)/logs-v2/components/table/query-logs.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { type GetLogsClickhousePayload, log } from "@unkey/clickhouse/src/logs";
import { z } from "zod";

const LogsResponse = z.object({
  logs: z.array(log),
  hasMore: z.boolean(),
  nextCursor: z
    .object({
      time: z.number().int(),
      requestId: z.string(),
    })
    .optional(),
});

type LogsResponse = z.infer<typeof LogsResponse>;

const getTimestampFromRelative = (relativeTime: string): number => {
  let totalMilliseconds = 0;

  for (const [, amount, unit] of relativeTime.matchAll(/(\d+)([hdm])/g)) {
    const value = Number.parseInt(amount, 10);

    switch (unit) {
      case "h":
        totalMilliseconds += value * 60 * 60 * 1000;
        break;
      case "d":
        totalMilliseconds += value * 24 * 60 * 60 * 1000;
        break;
      case "m":
        totalMilliseconds += value * 60 * 1000;
        break;
    }
  }

  return Date.now() - totalMilliseconds;
};
export function transformFilters(
  params: z.infer<typeof queryLogsPayload>,
): Omit<GetLogsClickhousePayload, "workspaceId"> {
  // Transform path filters to include operators
  const paths =
    params.path?.filters.map((f) => ({
      operator: f.operator,
      value: f.value,
    })) || [];

  // Extract other filters as before
  const requestIds = params.requestId?.filters.map((f) => f.value) || [];
  const hosts = params.host?.filters.map((f) => f.value) || [];
  const methods = params.method?.filters.map((f) => f.value) || [];
  const statusCodes = params.status?.filters.map((f) => f.value) || [];

  let startTime = params.startTime;
  let endTime = params.endTime;

  // If we have relativeTime filter `since`, ignore other time params
  const hasRelativeTime = params.since !== "";
  if (hasRelativeTime) {
    startTime = getTimestampFromRelative(params.since);
    endTime = Date.now();
  }

  return {
    limit: params.limit,
    startTime,
    endTime,
    requestIds,
    hosts,
    methods,
    paths,
    statusCodes,
    cursorTime: params.cursor?.time ?? null,
    cursorRequestId: params.cursor?.requestId ?? null,
  };
}

export const queryLogs = rateLimitedProcedure(ratelimit.update)
  .input(queryLogsPayload)
  .output(LogsResponse)
  .query(async ({ ctx, input }) => {
    // Get workspace
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve workspace logs due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    const transformedInputs = transformFilters(input);
    const result = await clickhouse.api.logs({
      ...transformedInputs,
      cursorRequestId: input.cursor?.requestId ?? null,
      cursorTime: input.cursor?.time ?? null,
      workspaceId: workspace.id,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const logs = result.val;

    // Prepare the response with pagination info
    const response: LogsResponse = {
      logs,
      hasMore: logs.length === input.limit,
      nextCursor:
        logs.length > 0
          ? {
              time: logs[logs.length - 1].time,
              requestId: logs[logs.length - 1].request_id,
            }
          : undefined,
    };

    return response;
  });
