import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import {
  PermissionsQueryResponse,
  permissionsQueryPayload,
  transformPermission,
} from "./schema-with-helpers";

export const queryRolesPermissions = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(permissionsQueryPayload)
  .output(PermissionsQueryResponse)
  .query(async ({ ctx, input }) => {
    const { cursor, limit } = input;
    const workspaceId = ctx.workspace.id;

    try {
      const permissionsQuery = await db.query.permissions.findMany({
        where: (permission, { and, eq, lt }) => {
          const conditions = [eq(permission.workspaceId, workspaceId)];
          if (cursor) {
            conditions.push(lt(permission.id, cursor));
          }
          return and(...conditions);
        },
        limit: limit + 1,
        orderBy: (permissions, { desc }) => desc(permissions.id),
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
          description: true,
          slug: true,
        },
      });

      const hasMore = permissionsQuery.length > limit;
      const permissions = hasMore ? permissionsQuery.slice(0, limit) : permissionsQuery;
      const nextCursor =
        hasMore && permissions.length > 0 ? permissions[permissions.length - 1].id : undefined;

      return {
        permissions: permissions.map(transformPermission),
        hasMore,
        nextCursor,
      };
    } catch (error) {
      console.error("Error retrieving permissions:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to retrieve permissions. If this issue persists, please contact support.",
      });
    }
  });
