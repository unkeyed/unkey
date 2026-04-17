import { identitiesQueryPayload } from "@/components/identities-table/schema/identities.schema";
import { clickhouse } from "@/lib/clickhouse";
import {
  and,
  asc,
  count,
  db,
  desc,
  eq,
  inArray,
  like,
  notInArray,
  or,
  schema,
  sql,
} from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../trpc";
import { escapeLike } from "../utils/sql";

export const IdentityResponseSchema = z.object({
  id: z.string(),
  externalId: z.string(),
  workspaceId: z.string(),
  environment: z.string(),
  meta: z.record(z.string(), z.unknown()).nullable(),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
  keys: z.array(z.object({ id: z.string() })),
  ratelimits: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      limit: z.number(),
      duration: z.number(),
      autoApply: z.boolean(),
    }),
  ),
});

const IdentitiesResponse = z.object({
  identities: z.array(IdentityResponseSchema),
  total: z.number(),
  page: z.number(),
  pageSize: z.number(),
  totalPages: z.number(),
});

type BaseCondition = ReturnType<typeof eq>;

/**
 * Gets identity IDs sorted by last verification time from ClickHouse,
 * merged with never-verified identities from MySQL, then paginated.
 * Never-verified identities appear last when sorting descending, first when ascending.
 */
async function getLastUsedSortedIds(params: {
  workspaceId: string;
  baseConditions: BaseCondition[];
  sortOrder: "asc" | "desc";
  limit: number;
  offset: number;
}): Promise<string[]> {
  const { workspaceId, baseConditions, sortOrder, limit: pageLimit, offset } = params;

  const lastUsedQuery = clickhouse.querier.query({
    query: `
      SELECT identity_id
      FROM default.key_verifications_per_minute_v3
      WHERE workspace_id = {workspaceId: String}
      GROUP BY identity_id
      ORDER BY max(time) ${sortOrder === "asc" ? "ASC" : "DESC"}
    `,
    params: z.object({
      workspaceId: z.string(),
    }),
    schema: z.object({
      identity_id: z.string(),
    }),
  });

  const chResult = await lastUsedQuery({ workspaceId });

  if (chResult.err) {
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Something went wrong when fetching data from ClickHouse.",
    });
  }

  const verifiedIds = chResult.val.map((r) => r.identity_id);

  // Get identity IDs that have never been verified (not in ClickHouse).
  const neverUsedResults = await db
    .select({ id: schema.identities.id })
    .from(schema.identities)
    .where(
      and(
        ...baseConditions,
        ...(verifiedIds.length > 0 ? [notInArray(schema.identities.id, verifiedIds)] : []),
      ),
    )
    .orderBy(desc(schema.identities.createdAt));

  const neverUsedIds = neverUsedResults.map((r) => r.id);

  // Filter verified IDs to only those matching base conditions.
  // ClickHouse may return IDs for deleted identities or ones not matching search.
  let filteredVerifiedIds = verifiedIds;
  if (verifiedIds.length > 0) {
    const matchingVerified = await db
      .select({ id: schema.identities.id })
      .from(schema.identities)
      .where(and(...baseConditions, inArray(schema.identities.id, verifiedIds)));
    const matchingSet = new Set(matchingVerified.map((r) => r.id));
    filteredVerifiedIds = verifiedIds.filter((id) => matchingSet.has(id));
  }

  // Ascending = never-used first, descending = verified first.
  const allSortedIds =
    sortOrder === "asc"
      ? [...neverUsedIds, ...filteredVerifiedIds]
      : [...filteredVerifiedIds, ...neverUsedIds];

  return allSortedIds.slice(offset, offset + pageLimit);
}

