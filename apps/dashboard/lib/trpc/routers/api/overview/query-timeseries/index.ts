import { verificationQueryTimeseriesPayload } from "@/app/(app)/apis/_components/hooks/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
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

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve ratelimit timeseries analytics due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
      });
    }

    return { timeseries: result.val, granularity };
  });
