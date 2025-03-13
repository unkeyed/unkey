import { keysOverviewQueryTimeseriesPayload } from "@/app/(app)/apis/[apiId]/_overview/components/charts/bar-chart/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { queryApiKeys } from "../api-query";
import { transformVerificationFilters } from "../timeseries.utils";

export const activeKeysTimeseries = rateLimitedProcedure(ratelimit.read)
  .input(keysOverviewQueryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    const { params: transformedInputs, granularity } = transformVerificationFilters(input);

    const { keyspaceId, keyIds } = await queryApiKeys({
      apiId: input.apiId,
      workspaceId: ctx.workspace.id,
      keyIds: transformedInputs.keyIds,
      names: transformedInputs.names,
      identities: transformedInputs.identities,
    });

    const result = await clickhouse.verifications.activeKeysTimeseries[granularity]({
      ...transformedInputs,
      workspaceId: ctx.workspace.id,
      keyspaceId: keyspaceId,
      keyIds: (keyIds ?? []).map((x) => ({
        value: String(x.value),
        operator: x.operator as "is" | "contains",
      })),
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve active keys timeseries analytics due to an error. If this issue persists, please contact support@unkey.dev with the time this occurred.",
      });
    }

    return { timeseries: result.val, granularity };
  });
