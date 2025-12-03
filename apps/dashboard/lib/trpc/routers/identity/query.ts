import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, requireWorkspace, t, withRatelimit } from "../../trpc";

const identitiesQueryPayload = z.object({
  cursor: z.string().optional(),
  limit: z.number().optional().default(50),
  search: z.string().optional(),
});

export const IdentityResponseSchema = z.object({
  id: z.string(),
  externalId: z.string(),
  workspaceId: z.string(),
  environment: z.string(),
  meta: z.record(z.unknown()).nullable(),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
  keys: z.array(z.object({ id: z.string() })),
  ratelimits: z.array(z.object({ id: z.string() })),
});

const IdentitiesResponse = z.object({
  identities: z.array(IdentityResponseSchema),
  hasMore: z.boolean(),
  nextCursor: z.string().nullish(),
});

export const queryIdentities = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(identitiesQueryPayload)
  .output(IdentitiesResponse)
  .query(async ({ ctx, input }) => {
    try {
      const { limit = 50, cursor, search } = input;
      const workspaceId = ctx.workspace.id;

      const identitiesQuery = await db.query.identities.findMany({
        where: (identity, { and, eq, lt, or, like }) => {
          const conditions = [eq(identity.workspaceId, workspaceId), eq(identity.deleted, false)];

          if (cursor) {
            conditions.push(lt(identity.id, cursor));
          }

          if (search) {
            const searchCondition = or(
              like(identity.externalId, `%${search}%`),
              like(identity.id, `%${search}%`),
            );
            if (searchCondition) {
              conditions.push(searchCondition);
            }
          }

          return and(...conditions);
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
