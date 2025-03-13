import type { KeysOverviewFilterUrlValue } from "@/app/(app)/apis/[apiId]/_overview/filters.schema";
import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";

// Input interface for the query abstraction
export interface QueryApiKeysInput {
  apiId: string;
  workspaceId: string;
  keyIds: KeysOverviewFilterUrlValue[] | null;
  names?: KeysOverviewFilterUrlValue[] | null;
  identities?: KeysOverviewFilterUrlValue[] | null;
}

// Response interface with the query results
export interface QueryApiKeysResult {
  keyspaceId: string;
  keys: any[];
  keyIds: KeysOverviewFilterUrlValue[] | null;
}

/**
 * Abstracts the common database query pattern for API keys
 * With proper identity relation joining
 *
 * @param input Query parameters including apiId, workspaceId, and optional filters
 * @returns The API's keyspace ID, matching keys, and processed keyIds filter
 */
export async function queryApiKeys({
  apiId,
  workspaceId,
  keyIds: keyIdsFromInput,
  names: namesFromInput,
  identities: identitiesFromInput,
}: QueryApiKeysInput): Promise<QueryApiKeysResult> {
  const combinedResults = await db.query.apis
    .findFirst({
      where: (api, { and, eq, isNull }) =>
        and(eq(api.id, apiId), eq(api.workspaceId, workspaceId), isNull(api.deletedAtM)),
      with: {
        keyAuth: {
          with: {
            keys: {
              with: {
                permissions: {
                  with: {
                    permission: {
                      columns: {
                        name: true,
                        description: true,
                      },
                    },
                  },
                },
                roles: {
                  with: {
                    role: {
                      columns: {
                        name: true,
                        description: true,
                      },
                    },
                  },
                },
                identity: {
                  columns: {
                    externalId: true,
                  },
                },
              },
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
                      conditions.push(sql`${key.id} LIKE ${`%${value}%`}`),
                    );
                  }
                }

                // Add identity filters
                if (identitiesFromInput && identitiesFromInput.length > 0) {
                  // First, categorize the identity filters by type
                  const externalIdFilters = [];
                  const ownerIdFilters = [];

                  // Parse the identity filter values to determine their type
                  for (const filter of identitiesFromInput) {
                    // Check if the value starts with known prefixes
                    const value = filter.value;
                    if (typeof value === "string") {
                      if (value.startsWith("ext_")) {
                        // This is an external ID filter
                        externalIdFilters.push(filter);
                      } else {
                        // Treat as owner ID filter for other formats
                        ownerIdFilters.push(filter);
                      }
                    }
                  }

                  // Initialize arrays for identity conditions
                  const identityConditions = [];

                  // Process owner ID filters
                  if (ownerIdFilters.length > 0) {
                    const ownerIsValues = ownerIdFilters
                      .filter((filter) => filter.operator === "is")
                      .map((filter) => filter.value);

                    const ownerContainsValues = ownerIdFilters
                      .filter((filter) => filter.operator === "contains")
                      .map((filter) => filter.value);

                    const ownerConditions = [];

                    // Add "is" conditions for ownerId
                    if (ownerIsValues.length > 0) {
                      ownerConditions.push(sql`(${key.ownerId} IN (${sql.join(ownerIsValues)}))`);
                    }

                    // Add "contains" conditions for ownerId
                    if (ownerContainsValues.length > 0) {
                      ownerContainsValues.forEach((value) => {
                        ownerConditions.push(sql`(${key.ownerId} LIKE ${`%${value}%`})`);
                      });
                    }

                    // Combine owner conditions with OR
                    if (ownerConditions.length > 0) {
                      identityConditions.push(sql`(${sql.join(ownerConditions, sql` OR `)})`);
                    }
                  }

                  // Process external ID filters
                  if (externalIdFilters.length > 0) {
                    const externalIdIsValues = externalIdFilters
                      .filter((filter) => filter.operator === "is")
                      .map((filter) => filter.value);

                    const externalIdContainsValues = externalIdFilters
                      .filter((filter) => filter.operator === "contains")
                      .map((filter) => filter.value);

                    // Build the subquery for external ID
                    if (externalIdIsValues.length > 0 || externalIdContainsValues.length > 0) {
                      const externalIdConditions = [];

                      // Add "is" conditions for externalId
                      if (externalIdIsValues.length > 0) {
                        const inClause = sql.join(externalIdIsValues.map((value) => sql`${value}`));
                        externalIdConditions.push(sql`identities.external_id IN (${inClause})`);
                      }

                      // Add "contains" conditions for externalId
                      if (externalIdContainsValues.length > 0) {
                        externalIdContainsValues.forEach((value) => {
                          externalIdConditions.push(
                            sql`identities.external_id LIKE ${`%${value}%`}`,
                          );
                        });
                      }

                      // Create the EXISTS subquery with OR between conditions
                      const externalIdCondition = sql`
      EXISTS (
        SELECT 1 FROM identities 
        WHERE identities.id = ${key.identityId}
        AND (${sql.join(externalIdConditions, sql` OR `)})
      )`;

                      identityConditions.push(externalIdCondition);
                    }
                  }

                  // Add the combined identity conditions with OR between different identity types
                  if (identityConditions.length > 0) {
                    conditions.push(sql`(${sql.join(identityConditions, sql` OR `)})`);
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
                workspaceId: true,
              },
            },
          },
        },
      },
    })
    .catch((err) => {
      console.error("Database query error:", err);
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

export function createKeyDetailsMap(keys: any[]): Map<string, any> {
  const keyDetailsMap = new Map();
  for (const key of keys) {
    const rolesData = key.roles
      ? key.roles
          .filter((roleRelation: any) => roleRelation.role != null)
          .map((roleRelation: any) => ({
            name: roleRelation.role.name,
            description: roleRelation.role.description,
            createdAt: roleRelation.role.createdAtM,
            updatedAt: roleRelation.role.updatedAtM,
          }))
      : [];
    const permissionsData = key.permissions
      ? key.permissions
          .filter((permRelation: any) => permRelation.permission != null)
          .map((permRelation: any) => ({
            name: permRelation.permission.name,
            description: permRelation.permission.description,
            createdAt: permRelation.permission.createdAtM,
            updatedAt: permRelation.permission.updatedAtM,
          }))
      : [];
    const identityData = key.identity
      ? {
          external_id: key.identity.externalId,
        }
      : null;
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
      workspace_id: key.workspaceId,
      identity: identityData,
      roles: rolesData,
      permissions: permissionsData,
    };
    keyDetailsMap.set(key.id, keyDetails);
  }
  return keyDetailsMap;
}

export function extractRolesAndPermissions(key: any) {
  const roles = key.roles
    ? key.roles
        .filter((roleRelation: any) => roleRelation.role != null)
        .map((roleRelation: any) => ({
          name: roleRelation.role.name,
          description: roleRelation.role.description,
        }))
    : [];
  const permissions = key.permissions
    ? key.permissions
        .filter((permRelation: any) => permRelation.permission != null)
        .map((permRelation: any) => ({
          name: permRelation.permission.name,
          description: permRelation.permission.description,
        }))
    : [];
  return {
    roles,
    permissions,
  };
}
