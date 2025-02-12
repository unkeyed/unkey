import { ratelimitQueryOverviewLogsPayload } from "@/app/(app)/ratelimits/[namespaceId]/_overview/components/table/query-logs.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { ratelimitOverviewLogs } from "@unkey/clickhouse/src/ratelimits";
import { z } from "zod";
import { transformFilters } from "./utils";

const RatelimitOverviewLogsResponse = z.object({
  ratelimitOverviewLogs: z.array(ratelimitOverviewLogs),
  hasMore: z.boolean(),
  nextCursor: z
    .object({
      time: z.number().int(),
      requestId: z.string(),
    })
    .optional(),
});

type RatelimitOverviewLogsResponse = z.infer<typeof RatelimitOverviewLogsResponse>;

export const queryRatelimitOverviewLogs = rateLimitedProcedure(ratelimit.update)
  .input(ratelimitQueryOverviewLogsPayload)
  .output(RatelimitOverviewLogsResponse)
  .query(async ({ ctx, input }) => {
    // Get workspace
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          ratelimitNamespaces: {
            where: (table, { and, eq, isNull }) =>
              and(eq(table.id, input.namespaceId), isNull(table.deletedAt)),
            columns: {
              id: true,
            },
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve ratelimit logs due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    if (workspace.ratelimitNamespaces.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Namespace not found",
      });
    }

    const transformedInputs = transformFilters(input);
    const result = await clickhouse.ratelimits.overview.logs({
      ...transformedInputs,
      cursorRequestId: input.cursor?.requestId ?? null,
      cursorTime: input.cursor?.time ?? null,
      workspaceId: workspace.id,
      namespaceId: workspace.ratelimitNamespaces[0].id,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const logs = result.val;

    const response: RatelimitOverviewLogsResponse = {
      ratelimitOverviewLogs: logs,
      hasMore: logs.length === input.limit,
      nextCursor:
        logs.length === input.limit
          ? {
              time: logs[logs.length - 1].time,
              requestId: logs[logs.length - 1].request_id,
            }
          : undefined,
    };

    return response;
  });
