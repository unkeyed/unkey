import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
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
      const permissionsQuery = await db.query.permissions.findMany({
        where: (permission, { and, eq, or, like }) => {
          return and(
            eq(permission.workspaceId, workspaceId),
            or(
              like(permission.id, query),
              like(permission.slug, query),
              like(permission.name, query),
              like(permission.description, query),
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
