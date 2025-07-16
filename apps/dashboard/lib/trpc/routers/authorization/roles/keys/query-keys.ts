import { and, db } from "@/lib/db";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { KeysResponse, keysQueryPayload, transformKey } from "./schema-with-helpers";

export const queryKeys = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(keysQueryPayload)
  .output(KeysResponse)
  .query(async ({ ctx, input }) => {
    const { cursor, limit } = input;
    const workspaceId = ctx.workspace.id;

    try {
      const apisQuery = await db.query.apis.findMany({
        where: (api, { eq, isNull }) =>
          and(eq(api.workspaceId, workspaceId), isNull(api.deletedAtM)),
        with: {
          keyAuth: {
            with: {
              keys: {
                where: (key, { and, lt, isNull }) => {
                  const conditions = [
                    isNull(key.deletedAtM), // Only non-deleted keys
                  ];

                  if (cursor) {
                    conditions.push(lt(key.id, cursor));
                  }

                  return and(...conditions);
                },
                limit: limit + 1,
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
              },
            },
          },
        },
      });

      const allKeys = apisQuery
        .flatMap((api) => api.keyAuth?.keys || [])
        .sort((a, b) => b.id.localeCompare(a.id));

      // Determine if there are more results
      const hasMore = allKeys.length > limit;

      // Remove the extra item if it exists
      const keys = hasMore ? allKeys.slice(0, limit) : allKeys;
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
