import { keysOverviewQueryTimeseriesPayload } from "@/app/(app)/apis/[apiId]/_overview/components/charts/bar-chart/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { getApi, queryApiKeys } from "../api-query";
import { transformVerificationFilters } from "../timeseries.utils";

export const activeKeysTimeseries = rateLimitedProcedure(ratelimit.read)
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

    const clickhouseResult = await clickhouse.verifications.activeKeysTimeseries[granularity]({
      ...transformedInputs,
      workspaceId: ctx.workspace.id,
      keyspaceId: keyspaceId,
      keyIds: input.keyIds ? transformedInputs.keyIds : null,
    });

    if (!clickhouseResult || clickhouseResult.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from ClickHouse.",
      });
    }

    const timeseriesWithKeys = clickhouseResult.val || [];
    if (timeseriesWithKeys.length === 0) {
      return {
        timeseries: null,
        granularity,
      };
    }

    if (input.names?.filters?.length || input.identities?.filters?.length) {
      const allKeyIds = new Set();
      timeseriesWithKeys.forEach((point) => {
        (point.key_ids ?? []).forEach((id) => allKeyIds.add(id));
      });

      const { keys } = await queryApiKeys({
        apiId: input.apiId,
        workspaceId: ctx.workspace.id,
        keyIds: Array.from(allKeyIds).map((id) => ({
          operator: "is",
          value: id as string,
        })),
        names: input.names?.filters || null,
        identities: input.identities?.filters || null,
      });

      const filteredKeyIdSet = new Set(keys.map((key) => key.id));

      const filteredTimeseries = timeseriesWithKeys.map((point) => {
        const filteredKeys = (point.key_ids ?? []).filter((id) => filteredKeyIdSet.has(id));

        return {
          x: point.x,
          y: {
            keys: filteredKeys.length,
          },
        };
      });

      return { timeseries: filteredTimeseries, granularity };
    }

    const timeseriesData = timeseriesWithKeys.map((point) => ({
      x: point.x,
      y: point.y,
    }));

    return { timeseries: timeseriesData, granularity };
  });
