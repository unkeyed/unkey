import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
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
    // Get workspace
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve workspace logs due to an error. If this issue persists, please contact support@unkey.com with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.com.",
      });
    }

    const transformedInputs = transformFilters(input);
    const { logsQuery, totalQuery } = await clickhouse.api.logs({
      ...transformedInputs,
      cursorTime: input.cursor ?? null,
      workspaceId: workspace.id,
    });

    const [countResult, logsResult] = await Promise.all([totalQuery, logsQuery]);

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
      total: countResult.val[0].total_count,
      nextCursor: logs.length > 0 ? logs[logs.length - 1].time : undefined,
    };

    return response;
  });
