import { db } from "@/lib/db";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const LIMIT = 50;

const permissionsQueryPayload = z.object({
  cursor: z.string().optional(),
});

const RoleSchema = z.object({
  id: z.string(),
  name: z.string(),
});

const PermissionResponseSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  roles: z.array(RoleSchema),
});

const PermissionsResponse = z.object({
  permissions: z.array(PermissionResponseSchema),
  hasMore: z.boolean(),
  nextCursor: z.string().nullish(),
});

export const queryRolesPermissions = t.procedure
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(permissionsQueryPayload)
  .output(PermissionsResponse)
  .query(async ({ ctx, input }) => {
    const { cursor } = input;
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
        limit: LIMIT + 1, // Fetch one extra to determine if there are more results
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
        },
      });

      // Determine if there are more results
      const hasMore = permissionsQuery.length > LIMIT;
      // Remove the extra item if it exists
      const permissions = hasMore ? permissionsQuery.slice(0, LIMIT) : permissionsQuery;

      const transformedPermissions = permissions.map((permission) => ({
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

      const nextCursor =
        hasMore && permissions.length > 0 ? permissions[permissions.length - 1].id : undefined;

      return {
        permissions: transformedPermissions,
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
