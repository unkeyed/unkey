import type { KeysOverviewFilterUrlValue } from "@/app/(app)/apis/[apiId]/_overview/filters.schema";
import { type SQL, db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { identities } from "@unkey/db/src/schema";

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

                if (namesFromInput && namesFromInput.length > 0) {
                  const nameIsValues = namesFromInput
                    .filter((filter) => filter.operator === "is")
                    .map((filter) => filter.value);

                  const nameContainsValues = namesFromInput
                    .filter((filter) => filter.operator === "contains")
                    .map((filter) => filter.value);

                  const nameStartsWithValues = namesFromInput
                    .filter((filter) => filter.operator === "startsWith")
                    .map((filter) => filter.value);

                  const nameEndsWithValues = namesFromInput
                    .filter((filter) => filter.operator === "endsWith")
                    .map((filter) => filter.value);

                  if (nameIsValues.length > 0) {
                    conditions.push(inArray(key.name, nameIsValues as string[]));
                  }

                  if (nameContainsValues.length > 0) {
                    nameContainsValues.forEach((value) => {
                      conditions.push(sql`${key.name} LIKE ${`%${value}%`}`);
                    });
                  }

                  if (nameStartsWithValues.length > 0) {
                    nameStartsWithValues.forEach((value) => {
                      conditions.push(sql`${key.name} LIKE ${`${value}%`}`);
                    });
                  }

                  if (nameEndsWithValues.length > 0) {
                    nameEndsWithValues.forEach((value) => {
                      conditions.push(sql`${key.name} LIKE ${`%${value}`}`);
                    });
                  }
                }

                if (keyIdsFromInput && keyIdsFromInput.length > 0) {
                  const keyIdValues = keyIdsFromInput
                    .filter((filter) => filter.operator === "is")
                    .map((filter) => filter.value);

                  const keyIdContainsValues = keyIdsFromInput
                    .filter((filter) => filter.operator === "contains")
                    .map((filter) => filter.value);

                  if (keyIdValues.length > 0) {
                    conditions.push(inArray(key.id, keyIdValues as string[]));
                  }

                  if (keyIdContainsValues.length > 0) {
                    keyIdContainsValues.forEach((value) =>
                      conditions.push(sql`${key.id} LIKE ${`%${value}%`}`),
                    );
                  }
                }

                const allIdentityConditions = [];
                if (identitiesFromInput && identitiesFromInput.length > 0) {
                  for (const filter of identitiesFromInput) {
                    const value = filter.value;
                    if (typeof value !== "string") {
                      continue;
                    }

                    const operator = filter.operator;

                    let condition: SQL<any>;

                    switch (operator) {
                      case "is":
                        condition = sql`identities.external_id = ${value}`;
                        break;
                      case "contains":
                        condition = sql`identities.external_id LIKE ${`%${value}%`}`;
                        break;
                      case "startsWith":
                        condition = sql`identities.external_id LIKE ${`${value}%`}`;
                        break;
                      case "endsWith":
                        condition = sql`identities.external_id LIKE ${`%${value}`}`;
                        break;
                      default:
                        condition = sql`identities.external_id = ${value}`;
                    }

                    allIdentityConditions.push(sql`
        EXISTS (
          SELECT 1 FROM ${identities} 
          WHERE ${identities.id} = ${key.identityId}
          AND ${condition}
        )`);
                    let ownerCondition: SQL<any>;

                    switch (operator) {
                      case "is":
                        ownerCondition = sql`${key.ownerId} = ${value}`;
                        break;
                      case "contains":
                        ownerCondition = sql`${key.ownerId} LIKE ${`%${value}%`}`;
                        break;
                      case "startsWith":
                        ownerCondition = sql`${key.ownerId} LIKE ${`${value}%`}`;
                        break;
                      case "endsWith":
                        ownerCondition = sql`${key.ownerId} LIKE ${`%${value}`}`;
                        break;
                      default:
                        ownerCondition = sql`${key.ownerId} = ${value}`;
                    }

                    allIdentityConditions.push(ownerCondition);
                  }
                }

                if (allIdentityConditions.length > 0) {
                  conditions.push(sql`(${sql.join(allIdentityConditions, sql` OR `)})`);
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

export const getApi = async (apiId: string, workspaceId: string) => {
  const api = await db.query.apis
    .findFirst({
      where: (api, { and, eq, isNull }) =>
        and(eq(api.id, apiId), eq(api.workspaceId, workspaceId), isNull(api.deletedAtM)),
      with: {
        keyAuth: {
          columns: {
            id: true,
          },
        },
      },
    })
    .catch((err) => {
      console.error("Database query error:", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve API information.",
      });
    });

  return api;
};
