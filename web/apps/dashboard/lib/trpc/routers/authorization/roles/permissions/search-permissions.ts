import { TRPCError } from "@trpc/server";
import { db, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import {
  LIMIT,
  PermissionsSearchResponse,
  permissionsSearchPayload,
  transformPermission,
} from "./schema-with-helpers";

export const searchRolesPermissions = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(permissionsSearchPayload)
  .output(PermissionsSearchResponse)
  .query(async ({ ctx, input }) => {
    const { query } = input;
    const workspaceId = ctx.workspace.id;

    try {
      const searchTerm = `%${query}%`;

      const permissionsQuery = await db.query.permissions.findMany({
        where: (permission, { and, eq, or }) => {
          return and(
            eq(permission.workspaceId, workspaceId),
            or(
              sql`LOWER(${permission.id}) LIKE LOWER(${searchTerm})`,
              sql`LOWER(${permission.slug}) LIKE LOWER(${searchTerm})`,
              sql`LOWER(${permission.name}) LIKE LOWER(${searchTerm})`,
              sql`LOWER(${permission.description}) LIKE LOWER(${searchTerm})`,
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
            columns: { roleId: true },
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
