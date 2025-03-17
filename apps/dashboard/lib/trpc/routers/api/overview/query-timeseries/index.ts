import { verificationQueryTimeseriesPayload } from "@/app/(app)/apis/_components/hooks/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { transformVerificationFilters } from "./utils";

export const queryVerificationTimeseries = rateLimitedProcedure(ratelimit.read)
  .input(verificationQueryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    const { params: transformedInputs, granularity } = transformVerificationFilters(input);

    const result = await clickhouse.verifications.timeseries[granularity]({
      ...transformedInputs,
      workspaceId: ctx.workspace.id,
      keyspaceId: input.keyspaceId,
    });

    return {
      timeseries: result,
      granularity,
    };
  });
