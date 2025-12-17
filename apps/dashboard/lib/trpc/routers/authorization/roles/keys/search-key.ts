import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { KeysSearchResponse, LIMIT, keysSearchPayload, transformKey } from "./schema-with-helpers";

export const searchKeys = workspaceProcedure
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

      return {
        keys: keysQuery.map(transformKey),
      };
    } catch (error) {
      console.error("Error searching keys:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to search keys. If this issue persists, please contact support.",
      });
    }
  });
