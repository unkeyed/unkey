import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, requireWorkspace, t, withRatelimit } from "../../trpc";
import { IdentityResponseSchema } from "./query";

const LIMIT = 5;

const SearchIdentitiesResponse = z.object({
  identities: z.array(IdentityResponseSchema),
});

export const searchIdentities = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      query: z
        .string()
        .trim()
        .min(1, "Search query is required")
        .max(255, "Search query is too long"),
    }),
  )
  .output(SearchIdentitiesResponse)
  .query(async ({ ctx, input }) => {
    const { query } = input;
    const workspaceId = ctx.workspace.id;

    try {
      const identitiesQuery = await db.query.identities.findMany({
        where: (identity, { and, eq, like }) => {
          return and(
            eq(identity.workspaceId, workspaceId),
            eq(identity.deleted, false),
            like(identity.externalId, `%${query}%`),
          );
        },
        limit: LIMIT,
        orderBy: (identities, { asc }) => [asc(identities.externalId)],
        columns: {
          id: true,
          externalId: true,
          workspaceId: true,
          environment: true,
          meta: true,
          createdAt: true,
          updatedAt: true,
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
      }));

      return {
        identities: transformedIdentities,
      };
    } catch (error) {
      console.error("Error searching identities:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to search identities. If this issue persists, please contact support@unkey.dev with the time this occurred.",
      });
    }
  });
