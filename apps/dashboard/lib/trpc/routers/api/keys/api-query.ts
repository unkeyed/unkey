import type { KeysOverviewFilterUrlValue } from "@/app/(app)/apis/[apiId]/_overview/filters.schema";
import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";

// Input interface for the query abstraction
export interface QueryApiKeysInput {
  apiId: string;
  workspaceId: string;
  keyIds: KeysOverviewFilterUrlValue[] | null;
  names?: KeysOverviewFilterUrlValue[] | null;
}

// Response interface with the query results
export interface QueryApiKeysResult {
  keyspaceId: string;
  keys: any[];
  keyIds: KeysOverviewFilterUrlValue[] | null;
}

/**
 * Abstracts the common database query pattern for API keys
 *
 * @param input Query parameters including apiId, workspaceId, and optional filters
 * @returns The API's keyspace ID, matching keys, and processed keyIds filter
 */
export async function queryApiKeys({
  apiId,
  workspaceId,
  keyIds: keyIdsFromInput,
  names: namesFromInput,
}: QueryApiKeysInput): Promise<QueryApiKeysResult> {
  // Query the API and related keys with filters
  const combinedResults = await db.query.apis
    .findFirst({
      where: (api, { and, eq, isNull }) =>
        and(
          eq(api.id, apiId),
          eq(api.workspaceId, workspaceId),
          isNull(api.deletedAtM)
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
                    conditions.push(inArray(key.name, nameValues as string[]));
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
                    conditions.push(inArray(key.id, keyIdValues as string[]));
                  }

                  const keyIdContainsValues = keyIdsFromInput
                    .filter((filter) => filter.operator === "contains")
                    .map((filter) => filter.value);
                  if (keyIdContainsValues.length > 0) {
                    keyIdContainsValues.forEach((value) =>
                      conditions.push(sql`${key.id} LIKE ${`%${value}%`}`)
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
  const keyIds = keysResult.map((key) => key.id);

  // Build keyIdsFilter for clickhouse query
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

  return {
    keyspaceId: combinedResults.keyAuth.id,
    keys: keysResult,
    keyIds: keyIdsFilter,
  };
}

/**
 * Helper function to convert key DB results to a Map for easy lookup
 *
 * @param keys Array of key objects from the database
 * @returns Map of key IDs to formatted key details
 */
export function createKeyDetailsMap(keys: any[]): Map<string, any> {
  const keyDetailsMap = new Map();

  for (const key of keys) {
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

  return keyDetailsMap;
}
