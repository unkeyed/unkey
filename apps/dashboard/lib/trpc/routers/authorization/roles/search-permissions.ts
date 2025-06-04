import { db } from "@/lib/db";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const LIMIT = 50;

const permissionsSearchPayload = z.object({
  query: z.string().min(1, "Search query cannot be empty"),
});

const RoleSchema = z.object({
  id: z.string(),
  name: z.string(),
});

const PermissionSearchResponseSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  roles: z.array(RoleSchema),
});

const PermissionsSearchResponse = z.object({
  permissions: z.array(PermissionSearchResponseSchema),
});

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
              like(permission.name, searchTerm),
              like(permission.description, searchTerm),
            ),
          );
        },
        limit: LIMIT,
        orderBy: (permissions, { asc }) => [
          asc(permissions.name), // Name matches first
          asc(permissions.id), // Then by ID for consistency
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
        },
      });

      const transformedPermissions = permissionsQuery.map((permission) => ({
        id: permission.id,
        name: permission.name,
        description: permission.description,
        roles: permission.roles
          .filter((rolePermission) => rolePermission.role !== null)
          .map((rolePermission) => ({
            id: rolePermission.role.id,
            name: rolePermission.role.name,
          })),
      }));

      return {
        permissions: transformedPermissions,
      };
    } catch (error) {
      console.error("Error searching permissions:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to search permissions. If this issue persists, please contact support.",
      });
    }
  });
