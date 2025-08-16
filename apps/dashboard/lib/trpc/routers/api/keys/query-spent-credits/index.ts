import { keysOverviewQueryTimeseriesPayload } from "@/app/(app)/apis/[apiId]/_overview/components/charts/bar-chart/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { getApi, queryApiKeys } from "../api-query";
import { transformVerificationFilters } from "../timeseries.utils";

export const queryApiSpentCredits = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
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
    const { params: transformedInputs } = transformVerificationFilters(input);

    // Check if we have any key-related filters
    const hasKeyFilters =
      (transformedInputs.keyIds !== null && transformedInputs.keyIds.length > 0) ||
      (transformedInputs.names !== null && transformedInputs.names.length > 0) ||
      (transformedInputs.identities !== null && transformedInputs.identities.length > 0);

    // Only query API keys if we have key-related filters
    if (hasKeyFilters) {
      await queryApiKeys({
        apiId: input.apiId,
        workspaceId: ctx.workspace.id,
        keyIds: transformedInputs.keyIds,
        names: transformedInputs.names,
        identities: transformedInputs.identities,
      });

      // Note: For spent credits total, we don't need to filter by specific keyIds
      // as the aggregation happens at the keyspace level with existing filters
    }

    const result = await clickhouse.verifications.spentCreditsTotal({
      workspaceId: ctx.workspace.id,
      keyspaceId,
      keyId: undefined, // Don't filter by specific key for API-wide totals
      startTime: transformedInputs.startTime,
      endTime: transformedInputs.endTime,
      outcomes: transformedInputs.outcomes || null,
      tags:
        transformedInputs.tags && transformedInputs.tags.length > 0
          ? transformedInputs.tags[0]
          : null,
    });

    return {
      spentCredits: result.val?.[0]?.spent_credits ?? 0,
    };
  });
