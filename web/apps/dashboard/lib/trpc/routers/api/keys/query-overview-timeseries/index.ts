import { keysOverviewQueryTimeseriesPayload } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/components/charts/bar-chart/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { getApi, queryApiKeys } from "../api-query";
import { transformVerificationFilters } from "../timeseries.utils";

export const keyVerificationsTimeseries = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
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

    // keyIds/names/identities live in Postgres (CH has no name/identity
    // columns). Resolve them to a concrete keyId set so CH filters by the
    // intersection. When this resolves to zero rows (e.g. filter by a name
    // that no key has), we must return empty — a `[]` keyIds list makes CH's
    // `if (params.keyIds?.length)` falsy and drops the WHERE clause entirely,
    // which would otherwise leak unfiltered verifications into the chart.
    const hasKeyFilters =
      (transformedInputs.keyIds !== null && transformedInputs.keyIds.length > 0) ||
      (transformedInputs.names !== null && transformedInputs.names.length > 0) ||
      (transformedInputs.identities !== null && transformedInputs.identities.length > 0);

    let keyIdsForClickhouse: { operator: "is" | "contains"; value: string }[] | null = null;

    if (hasKeyFilters) {
      const apiKeysResult = await queryApiKeys({
        apiId: input.apiId,
        workspaceId: ctx.workspace.id,
        keyIds: transformedInputs.keyIds,
        names: transformedInputs.names,
        identities: transformedInputs.identities,
      });

      if (apiKeysResult.keys.length === 0) {
        return {
          timeseries: [],
          granularity,
        };
      }

      keyIdsForClickhouse = apiKeysResult.keys.map((k) => ({
        operator: "is" as const,
        value: k.id,
      }));
    }

    const result = await clickhouse.verifications.timeseries[granularity]({
      ...transformedInputs,
      workspaceId: ctx.workspace.id,
      keyspaceId: keyspaceId,
      keyIds: keyIdsForClickhouse,
    });

    return {
      timeseries: result,
      granularity,
    };
  });
