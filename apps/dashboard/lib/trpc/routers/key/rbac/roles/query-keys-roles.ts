import { db } from "@/lib/db";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { LIMIT, RolesResponse, rolesQueryPayload, transformRole } from "./schema-with-helpers";

export const queryKeysRoles = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(rolesQueryPayload)
  .output(RolesResponse)
  .query(async ({ ctx, input }) => {
    const { cursor } = input;
    const workspaceId = ctx.workspace.id;

    try {
      const rolesQuery = await db.query.roles.findMany({
        where: (role, { and, eq, lt }) => {
          const conditions = [eq(role.workspaceId, workspaceId)];

          if (cursor) {
            conditions.push(lt(role.id, cursor));
          }

          return and(...conditions);
        },
        limit: LIMIT + 1, // Fetch one extra to determine if there are more results
        orderBy: (roles, { desc }) => desc(roles.id),
        with: {
          keys: {
            with: {
              key: {
                columns: {
                  id: true,
                  name: true,
                },
              },
            },
          },
          permissions: {
            with: {
              permission: {
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
          description: true,
        },
      });

      // Determine if there are more results
      const hasMore = rolesQuery.length > LIMIT;
      // Remove the extra item if it exists
      const roles = hasMore ? rolesQuery.slice(0, LIMIT) : rolesQuery;
      const nextCursor = hasMore && roles.length > 0 ? roles[roles.length - 1].id : undefined;

      return {
        roles: roles.map(transformRole),
        hasMore,
        nextCursor,
      };
    } catch (error) {
      console.error("Error retrieving roles:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve roles. If this issue persists, please contact support.",
      });
    }
  });
