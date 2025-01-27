import { getTimeseriesGranularity } from "@/app/(app)/logs/utils";
import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { logsTimeseriesParams } from "@unkey/clickhouse/src/logs";

export const queryTimeseries = rateLimitedProcedure(ratelimit.update)
  .input(logsTimeseriesParams.omit({ workspaceId: true }))
  .query(async ({ ctx, input }) => {
    const { startTime, endTime, granularity } = getTimeseriesGranularity(
      input.startTime,
      input.endTime,
    );

    const result = await clickhouse.api.timeseries[granularity]({
      ...input,
      startTime,
      endTime,
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
