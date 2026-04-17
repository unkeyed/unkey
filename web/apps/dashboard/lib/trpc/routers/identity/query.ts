import { identitiesQueryPayload } from "@/components/identities-table/schema/identities.schema";
import { and, count, db, eq, like, or, schema } from "@/lib/db";
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
        limit,
        offset,
        orderBy: (identities, { asc, desc }) => {
          const direction = sortOrder === "asc" ? asc : desc;
          switch (sortBy) {
            case "externalId": {
              return direction(identities.externalId);
            }
            default: {
              return direction(identities.createdAt);
            }
          }
        },
      });

      const transformedIdentities = identitiesQuery.map((identity) => ({
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
