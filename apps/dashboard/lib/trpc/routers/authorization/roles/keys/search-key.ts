import { db } from "@/lib/db";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const LIMIT = 50;
const keysSearchPayload = z.object({
  query: z.string().min(1, "Search query cannot be empty"),
});

const RoleSchema = z.object({
  id: z.string(),
  name: z.string(),
});

const KeySearchResponseSchema = z.object({
  id: z.string(),
  name: z.string().nullable(),
  roles: z.array(RoleSchema),
});

const KeysSearchResponse = z.object({
  keys: z.array(KeySearchResponseSchema),
});

export const searchKeys = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(keysSearchPayload)
  .output(KeysSearchResponse)
  .query(async ({ ctx, input }) => {
    const { query } = input;
    const workspaceId = ctx.workspace.id;

    if (!query.trim()) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "Search query cannot be empty",
      });
    }

    try {
      const searchTerm = `%${query.trim()}%`;

      const keysQuery = await db.query.keys.findMany({
        where: (key, { and, eq, or, like, isNull }) => {
          return and(
            eq(key.workspaceId, workspaceId),
            isNull(key.deletedAtM), // Only non-deleted keys
            or(like(key.id, searchTerm), like(key.name, searchTerm)),
          );
        },
        limit: LIMIT,
        orderBy: (keys, { asc }) => [
          asc(keys.name), // Name matches first
          asc(keys.id), // Then by ID for consistency
        ],
        with: {
          roles: {
            with: {
              role: {
                columns: {
                  id: true,
                  name: true,
                },
              },
            },
          },
        },
        columns: {
          id: true,
          name: true,
        },
      });

      const transformedKeys = keysQuery.map((key) => ({
        id: key.id,
        name: key.name,
        roles: key.roles.map((keyRole) => ({
          id: keyRole.role.id,
          name: keyRole.role.name,
        })),
      }));

      return {
        keys: transformedKeys,
      };
    } catch (error) {
      console.error("Error searching keys:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to search keys. If this issue persists, please contact support.",
      });
    }
  });
