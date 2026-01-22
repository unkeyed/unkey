import { and, count, db, eq, like, or, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../trpc";
import { escapeLike } from "../utils/sql";

const identitiesQueryPayload = z.object({
  cursor: z.string().optional(),
  limit: z.number().optional().prefault(50),
  search: z.string().optional(),
});

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
  hasMore: z.boolean(),
  nextCursor: z.string().nullish(),
  totalCount: z.number(),
});

export const queryIdentities = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(identitiesQueryPayload)
  .output(IdentitiesResponse)
  .query(async ({ ctx, input }) => {
    try {
      const { limit = 50, cursor, search } = input;
      const workspaceId = ctx.workspace.id;

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

      const totalCount = countResult.count;

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

        return helpers.and(...conditions);
      };

      const identitiesQuery = await db.query.identities.findMany({
        where: (identity, helpers) => {
          const { and, lt } = helpers;
          // Get base filter conditions
          const filterConditions = buildFilterConditions(identity, helpers);

          // Add cursor condition for pagination only
          if (cursor) {
            return and(filterConditions, lt(identity.id, cursor));
          }

          return filterConditions;
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
        limit: limit + 1, // Fetch one extra to determine if there are more results
        orderBy: (identities, { desc }) => desc(identities.id),
      });

      // Determine if there are more results
      const hasMore = identitiesQuery.length > limit;

      // Remove the extra item if it exists
      const identities = hasMore ? identitiesQuery.slice(0, limit) : identitiesQuery;

      const transformedIdentities = identities.map((identity) => ({
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

      const lastId = identities.length > 0 ? identities[identities.length - 1].id : null;

      return {
        identities: transformedIdentities,
        hasMore,
        nextCursor: hasMore && lastId ? lastId : undefined,
        totalCount,
      };
    } catch (error) {
      console.error("Error retrieving identities:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve identities. If this issue persists, please contact support@unkey.dev with the time this occurred.",
      });
    }
  });
