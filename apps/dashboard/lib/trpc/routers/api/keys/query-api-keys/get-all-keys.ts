import type { AllOperatorsUrlValue } from "@/app/(app)/[workspace]/apis/[apiId]/_overview/filters.schema";
import { clickhouse } from "@/lib/clickhouse";
import { type SQL, db, like, or } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { identities } from "@unkey/db/src/schema";
import { z } from "zod";
import type { KeyDetails } from "./schema";

interface GetAllKeysInput {
  keyspaceId: string;
  workspaceId: string;
  filters?: {
    keyIds?: AllOperatorsUrlValue[] | null;
    names?: AllOperatorsUrlValue[] | null;
    identities?: AllOperatorsUrlValue[] | null;
    tags?: AllOperatorsUrlValue[] | null;
  };
  limit?: number;
  cursorKeyId?: string | null;
}

interface GetAllKeysResult {
  keys: KeyDetails[];
  hasMore: boolean;
  totalCount: number;
}

/**
 * Gets API keys with filtering and cursor-based pagination
 *
 * @param input Query parameters including keyspaceId, workspaceId, filters, and cursor
 * @returns Object containing matching keys, hasMore flag for pagination, and totalCount
 */
export async function getAllKeys({
  keyspaceId,
  workspaceId,
  filters = {},
  limit = 50,
  cursorKeyId = null,
}: GetAllKeysInput): Promise<GetAllKeysResult> {
  const { keyIds, names, identities: identityFilters, tags } = filters;

  try {
    // Security verification - ensure the keyspaceId belongs to the workspaceId
    const keyAuth = await db.query.keyAuth.findFirst({
      where: (keyAuth, { and, eq }) =>
        and(eq(keyAuth.id, keyspaceId), eq(keyAuth.workspaceId, workspaceId)),
      with: {
        api: {
          columns: {
            workspaceId: true,
          },
        },
      },
    });
    if (!keyAuth) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Keyspace not found or not authorized",
      });
    }

    // Get keys that match tag filters if provided
    let tagFilteredKeyIds: string[] | null = null;

    if (tags && tags.length > 0) {
      try {
        // Build tag filter conditions with proper parameterization
        const tagQueries = tags.map((tag, index) => {
          const paramKey = `tagValue${index}`;
          let condition: string;

          switch (tag.operator) {
            case "is":
              condition = `has(tags, {${paramKey}: String})`;
              break;
            case "contains":
              condition = `arrayExists(x -> position(x, {${paramKey}: String}) > 0, tags)`;
              break;
            case "startsWith":
              condition = `arrayExists(x -> startsWith(x, {${paramKey}: String}), tags)`;
              break;
            case "endsWith":
              condition = `arrayExists(x -> endsWith(x, {${paramKey}: String}), tags)`;
              break;
            default:
              condition = `has(tags, {${paramKey}: String})`;
          }

          return { condition, paramKey, value: tag.value };
        });

        // Build the params schema dynamically
        const paramsObj: Record<string, z.ZodString> = {
          workspaceId: z.string(),
          keyspaceId: z.string(),
        };
        tagQueries.forEach(({ paramKey }) => {
          paramsObj[paramKey] = z.string();
        });

        // Build the query parameters
        const queryParams: Record<string, string> = {
          workspaceId,
          keyspaceId,
        };
        tagQueries.forEach(({ paramKey, value }) => {
          queryParams[paramKey] = value;
        });

        const tagQuery = clickhouse.querier.query({
          query: `
            SELECT DISTINCT key_id
            FROM verifications.raw_key_verifications_v1
            WHERE workspace_id = {workspaceId: String}
              AND key_space_id = {keyspaceId: String}
              AND (${tagQueries.map(({ condition }) => condition).join(" OR ")})
          `,
          params: z.object(paramsObj),
          schema: z.object({
            key_id: z.string(),
          }),
        });

        const result = await tagQuery(queryParams);

        if (result.err) {
          console.error("ClickHouse query error:", result.err);
          tagFilteredKeyIds = [];
        } else {
          tagFilteredKeyIds = result.val.map((row) => row.key_id);
        }
      } catch (error) {
        console.error("Error querying tags from ClickHouse:", error);
        tagFilteredKeyIds = [];
      }
    }

    // Helper function to build the filter conditions (without cursor)
    // biome-ignore lint/suspicious/noExplicitAny: Leave it as is for now
    const buildFilterConditions = (key: any, { and, isNull, eq, sql }: any) => {
      const conditions = [eq(key.keyAuthId, keyspaceId), isNull(key.deletedAtM)];

      // Apply tag-based key filtering if we have filtered key IDs
      if (tagFilteredKeyIds !== null) {
        if (tagFilteredKeyIds.length === 0) {
          conditions.push(sql`1 = 0`);
        } else {
          conditions.push(
            sql`${key.id} IN (${sql.join(
              tagFilteredKeyIds.map((id) => sql`${id}`),
              sql`, `,
            )})`,
          );
        }
      }

      // Apply name filters
      if (names && names.length > 0) {
        const nameConditions = [];
        for (const filter of names) {
          const value = filter.value;
          if (typeof value !== "string") {
            continue;
          }
          switch (filter.operator) {
            case "is":
              nameConditions.push(eq(key.name, value));
              break;
            case "contains":
              nameConditions.push(sql`${key.name} LIKE ${`%${value}%`}`);
              break;
            case "startsWith":
              nameConditions.push(sql`${key.name} LIKE ${`${value}%`}`);
              break;
            case "endsWith":
              nameConditions.push(sql`${key.name} LIKE ${`%${value}`}`);
              break;
          }
        }
        if (nameConditions.length > 0) {
          conditions.push(sql`(${sql.join(nameConditions, sql` OR `)})`);
        }
      }

      // Apply key ID filters
      if (keyIds && keyIds.length > 0) {
        const keyIdConditions = [];
        for (const filter of keyIds) {
          const value = filter.value;
          if (typeof value !== "string") {
            continue;
          }
          switch (filter.operator) {
            case "is":
              keyIdConditions.push(eq(key.id, value));
              break;
            case "contains":
              keyIdConditions.push(sql`${key.id} LIKE ${`%${value}%`}`);
              break;
            case "startsWith":
              keyIdConditions.push(sql`${key.id} LIKE ${`${value}%`}`);
              break;
            case "endsWith":
              keyIdConditions.push(sql`${key.id} LIKE ${`%${value}`}`);
              break;
          }
        }
        if (keyIdConditions.length > 0) {
          conditions.push(sql`(${sql.join(keyIdConditions, sql` OR `)})`);
        }
      }

      if (identityFilters && identityFilters.length > 0) {
        const individualIdentityFilterConditions = [];
        for (const filter of identityFilters) {
          const value = filter.value;
          if (typeof value !== "string") {
            continue;
          }

          let ownerIdCondition: SQL<unknown>;
          switch (filter.operator) {
            case "is":
              ownerIdCondition = eq(key.ownerId, value);
              break;
            case "contains":
              ownerIdCondition = like(key.ownerId, `%${value}%`);
              break;
            case "startsWith":
              ownerIdCondition = like(key.ownerId, `${value}%`);
              break;
            case "endsWith":
              ownerIdCondition = like(key.ownerId, `%${value}`);
              break;
            default:
              ownerIdCondition = eq(key.ownerId, value);
          }

          const combinedCheckForThisFilter = or(
            sql`EXISTS (
                SELECT 1 FROM ${identities} -- Use schema object for table name is fine
                WHERE ${sql.raw("identities.id")} = ${
                  key.identityId
                } -- Use raw 'identities.id'; use schema 'key.identityId' for outer ref
                  AND ${(() => {
                    const rawExternalIdColumn = sql.raw("identities.external_id");
                    switch (filter.operator) {
                      case "is":
                        return sql`${rawExternalIdColumn} = ${value}`;
                      case "contains":
                        return sql`${rawExternalIdColumn} LIKE ${`%${value}%`}`;
                      case "startsWith":
                        return sql`${rawExternalIdColumn} LIKE ${`${value}%`}`;
                      case "endsWith":
                        return sql`${rawExternalIdColumn} LIKE ${`%${value}`}`;
                      default:
                        return sql`${rawExternalIdColumn} = ${value}`;
                    }
                  })()}
            )`,
            ownerIdCondition,
          );

          individualIdentityFilterConditions.push(combinedCheckForThisFilter);
        }

        if (individualIdentityFilterConditions.length > 0) {
          conditions.push(...individualIdentityFilterConditions);
        }
      }

      return and(...conditions);
    };

    // Get the total count of keys matching the filters (without pagination)
    const countQuery = await db.query.keys.findMany({
      where: buildFilterConditions,
      columns: {
        id: true,
      },
    });

    const totalCount = countQuery.length;
    const keysQuery = await db.query.keys.findMany({
      where: (key, helpers) => {
        const { and, lt } = helpers;
        // Get base filter conditions
        const filterConditions = buildFilterConditions(key, helpers);

        // Add cursor condition for pagination only
        if (cursorKeyId) {
          return and(filterConditions, lt(key.id, cursorKeyId));
        }

        return filterConditions;
      },
      with: {
        ratelimits: {
          columns: {
            id: true,
            name: true,
            limit: true,
            duration: true,
            autoApply: true,
          },
        },
        identity: {
          columns: {
            externalId: true,
          },
        },
      },
      limit: limit + 1, // Fetch one extra to determine if there are more results
      orderBy: (keys, { desc }) => desc(keys.id),
    });

    // Determine if there are more results
    const hasMore = keysQuery.length > limit;
    // Remove the extra item if it exists
    const keys = hasMore ? keysQuery.slice(0, limit) : keysQuery;

    const transformedKeys: KeyDetails[] = keys.map((key) => {
      const identityData = key.identity
        ? {
            external_id: key.identity.externalId,
          }
        : null;
      return {
        id: key.id,
        name: key.name,
        owner_id: key.ownerId,
        identity_id: key.identityId,
        enabled: key.enabled,
        expires: key.expires ? key.expires.getTime() : null,
        identity: identityData,
        updated_at_m: key.updatedAtM,
        start: key.start,
        metadata: key.meta,
        key: {
          credits: {
            enabled: key.remaining !== null,
            remaining: key.remaining,
            refillAmount: key.refillAmount,
            refillDay: key.refillDay,
          },
          ratelimits: {
            enabled: key.ratelimits.length > 0,
            items: key.ratelimits.map((r) => ({
              limit: r.limit,
              name: r.name,
              refillInterval: r.duration,
              id: r.id,
              autoApply: r.autoApply,
            })),
          },
        },
      };
    });

    return {
      keys: transformedKeys,
      hasMore,
      totalCount,
    };
  } catch (error) {
    if (error instanceof TRPCError) {
      throw error;
    }
    console.error("Error retrieving API keys:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message:
        "Failed to retrieve API keys. If this issue persists, please contact support@unkey.dev with the time this occurred.",
    });
  }
}
