import { ratelimitQueryLogsPayload } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/logs/components/table/query-logs.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { ratelimitLogs } from "@unkey/clickhouse/src/ratelimits";
import { z } from "zod";
import { transformFilters } from "./utils";

const RatelimitLogsResponse = z.object({
  ratelimitLogs: z.array(ratelimitLogs),
  hasMore: z.boolean(),
  total: z.number(),
  nextCursor: z.number().int().optional(),
});

type RatelimitLogsResponse = z.infer<typeof RatelimitLogsResponse>;

export const queryRatelimitLogs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(ratelimitQueryLogsPayload)
  .output(RatelimitLogsResponse)
  .query(async ({ ctx, input }) => {
    const ratelimitNamespaces = await db.query.ratelimitNamespaces
      .findMany({
        where: (table, { and, eq, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            and(eq(table.id, input.namespaceId), isNull(table.deletedAtM)),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve ratelimit timeseries analytics due to a workspace error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!ratelimitNamespaces) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Ratelimit namespaces not found, please contact support using support@unkey.dev.",
      });
    }

    if (ratelimitNamespaces.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Namespace not found",
      });
    }

    const transformedInputs = transformFilters(input);
    const { countQuery, logsQuery } = await clickhouse.ratelimits.logs({
      ...transformedInputs,
      cursorTime: input.cursor ?? null,
      workspaceId: ctx.workspace.id,
      namespaceId: ratelimitNamespaces[0].id,
    });

    const [countResult, logsResult] = await Promise.all([countQuery, logsQuery]);

    if (countResult.err || logsResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const logs = logsResult.val;
    const response: RatelimitLogsResponse = {
      ratelimitLogs: logs,
      total: countResult.val[0].total_count,
      hasMore: logs.length === input.limit,
      nextCursor: logs.length > 0 ? logs[logs.length - 1].time : undefined,
    };

    return response;
  });
