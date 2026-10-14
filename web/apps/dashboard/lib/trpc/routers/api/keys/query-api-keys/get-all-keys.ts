import type { AllOperatorsUrlValue } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/filters.schema";
import type { ApiKeysCursor } from "@/components/api-keys-table/schema/api-keys.schema";
import { clickhouse } from "@/lib/clickhouse";
import { type SQL, db, eq, sql } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { identities } from "@unkey/db/src/schema";
import { z } from "zod";
import { resolveIdentityFilter } from "../identity-filter";
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
  cursor?: ApiKeysCursor | null;
  sortBy?: "id" | "start" | "lastUsedAt";
  sortOrder?: "asc" | "desc";
}

interface GetAllKeysResult {
  keys: KeyDetails[];
  nextCursor: ApiKeysCursor | null;
}

export async function getAllKeys({
  keyspaceId,
  workspaceId,
  filters = {},
  limit = 50,
  cursor,
  sortBy = "id",
  sortOrder = "desc",
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
      columns: { id: true },
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

    const identityConditions: SQL[] = [];
    for (const filter of identityFilters ?? []) {
      const value = filter.value;
      if (typeof value !== "string") {
        continue;
      }

      switch (filter.operator) {
        case "contains":
          identityConditions.push(sql`LOWER(${identities.externalId}) LIKE LOWER(${`%${value}%`})`);
          break;
        case "startsWith":
          identityConditions.push(sql`LOWER(${identities.externalId}) LIKE LOWER(${`${value}%`})`);
          break;
        case "endsWith":
          identityConditions.push(sql`LOWER(${identities.externalId}) LIKE LOWER(${`%${value}`})`);
          break;
        default:
          identityConditions.push(eq(identities.externalId, value));
      }
    }

    const identityFilter =
      identityConditions.length > 0
        ? await resolveIdentityFilter({
            workspaceId,
            keyspaceId,
            condition: sql`(${sql.join(identityConditions, sql` AND `)})`,
          })
        : undefined;

    const keysQuery = await db.query.keys.findMany({
      where: (key, helpers) => {
        const conditions = [
          helpers.eq(key.keyAuthId, keyspaceId),
          helpers.isNull(key.deletedAtM),
          helpers.or(helpers.isNull(key.expires), helpers.gt(key.expires, new Date())),
        ];

        if (cursor) {
          const compare = sortOrder === "desc" ? helpers.lt : helpers.gt;
          const tiebreaker =
            sortBy === "lastUsedAt" ? compare(key.pk, cursor.pk) : compare(key.id, cursor.id);
          conditions.push(
            sortBy === "id"
              ? compare(key.id, cursor.id)
              : helpers.or(
                  compare(key[sortBy], cursor[sortBy]),
                  helpers.and(helpers.eq(key[sortBy], cursor[sortBy]), tiebreaker),
                ),
          );
        }

        if (identityFilter) {
          conditions.push(identityFilter);
        }

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
                nameConditions.push(helpers.sql`LOWER(${key.name}) LIKE LOWER(${`%${value}%`})`);
                break;
              case "startsWith":
                nameConditions.push(helpers.sql`LOWER(${key.name}) LIKE LOWER(${`${value}%`})`);
                break;
              case "endsWith":
                nameConditions.push(helpers.sql`LOWER(${key.name}) LIKE LOWER(${`%${value}`})`);
                break;
            }
          }
          if (nameConditions.length > 0) {
            conditions.push(helpers.sql`(${helpers.sql.join(nameConditions, helpers.sql` OR `)})`);
          }
        }

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
                keyIdConditions.push(helpers.sql`LOWER(${key.id}) LIKE LOWER(${`%${value}%`})`);
                break;
              case "startsWith":
                keyIdConditions.push(helpers.sql`LOWER(${key.id}) LIKE LOWER(${`${value}%`})`);
                break;
              case "endsWith":
                keyIdConditions.push(helpers.sql`LOWER(${key.id}) LIKE LOWER(${`%${value}`})`);
                break;
            }
          }
          if (keyIdConditions.length > 0) {
            conditions.push(helpers.sql`(${helpers.sql.join(keyIdConditions, helpers.sql` OR `)})`);
          }
        }

        return helpers.and(...conditions);
      },
      columns: {
        pk: true,
        id: true,
        name: true,
        identityId: true,
        enabled: true,
        expires: true,
        updatedAtM: true,
        prefix: true,
        start: true,
        end: true,
        lastUsedAt: true,
        meta: true,
        remaining: true,
        refillAmount: true,
        refillDay: true,
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
      limit: limit + 1,
      orderBy: (keys, helpers) => {
        const column = keys[sortBy];
        const primary = sortOrder === "desc" ? helpers.desc(column) : helpers.asc(column);
        const tiebreakerColumn = sortBy === "lastUsedAt" ? keys.pk : keys.id;
        const tiebreaker =
          sortOrder === "desc" ? helpers.desc(tiebreakerColumn) : helpers.asc(tiebreakerColumn);
        return [primary, tiebreaker];
      },
    });

    const hasMore = keysQuery.length > limit;
    const keys = keysQuery.slice(0, limit);
    const last = keys.at(-1);

    const transformedKeys: KeyDetails[] = keys.map((key) => {
      const identityData = key.identity
        ? {
            external_id: key.identity.externalId,
          }
        : null;
      return {
        id: key.id,
        name: key.name,
        identity_id: key.identityId,
        enabled: key.enabled,
        expires: key.expires ? key.expires.getTime() : null,
        identity: identityData,
        updated_at_m: key.updatedAtM,
        prefix: key.prefix,
        start: key.start,
        end: key.end,
        last_used_at: key.lastUsedAt,
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
      nextCursor:
        hasMore && last
          ? { pk: last.pk, id: last.id, start: last.start, lastUsedAt: last.lastUsedAt }
          : null,
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
