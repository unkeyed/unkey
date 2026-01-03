import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { KeysResponse, keysQueryPayload, transformKey } from "./schema-with-helpers";

export const queryKeys = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(keysQueryPayload)
  .output(KeysResponse)
  .query(async ({ ctx, input }) => {
    const { cursor, limit } = input;
    const workspaceId = ctx.workspace.id;

    try {
      const keysQuery = await db.query.keys.findMany({
        where: (key, { and, eq, lt, isNull }) => {
          const conditions = [
            eq(key.workspaceId, workspaceId),
            isNull(key.deletedAtM), // Only non-deleted keys
          ];

          if (cursor) {
            conditions.push(lt(key.id, cursor));
          }

          return and(...conditions);
        },
        limit: limit + 1, // Fetch one extra to determine if there are more results
        orderBy: (keys, { desc }) => desc(keys.id),
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

      // Determine if there are more results
      const hasMore = keysQuery.length > limit;

      // Remove the extra item if it exists
      const keys = hasMore ? keysQuery.slice(0, limit) : keysQuery;
      const nextCursor = hasMore && keys.length > 0 ? keys[keys.length - 1].id : undefined;

      return {
        keys: keys.map(transformKey),
        hasMore,
        nextCursor,
      };
    } catch (error) {
      console.error("Error retrieving keys:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve keys. If this issue persists, please contact support.",
      });
    }
  });
