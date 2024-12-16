import { getTimeseriesGranularity } from "@/app/(app)/logs/utils";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { logsTimeseriesParams } from "@unkey/clickhouse/src/logs";

export const queryTimeseries = rateLimitedProcedure(ratelimit.update)
  .input(logsTimeseriesParams.omit({ workspaceId: true }))
  .query(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          //TODO: change error message later
          message:
            "Failed to retrieve timeseries analytics due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.dev.",
      });
    }

    const { startTime, endTime, granularity } = getTimeseriesGranularity(
      input.startTime,
      input.endTime,
    );

    const result = await clickhouse.api.timeseries[granularity]({
      ...input,
      startTime,
      endTime,
      workspaceId: workspace.id,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }
    return result.val;
  });
