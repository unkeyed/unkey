import type { KeysListFilterUrlValue } from "@/app/(app)/apis/[apiId]/keys_v2/[keyAuthId]/_components/filters.schema";
import { type SQL, db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { identities } from "@unkey/db/src/schema";
import type { KeyDetails } from "./schema";

interface GetAllKeysInput {
  keyspaceId: string;
  workspaceId: string;
  filters?: {
    keyIds?: KeysListFilterUrlValue[] | null;
    names?: KeysListFilterUrlValue[] | null;
    identities?: KeysListFilterUrlValue[] | null;
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
  const { keyIds, names, identities: identityFilters } = filters;
  try {
    // Step 1: Security verification - ensure the keyspaceId belongs to the workspaceId
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

    // Helper function to build the filter conditions (without cursor)
    const buildFilterConditions = (key: any, { and, isNull, eq, sql }: any) => {
      const conditions = [eq(key.keyAuthId, keyspaceId), isNull(key.deletedAtM)];

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
          }
        }
        if (keyIdConditions.length > 0) {
          conditions.push(sql`(${sql.join(keyIdConditions, sql` OR `)})`);
        }
      }

      // Apply identity filters - checking both identities table and ownerId
      if (identityFilters && identityFilters.length > 0) {
        const allIdentityConditions = [];
        for (const filter of identityFilters) {
          const value = filter.value;
          if (typeof value !== "string") {
            continue;
          }
          let condition: SQL<any>;
          let ownerCondition: SQL<any>;
          switch (filter.operator) {
            case "is":
              condition = sql`identities.external_id = ${value}`;
              ownerCondition = sql`${key.ownerId} = ${value}`;
              break;
            case "contains":
              condition = sql`identities.external_id LIKE ${`%${value}%`}`;
              ownerCondition = sql`${key.ownerId} LIKE ${`%${value}%`}`;
              break;
            case "startsWith":
              condition = sql`identities.external_id LIKE ${`${value}%`}`;
              ownerCondition = sql`${key.ownerId} LIKE ${`${value}%`}`;
              break;
            case "endsWith":
              condition = sql`identities.external_id LIKE ${`%${value}`}`;
              ownerCondition = sql`${key.ownerId} LIKE ${`%${value}`}`;
              break;
            default:
              condition = sql`identities.external_id = ${value}`;
              ownerCondition = sql`${key.ownerId} = ${value}`;
          }
          // Check for matches in identity external ID via identities table
          allIdentityConditions.push(sql`
            EXISTS (
              SELECT 1 FROM ${identities} 
              WHERE ${identities.id} = ${key.identityId}
              AND ${condition}
            )`);
          // Also check if it matches the ownerId directly
          allIdentityConditions.push(ownerCondition);
        }
        if (allIdentityConditions.length > 0) {
          conditions.push(sql`(${sql.join(allIdentityConditions, sql` OR `)})`);
        }
      }

      return and(...conditions);
    };

    // Step 2: Get the total count of keys matching the filters (without pagination)
    const countQuery = await db.query.keys.findMany({
      where: buildFilterConditions,
      columns: {
        id: true,
      },
    });

    const totalCount = countQuery.length;

    // Step 3: Get the paginated keys with filters and cursor
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

    const transformedKeys = keys.map((key) => {
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