export const queryIdentities = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(identitiesQueryPayload)
  .output(IdentitiesResponse)
  .query(async ({ ctx, input }) => {
    try {
      const { limit = 50, page = 1, search, sortBy = "createdAt", sortOrder = "desc" } = input;
      const workspaceId = ctx.workspace.id;
      const offset = (page - 1) * limit;

      // Build base filter conditions for SQL queries
      const baseConditions = [
        eq(schema.identities.workspaceId, workspaceId),
        eq(schema.identities.deleted, false),
      ];

      if (search) {
        const escapedSearch = escapeLike(search);
        const searchCondition = or(
          like(schema.identities.externalId, `%${escapedSearch}%`),
          like(schema.identities.id, `%${escapedSearch}%`),
        );
        if (searchCondition) {
          baseConditions.push(searchCondition);
        }
      }

      // Get total count of identities matching the filters (without pagination)
      const [countResult] = await db
        .select({ count: count() })
        .from(schema.identities)
        .where(and(...baseConditions));

      const total = countResult.count;

      const isPreSorted =
        sortBy === "keyCount" || sortBy === "ratelimitCount" || sortBy === "lastUsed";

      // For count-based and ClickHouse-based sorts, first get the sorted page of identity IDs,
      // then fetch full relational data for those IDs.
      // For direct column sorts, use the relational query directly.
      let sortedIds: string[] | null = null;

      if (sortBy === "keyCount" || sortBy === "ratelimitCount") {
        const countQuery = db.select({ id: schema.identities.id }).from(schema.identities);

        if (sortBy === "keyCount") {
          countQuery.leftJoin(schema.keys, eq(schema.keys.identityId, schema.identities.id));
        } else {
          countQuery.leftJoin(
            schema.ratelimits,
            eq(schema.ratelimits.identityId, schema.identities.id),
          );
        }

        const countResults = await countQuery
          .where(and(...baseConditions))
          .groupBy(schema.identities.id)
          .orderBy(
            sortOrder === "asc" ? asc(sql`count(*)`) : desc(sql`count(*)`),
            desc(schema.identities.createdAt),
          )
          .limit(limit)
          .offset(offset);

        sortedIds = countResults.map((r) => r.id);
      } else if (sortBy === "lastUsed") {
        sortedIds = await getLastUsedSortedIds({
          workspaceId,
          baseConditions,
          sortOrder,
          limit,
          offset,
        });
      }

      // No matching identities for this page — return early
      if (isPreSorted && sortedIds !== null && sortedIds.length === 0) {
        const totalPages = Math.max(1, Math.ceil(total / limit));
        return {
          identities: [],
          total,
          page,
          pageSize: limit,
          totalPages,
        };
      }

      // Helper function to build filter conditions for query API
      // biome-ignore lint/suspicious/noExplicitAny: Drizzle query builder types are complex and vary between schema and query contexts
      const buildFilterConditions = (identity: any, helpers: any) => {
        const conditions = [
          helpers.eq(identity.workspaceId, workspaceId),
          helpers.eq(identity.deleted, false),
        ];

        if (search) {
          const escapedSearch = escapeLike(search);
          const searchCondition = helpers.or(
            helpers.like(identity.externalId, `%${escapedSearch}%`),
            helpers.like(identity.id, `%${escapedSearch}%`),
          );
          if (searchCondition) {
            conditions.push(searchCondition);
          }
        }

        // When pre-sorted, filter to only the pre-sorted IDs
        if (sortedIds !== null && sortedIds.length > 0) {
          conditions.push(helpers.inArray(identity.id, sortedIds));
        }

        return helpers.and(...conditions);
      };

      const identitiesQuery = await db.query.identities.findMany({
        where: buildFilterConditions,
        with: {
          keys: {
            columns: {
              id: true,
            },
          },
          ratelimits: {
            columns: {
              id: true,
              name: true,
              limit: true,
              duration: true,
              autoApply: true,
            },
          },
        },
        // When pre-sorted, pagination is handled by the pre-sort query above
        ...(isPreSorted
          ? {}
          : {
              limit,
              offset,
            }),
        orderBy: (identities, { asc: a, desc: d }) => {
          const direction = sortOrder === "asc" ? a : d;
          switch (sortBy) {
            case "externalId": {
              return direction(identities.externalId);
            }
            case "keyCount":
            case "ratelimitCount":
            case "lastUsed": {
              // Pre-sorted: results are re-ordered in JS after fetch
              return d(identities.createdAt);
            }
            default: {
              return direction(identities.createdAt);
            }
          }
        },
      });

      let transformedIdentities = identitiesQuery.map((identity) => ({
        id: identity.id,
        externalId: identity.externalId,
        workspaceId: identity.workspaceId,
        environment: identity.environment,
        meta: identity.meta,
        createdAt: identity.createdAt,
        updatedAt: identity.updatedAt ? identity.updatedAt : null,
        keys: identity.keys,
        ratelimits: identity.ratelimits,
      }));

      // Re-sort in JS to match the order from the pre-sort query
      if (sortedIds !== null && sortedIds.length > 0) {
        const idOrder = new Map(sortedIds.map((id, index) => [id, index]));
        transformedIdentities = transformedIdentities.sort(
          (a, b) => (idOrder.get(a.id) ?? 0) - (idOrder.get(b.id) ?? 0),
        );
      }

      const totalPages = Math.max(1, Math.ceil(total / limit));

      return {
        identities: transformedIdentities,
        total,
        page,
        pageSize: limit,
        totalPages,
      };
    } catch (error) {
      console.error("Error retrieving identities:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve identities. If this issue persists, please contact support@unkey.com with the time this occurred.",
      });
    }
  });
