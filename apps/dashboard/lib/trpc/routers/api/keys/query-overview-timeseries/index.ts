import { keysOverviewQueryTimeseriesPayload } from "@/app/(app)/apis/[apiId]/_overview/components/charts/bar-chart/query-timeseries.schema";
import type { KeysOverviewFilterUrlValue } from "@/app/(app)/apis/[apiId]/_overview/filters.schema";
import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { getApi, queryApiKeys } from "../api-query";
import { transformVerificationFilters } from "../timeseries.utils";

export const keyVerificationsTimeseries = rateLimitedProcedure(ratelimit.read)
  .input(keysOverviewQueryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    const api = await getApi(input.apiId, ctx.workspace.id);
    if (!api || !api.keyAuth?.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "API not found or does not have key authentication enabled",
      });
    }

    const keyspaceId = api.keyAuth.id;
    const { params: transformedInputs, granularity } = transformVerificationFilters(input);

    // Check if we have any key-related filters
    const hasKeyFilters =
      (transformedInputs.keyIds !== null && transformedInputs.keyIds.length > 0) ||
      (transformedInputs.names !== null && transformedInputs.names.length > 0) ||
      (transformedInputs.identities !== null && transformedInputs.identities.length > 0);

    let keyIds: KeysOverviewFilterUrlValue[] | null = [];

    // Only query API keys if we have key-related filters
    if (hasKeyFilters) {
      const apiKeysResult = await queryApiKeys({
        apiId: input.apiId,
        workspaceId: ctx.workspace.id,
        keyIds: transformedInputs.keyIds,
        names: transformedInputs.names,
        identities: transformedInputs.identities,
      });

      keyIds = apiKeysResult.keyIds || [];
    }

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
      timeseries: result,
      granularity,
    };
  });
