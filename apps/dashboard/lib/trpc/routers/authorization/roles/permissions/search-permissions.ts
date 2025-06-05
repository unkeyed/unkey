import { db } from "@/lib/db";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import {
  LIMIT,
  PermissionsSearchResponse,
  permissionsSearchPayload,
  transformPermission,
} from "./schema-with-helpers";

export const searchRolesPermissions = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(permissionsSearchPayload)
  .output(PermissionsSearchResponse)
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
      const permissionsQuery = await db.query.permissions.findMany({
        where: (permission, { and, eq, or, like }) => {
          return and(
            eq(permission.workspaceId, workspaceId),
            or(
              like(permission.id, searchTerm),
              like(permission.slug, searchTerm),
              like(permission.name, searchTerm),
              like(permission.description, searchTerm),
            ),
          );
        },
        limit: LIMIT,
        orderBy: (permissions, { asc }) => [
          asc(permissions.name),
          asc(permissions.slug),
          asc(permissions.id),
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
          description: true,
          slug: true,
        },
      });

      return {
        permissions: permissionsQuery.map(transformPermission),
      };
    } catch (error) {
      console.error("Error searching permissions:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to search permissions. If this issue persists, please contact support.",
      });
    }
  });
