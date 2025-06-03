import { db } from "@/lib/db";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const LIMIT = 50;
const keysQueryPayload = z.object({
  cursor: z.string().optional(),
});

const RoleSchema = z.object({
  id: z.string(),
  name: z.string(),
});

const KeyResponseSchema = z.object({
  id: z.string(),
  name: z.string().nullable(),
  roles: z.array(RoleSchema),
});

const KeysResponse = z.object({
  keys: z.array(KeyResponseSchema),
  hasMore: z.boolean(),
  nextCursor: z.string().nullish(),
});

export const queryKeys = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(keysQueryPayload)
  .output(KeysResponse)
  .query(async ({ ctx, input }) => {
    const { cursor } = input;
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
        limit: LIMIT + 1, // Fetch one extra to determine if there are more results
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
      const hasMore = keysQuery.length > LIMIT;

      // Remove the extra item if it exists
      const keys = hasMore ? keysQuery.slice(0, LIMIT) : keysQuery;

      const transformedKeys = keys.map((key) => ({
        id: key.id,
        name: key.name,
        roles: key.roles
          .filter((keyRole) => keyRole.role !== null)
          .map((keyRole) => ({
            id: keyRole.role.id,
            name: keyRole.role.name,
          })),
      }));
      const nextCursor = hasMore && keys.length > 0 ? keys[keys.length - 1].id : undefined;

      return {
        keys: transformedKeys,
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
