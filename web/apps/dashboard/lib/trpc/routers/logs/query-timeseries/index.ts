import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

import { timeseriesRequestSchema } from "@/lib/schemas/logs.schema";
import { TRPCError } from "@trpc/server";
import { transformFilters } from "./utils";

export const queryTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(timeseriesRequestSchema)
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
          "Failed to retrieve timeseries analytics due to an error. If this issue persists, please contact support@unkey.com with the time this occurred.",
      });
    }
    return { timeseries: result.val, granularity };
  });
