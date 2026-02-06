import type { AllOperatorsUrlValue } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/filters.schema";
import { clickhouse } from "@/lib/clickhouse";
import { type SQL, and, count, db, eq, isNull, like, or, sql } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { identities, keys as keysSchema } from "@unkey/db/src/schema";
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
            FROM default.key_verifications_raw_v2
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
    // biome-ignore lint/suspicious/noExplicitAny: Drizzle query builder types are complex and vary between schema and query contexts
    const buildFilterConditions = (key: any, helpers: any): SQL<unknown> => {
      const conditions = [helpers.eq(key.keyAuthId, keyspaceId), helpers.isNull(key.deletedAtM)];

      // Apply tag-based key filtering if we have filtered key IDs
      if (tagFilteredKeyIds !== null) {
        if (tagFilteredKeyIds.length === 0) {
          conditions.push(helpers.sql`1 = 0`);
        } else {
          conditions.push(
            helpers.sql`${key.id} IN (${helpers.sql.join(
              tagFilteredKeyIds.map((id) => helpers.sql`${id}`),
              helpers.sql`, `,
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
              nameConditions.push(helpers.eq(key.name, value));
              break;
            case "contains":
              nameConditions.push(helpers.sql`${key.name} LIKE ${`%${value}%`}`);
              break;
            case "startsWith":
              nameConditions.push(helpers.sql`${key.name} LIKE ${`${value}%`}`);
              break;
            case "endsWith":
              nameConditions.push(helpers.sql`${key.name} LIKE ${`%${value}`}`);
              break;
          }
        }
        if (nameConditions.length > 0) {
          conditions.push(helpers.sql`(${helpers.sql.join(nameConditions, helpers.sql` OR `)})`);
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
              keyIdConditions.push(helpers.eq(key.id, value));
              break;
            case "contains":
              keyIdConditions.push(helpers.sql`${key.id} LIKE ${`%${value}%`}`);
              break;
            case "startsWith":
              keyIdConditions.push(helpers.sql`${key.id} LIKE ${`${value}%`}`);
              break;
            case "endsWith":
              keyIdConditions.push(helpers.sql`${key.id} LIKE ${`%${value}`}`);
              break;
          }
        }
        if (keyIdConditions.length > 0) {
          conditions.push(helpers.sql`(${helpers.sql.join(keyIdConditions, helpers.sql` OR `)})`);
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
              ownerIdCondition = helpers.eq(key.ownerId, value);
              break;
            case "contains":
              ownerIdCondition = helpers.like(key.ownerId, `%${value}%`);
              break;
            case "startsWith":
              ownerIdCondition = helpers.like(key.ownerId, `${value}%`);
              break;
            case "endsWith":
              ownerIdCondition = helpers.like(key.ownerId, `%${value}`);
              break;
            default:
              ownerIdCondition = helpers.eq(key.ownerId, value);
          }

          const combinedCheckForThisFilter = helpers.or(
            helpers.sql`EXISTS (
                SELECT 1 FROM ${identities} -- Use schema object for table name is fine
                WHERE ${helpers.sql.raw("identities.id")} = ${
                  key.identityId
                } -- Use raw 'identities.id'; use schema 'key.identityId' for outer ref
                  AND ${(() => {
                    const rawExternalIdColumn = helpers.sql.raw("identities.external_id");
                    switch (filter.operator) {
                      case "is":
                        return helpers.sql`${rawExternalIdColumn} = ${value}`;
                      case "contains":
                        return helpers.sql`${rawExternalIdColumn} LIKE ${`%${value}%`}`;
                      case "startsWith":
                        return helpers.sql`${rawExternalIdColumn} LIKE ${`${value}%`}`;
                      case "endsWith":
                        return helpers.sql`${rawExternalIdColumn} LIKE ${`%${value}`}`;
                      default:
                        return helpers.sql`${rawExternalIdColumn} = ${value}`;
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

      return helpers.and(...conditions);
    };

    // Get the total count using a proper COUNT query instead of fetching all rows
    const [countResult] = await db
      .select({ count: count() })
      .from(keysSchema)
      .where(buildFilterConditions(keysSchema, { and, isNull, eq, sql, like, or }));

    const totalCount = countResult?.count ?? 0;
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
        "Failed to retrieve API keys. If this issue persists, please contact support@unkey.com with the time this occurred.",
    });
  }
}
