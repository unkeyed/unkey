import { identitiesQueryPayload } from "@/components/identities-table/schema/identities.schema";
import { clickhouse } from "@/lib/clickhouse";
import { and, asc, count, db, desc, eq, inArray, like, notInArray, or, schema } from "@/lib/db";
import { getTimestampFromRelative } from "@/lib/utils";
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
  lastUsed: z.number().nullable(),
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

/**
 * Minimal identity shape used by the row-action menu and its dialogs
 * (edit ratelimit, edit metadata, delete). Both the list query and the
 * per-identity `getById` query return supersets of this.
 */
export type IdentityForActions = Pick<
  z.infer<typeof IdentityResponseSchema>,
  "id" | "externalId" | "meta" | "ratelimits"
>;

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

  // Fetch only the window of ClickHouse IDs needed for this page instead of
  // all verified IDs for the workspace. The IN/NOT IN lists stay bounded to
  // this window, keeping each request O(page depth) rather than O(total
  // identities). Correctness guarantee: accurate when DB-side filter attrition
  // within the window (deleted identities that still have ClickHouse records,
  // search mismatches) is low — the typical case. Pages near the
  // verified/never-used boundary may show minor inconsistencies when attrition
  // is significant.
  const chWindow = offset + pageLimit;

  const lastUsedQuery = clickhouse.querier.query({
    query: `
      SELECT identity_id
      FROM default.key_verifications_per_minute_v3
      WHERE workspace_id = {workspaceId: String}
      GROUP BY identity_id
      ORDER BY max(time) ${sortOrder === "asc" ? "ASC" : "DESC"}
      LIMIT {chWindow: UInt32}
    `,
    params: z.object({
      workspaceId: z.string(),
      chWindow: z.number(),
    }),
    schema: z.object({
      identity_id: z.string(),
    }),
  });

  const chResult = await lastUsedQuery({ workspaceId, chWindow });

  if (chResult.err) {
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Something went wrong when fetching data from ClickHouse.",
    });
  }

  const verifiedIds = chResult.val.map((r) => r.identity_id);

  // Filter verified IDs against DB base conditions (deleted flag, search).
  // The inArray list is bounded by chWindow.
  let filteredVerifiedIds: string[] = [];
  if (verifiedIds.length > 0) {
    const matchingVerified = await db
      .select({ id: schema.identities.id })
      .from(schema.identities)
      .where(and(...baseConditions, inArray(schema.identities.id, verifiedIds)));
    const matchingSet = new Set(matchingVerified.map((r) => r.id));
    // Preserve ClickHouse sort order.
    filteredVerifiedIds = verifiedIds.filter((id) => matchingSet.has(id));
  }

  const neverUsedWhere = and(
    ...baseConditions,
    ...(verifiedIds.length > 0 ? [notInArray(schema.identities.id, verifiedIds)] : []),
  );

  if (sortOrder === "desc") {
    // Most-recently-used verified items first, never-used (newest created) after.
    const verifiedOnPage = filteredVerifiedIds.slice(offset, offset + pageLimit);
    if (verifiedOnPage.length === pageLimit) {
      return verifiedOnPage;
    }

    // Spilled into never-used territory — fetch only the remaining slots.
    const neverUsedNeeded = pageLimit - verifiedOnPage.length;
    const neverUsedOffset = Math.max(0, offset - filteredVerifiedIds.length);

    const neverUsedRows = await db
      .select({ id: schema.identities.id })
      .from(schema.identities)
      .where(neverUsedWhere)
      .orderBy(desc(schema.identities.createdAt))
      .limit(neverUsedNeeded)
      .offset(neverUsedOffset);

    return [...verifiedOnPage, ...neverUsedRows.map((r) => r.id)];
  }

  // Ascending: never-used (newest created) first, verified (oldest last-used) after.
  // Count never-used to locate the page boundary before fetching rows.
  const [neverUsedCountRow] = await db
    .select({ count: count() })
    .from(schema.identities)
    .where(neverUsedWhere);
  const neverUsedCount = neverUsedCountRow.count;

  if (offset >= neverUsedCount) {
    // Page falls entirely within verified IDs.
    const verifiedOffset = offset - neverUsedCount;
    return filteredVerifiedIds.slice(verifiedOffset, verifiedOffset + pageLimit);
  }

  // Page starts within never-used IDs.
  const neverUsedOnPage = Math.min(pageLimit, neverUsedCount - offset);
  const neverUsedRows = await db
    .select({ id: schema.identities.id })
    .from(schema.identities)
    .where(neverUsedWhere)
    .orderBy(desc(schema.identities.createdAt))
    .limit(neverUsedOnPage)
    .offset(offset);

  if (neverUsedOnPage === pageLimit) {
    return neverUsedRows.map((r) => r.id);
  }

  // Fill remaining slots from the start of the verified list.
  const verifiedNeeded = pageLimit - neverUsedOnPage;
  return [...neverUsedRows.map((r) => r.id), ...filteredVerifiedIds.slice(0, verifiedNeeded)];
}

