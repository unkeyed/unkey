import { queryTimeseriesPayload } from "@/app/(app)/logs/components/charts/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { transformFilters } from "./utils";

export const queryTimeseries = rateLimitedProcedure(ratelimit.read)
  .input(queryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    const { params: transformedInputs, granularity } = transformFilters(input);
    const result = await clickhouse.api.timeseries[granularity]({
      ...transformedInputs,
      workspaceId: ctx.workspace.id,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve timeseries analytics due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
      });
    }
    return { timeseries: result.val, granularity };
  });
