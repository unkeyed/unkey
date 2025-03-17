import { keysOverviewQueryTimeseriesPayload } from "@/app/(app)/apis/[apiId]/_overview/components/charts/bar-chart/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import type { VerificationTimeseriesDataPoint } from "@unkey/clickhouse/src/verifications";
import { queryApiKeys } from "../api-query";
import { transformVerificationFilters } from "../timeseries.utils";

export const keyVerificationsTimeseries = rateLimitedProcedure(ratelimit.read)
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

    const result = await clickhouse.verifications.timeseries[granularity]({
      ...transformedInputs,
      workspaceId: ctx.workspace.id,
      keyspaceId: keyspaceId,
      keyIds: (keyIds ?? []).map((x) => ({
        value: String(x.value),
        operator: x.operator as "is" | "contains",
      })),
    });

    return {
      timeseries: result as VerificationTimeseriesDataPoint[],
      granularity,
    };
  });