export const queryIdentities = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(identitiesQueryPayload)
  .output(IdentitiesResponse)
  .query(async ({ ctx, input }) => {
    try {
      const {
        limit = 50,
        page = 1,
        search,
        externalId: externalIdFilter,
        lastUsedStart,
        lastUsedEnd,
        lastUsedSince,
        sortBy = "createdAt",
        sortOrder = "desc",
      } = input;
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

      if (externalIdFilter && externalIdFilter.length > 0) {
        const externalIdConditions = externalIdFilter.map((f) => {
          const escaped = escapeLike(f.value);
          switch (f.operator) {
            case "is":
              return eq(schema.identities.externalId, f.value);
            case "startsWith":
              return like(schema.identities.externalId, `${escaped}%`);
            case "endsWith":
              return like(schema.identities.externalId, `%${escaped}`);
            default:
              return like(schema.identities.externalId, `%${escaped}%`);
          }
        });
        const combined =
          externalIdConditions.length === 1 ? externalIdConditions[0] : or(...externalIdConditions);
        if (combined) {
          baseConditions.push(combined);
        }
      }

      // Filter by lastUsed time range: query ClickHouse for matching identity IDs,
      // then restrict baseConditions to only those IDs.
      let lastUsedMatchingIds: string[] | null = null;
      const hasLastUsedFilter = lastUsedSince || lastUsedStart !== undefined;
      if (hasLastUsedFilter) {
        const resolvedStart = lastUsedSince
          ? getTimestampFromRelative(lastUsedSince)
          : (lastUsedStart ?? 0);
        const resolvedEnd = lastUsedSince ? Date.now() : (lastUsedEnd ?? Date.now());

        const lastUsedRangeQuery = clickhouse.querier.query({
          query: `
            SELECT identity_id
            FROM default.key_verifications_per_minute_v3
            WHERE workspace_id = {workspaceId: String}
            GROUP BY identity_id
            HAVING max(toUnixTimestamp(time) * 1000) >= {startMs: UInt64}
               AND max(toUnixTimestamp(time) * 1000) <= {endMs: UInt64}
          `,
          params: z.object({
            workspaceId: z.string(),
            startMs: z.number(),
            endMs: z.number(),
          }),
          schema: z.object({ identity_id: z.string() }),
        });

        const rangeResult = await lastUsedRangeQuery({
          workspaceId,
          startMs: resolvedStart,
          endMs: resolvedEnd,
        });

        if (rangeResult.err) {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Something went wrong when fetching data from ClickHouse.",
          });
        }

        lastUsedMatchingIds = rangeResult.val.map((r) => r.identity_id);
        // No identities match → short-circuit before any DB queries
        if (lastUsedMatchingIds.length === 0) {
          const totalPages = Math.max(1, Math.ceil(0 / limit));
          return { identities: [], total: 0, page, pageSize: limit, totalPages };
        }
        baseConditions.push(inArray(schema.identities.id, lastUsedMatchingIds));
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
        // Count the joined column rather than count(*). With a LEFT JOIN, identities
        // with zero matches still produce one row (with NULL columns), which count(*)
        // would count as 1 — ranking them identically to identities with one match.
        const countColumn = sortBy === "keyCount" ? schema.keys.id : schema.ratelimits.id;
        const countExpr = count(countColumn);
        const orderDir = sortOrder === "asc" ? asc(countExpr) : desc(countExpr);

        if (sortBy === "keyCount") {
          const results = await db
            .select({ id: schema.identities.id })
            .from(schema.identities)
            .leftJoin(schema.keys, eq(schema.keys.identityId, schema.identities.id))
            .where(and(...baseConditions))
            .groupBy(schema.identities.id)
            .orderBy(orderDir, desc(schema.identities.createdAt))
            .limit(limit)
            .offset(offset);
          sortedIds = results.map((r) => r.id);
        } else {
          const results = await db
            .select({ id: schema.identities.id })
            .from(schema.identities)
            .leftJoin(schema.ratelimits, eq(schema.ratelimits.identityId, schema.identities.id))
            .where(and(...baseConditions))
            .groupBy(schema.identities.id)
            .orderBy(orderDir, desc(schema.identities.createdAt))
            .limit(limit)
            .offset(offset);
          sortedIds = results.map((r) => r.id);
        }
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

      const identitiesQuery = await db.query.identities.findMany({
        where: (identity, helpers) => {
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

          if (sortedIds !== null && sortedIds.length > 0) {
            // Pre-sorted: sortedIds already incorporates all baseConditions (externalId, lastUsed, etc.)
            conditions.push(helpers.inArray(identity.id, sortedIds));
          } else {
            // Non-pre-sorted: apply column-level filters explicitly since baseConditions
            // is not wired into this relational query path.
            if (externalIdFilter && externalIdFilter.length > 0) {
              const externalIdConditions = externalIdFilter.map((f) => {
                const escaped = escapeLike(f.value);
                switch (f.operator) {
                  case "is":
                    return helpers.eq(identity.externalId, f.value);
                  case "startsWith":
                    return helpers.like(identity.externalId, `${escaped}%`);
                  case "endsWith":
                    return helpers.like(identity.externalId, `%${escaped}`);
                  default:
                    return helpers.like(identity.externalId, `%${escaped}%`);
                }
              });
              const combined =
                externalIdConditions.length === 1
                  ? externalIdConditions[0]
                  : helpers.or(...externalIdConditions);
              if (combined) {
                conditions.push(combined);
              }
            }
            if (lastUsedMatchingIds !== null && lastUsedMatchingIds.length > 0) {
              conditions.push(helpers.inArray(identity.id, lastUsedMatchingIds));
            }
          }

          return helpers.and(...conditions);
        },
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
              return [direction(identities.externalId), direction(identities.id)];
            }
            case "keyCount":
            case "ratelimitCount":
            case "lastUsed": {
              // Pre-sorted: results are re-ordered in JS after fetch
              return d(identities.createdAt);
            }
            default: {
              return [direction(identities.createdAt), direction(identities.id)];
            }
          }
        },
      });

      // Batch-fetch last-used timestamps from ClickHouse for all identities on this page.
      // On ClickHouse failure, degrade gracefully — identities render with "Never used"
      // rather than failing the entire query.
      const identityIds = identitiesQuery.map((i) => i.id);
      const lastUsedMap = new Map<string, number>();

      if (identityIds.length > 0) {
        const lastUsedQuery = clickhouse.querier.query({
          query: `
            SELECT
              identity_id,
              maxOrNull(toUnixTimestamp(time) * 1000) as last_used
            FROM default.key_verifications_per_minute_v3
            WHERE workspace_id = {workspaceId: String}
              AND identity_id IN ({identityIds: Array(String)})
            GROUP BY identity_id
          `,
          params: z.object({
            workspaceId: z.string(),
            identityIds: z.array(z.string()),
          }),
          schema: z.object({
            identity_id: z.string(),
            last_used: z.number().nullable(),
          }),
        });

        const chResult = await lastUsedQuery({ workspaceId, identityIds });
        if (!chResult.err) {
          for (const row of chResult.val) {
            if (row.last_used !== null) {
              lastUsedMap.set(row.identity_id, row.last_used);
            }
          }
        }
      }

      let transformedIdentities = identitiesQuery.map((identity) => ({
        id: identity.id,
        externalId: identity.externalId,
        workspaceId: identity.workspaceId,
        environment: identity.environment,
        meta: identity.meta,
        createdAt: identity.createdAt,
        updatedAt: identity.updatedAt ? identity.updatedAt : null,
        lastUsed: lastUsedMap.get(identity.id) ?? null,
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
