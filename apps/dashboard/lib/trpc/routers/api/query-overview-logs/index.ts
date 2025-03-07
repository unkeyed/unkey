import { keysQueryOverviewLogsPayload } from "@/app/(app)/apis/[apiId]/_overview/components/table/query-logs.schema";
import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { keysOverviewLogs as keysLogs } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";
import { transformKeysFilters } from "./utils";

const KeysOverviewLogsResponse = z.object({
  keysOverviewLogs: z.array(keysLogs),
  hasMore: z.boolean(),
  nextCursor: z
    .object({
      time: z.number().int(),
      requestId: z.string(),
    })
    .optional(),
});

type KeysOverviewLogsResponse = z.infer<typeof KeysOverviewLogsResponse>;

export const queryKeysOverviewLogs = rateLimitedProcedure(ratelimit.read)
  .input(keysQueryOverviewLogsPayload)
  .output(KeysOverviewLogsResponse)
  .query(async ({ ctx, input }) => {
    const transformedInputs = transformKeysFilters(input);
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

    const keyDetailsMap = new Map();
    const keyIds = [];
    for (const key of keysResult) {
      keyIds.push(key.id);
      const keyDetails = {
        id: key.id,
        key_auth_id: key.keyAuthId,
        name: key.name,
        owner_id: key.ownerId,
        identity_id: key.identityId,
        meta: key.meta,
        enabled: key.enabled,
        remaining_requests: key.remaining,
        ratelimit_async: key.ratelimitAsync,
        ratelimit_limit: key.ratelimitLimit,
        ratelimit_duration: key.ratelimitDuration,
        environment: key.environment,
        refill_day: key.refillDay,
        refill_amount: key.refillAmount,
        last_refill_at: key.lastRefillAt,
        expires: key.expires,
        workspace_id: key.workspaceId,
      };

      keyDetailsMap.set(key.id, keyDetails);
    }

    // If we already filtered by keyIds in the database query,
    // use those same keyIds for the clickhouse query
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

    const result = await clickhouse.api.keys.logs({
      ...transformedInputs,
      cursorRequestId: input.cursor?.requestId ?? null,
      cursorTime: input.cursor?.time ?? null,
      workspaceId: ctx.workspace.id,
      keyspaceId: combinedResults.keyAuth.id,
      keyIds: keyIdsFilter,
    });

    if (!result || result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from clickhouse.",
      });
    }

    const logs = result.val || [];

    if (logs.length === 0) {
      return {
        keysOverviewLogs: [],
        hasMore: false,
      };
    }

    // Join the keys data with the logs
    const keysOverviewLogs = logs.map((log) => ({
      ...log,
      key_details: keyDetailsMap.get(log.key_id) || null,
    }));

    const response: KeysOverviewLogsResponse = {
      keysOverviewLogs,
      hasMore: logs.length === input.limit,
      nextCursor:
        logs.length === input.limit
          ? {
              time: logs[logs.length - 1].time,
              requestId: logs[logs.length - 1].request_id,
            }
          : undefined,
    };

    return response;
  });
