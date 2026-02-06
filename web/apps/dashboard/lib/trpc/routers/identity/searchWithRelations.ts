import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../trpc";

const SearchWithRelationsInput = z.object({
  search: z.string().optional().prefault(""),
  limit: z.number().optional().prefault(50),
});

const IdentityWithRelationsSchema = z.object({
  id: z.string(),
  externalId: z.string(),
  meta: z.record(z.string(), z.unknown()).nullable(),
  ratelimits: z.array(
    z.object({
      id: z.string(),
    }),
  ),
  keys: z.array(
    z.object({
      id: z.string(),
    }),
  ),
});

const WorkspaceWithIdentitiesSchema = z.object({
  id: z.string(),
  slug: z.string(),
  identities: z.array(IdentityWithRelationsSchema),
});

export const searchIdentitiesWithRelations = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(SearchWithRelationsInput)
  .output(WorkspaceWithIdentitiesSchema)
  .query(async ({ ctx, input }) => {
    const { search, limit } = input;
    const workspaceId = ctx.workspace.id;

    try {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.id, workspaceId), isNull(table.deletedAtM)),
        with: {
          identities: {
            where: (table, { or, like, and, eq }) => {
              const deletedFilter = eq(table.deleted, false);
              const searchFilter = search
                ? or(like(table.externalId, `%${search}%`), like(table.id, `%${search}%`))
                : undefined;
              return searchFilter ? and(deletedFilter, searchFilter) : deletedFilter;
            },
            limit,
            orderBy: (table, { asc }) => asc(table.id),
            with: {
              ratelimits: {
                columns: {
                  id: true,
                },
              },
              keys: {
                columns: {
                  id: true,
                },
              },
            },
          },
        },
      });

      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Workspace not found",
        });
      }

      // If we have a search term, prioritize exact matches
      if (search) {
        const exactMatchIndex = workspace.identities.findIndex(
          ({ id, externalId }) => search === id || search === externalId,
        );
        if (exactMatchIndex > 0) {
          workspace.identities.unshift(workspace.identities.splice(exactMatchIndex, 1)[0]);
        }
      }

      return {
        id: workspace.id,
        slug: workspace.slug,
        identities: workspace.identities.map((identity) => ({
          id: identity.id,
          externalId: identity.externalId,
          meta: identity.meta,
          ratelimits: identity.ratelimits,
          keys: identity.keys,
        })),
      };
    } catch (error) {
      console.error("Error searching identities with relations:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to search identities. If this issue persists, please contact support@unkey.com with the time this occurred.",
      });
    }
  });
