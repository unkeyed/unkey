import { keysOverviewQueryTimeseriesPayload } from "@/app/(app)/apis/[apiId]/_overview/components/charts/bar-chart/query-timeseries.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { transformVerificationFilters } from "./utils";

export const keyVerificationsTimeseries = rateLimitedProcedure(ratelimit.read)
  .input(keysOverviewQueryTimeseriesPayload)
  .query(async ({ ctx, input }) => {
    const { params: transformedInputs, granularity } = transformVerificationFilters(input);
    const keyIdsFromInput = transformedInputs.keyIds;
    const namesFromInput = transformedInputs.names;

    const combinedResults = await db.query.apis
      .findFirst({
        where: (api, { and, eq, isNull }) =>
          and(
            eq(api.id, input.apiId),
            eq(api.workspaceId, ctx.workspace.id),
            isNull(api.deletedAtM),
          ),
        with: {
          keyAuth: {
            with: {
              keys: {
                where: (key, { and, isNull, inArray, sql }) => {
                  const conditions = [isNull(key.deletedAtM)];

                  // Add name filters
                  if (namesFromInput && namesFromInput.length > 0) {
                    const nameValues = namesFromInput
                      .filter((filter) => filter.operator === "is")
                      .map((filter) => filter.value);

                    if (nameValues.length > 0) {
                      conditions.push(inArray(key.name, nameValues));
                    }

                    const nameContainsValues = namesFromInput
                      .filter((filter) => filter.operator === "contains")
                      .map((filter) => filter.value);
                    if (nameContainsValues.length > 0) {
                      nameContainsValues.forEach((value) => {
                        conditions.push(sql`${key.name} LIKE ${`%${value}%`}`);
                      });
                    }
                  }

                  // Add keyId filters
                  if (keyIdsFromInput && keyIdsFromInput.length > 0) {
                    const keyIdValues = keyIdsFromInput
                      .filter((filter) => filter.operator === "is")
                      .map((filter) => filter.value);

                    if (keyIdValues.length > 0) {
                      conditions.push(inArray(key.id, keyIdValues));
                    }

                    const keyIdContainsValues = keyIdsFromInput
                      .filter((filter) => filter.operator === "contains")
                      .map((filter) => filter.value);
                    if (keyIdContainsValues.length > 0) {
                      keyIdContainsValues.forEach((value) =>
                        conditions.push(sql`${key.id} LIKE ${`%${value}%`}`),
                      );
                    }
                  }

                  return and(...conditions);
                },
                columns: {
                  id: true,
                  keyAuthId: true,
                  name: true,
                  ownerId: true,
                  identityId: true,
                  meta: true,
                  enabled: true,
                  remaining: true,
                  ratelimitAsync: true,
                  ratelimitLimit: true,
                  ratelimitDuration: true,
                  environment: true,
                  refillDay: true,
                  refillAmount: true,
                  lastRefillAt: true,
                  expires: true,
                  workspaceId: true,
                },
              },
            },
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve API information. If this issue persists, please contact support@unkey.dev with the time this occurred.",
        });
      });

    if (!combinedResults || !combinedResults?.keyAuth?.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "API not found or does not have key authentication enabled",
      });
    }

    const keysResult = combinedResults.keyAuth.keys;
    const keyIds = [];
    for (const key of keysResult) {
      keyIds.push(key.id);
    }

    let keyIdsFilter = keyIdsFromInput;
    if (!keyIdsFilter || keyIdsFilter.length === 0) {
      if (keyIds.length > 0) {
        // Build a filter using OR conditions with "is" operators
        keyIdsFilter = keyIds.map((keyId) => ({
          operator: "is" as const,
          value: keyId,
        }));
      }
    }

    const result = await clickhouse.verifications.timeseries[granularity]({
      ...transformedInputs,
      workspaceId: ctx.workspace.id,
      keyspaceId: combinedResults.keyAuth.id,
      keyIds: keyIdsFilter,
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
