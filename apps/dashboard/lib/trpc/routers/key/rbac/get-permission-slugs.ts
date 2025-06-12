import { and, db, eq, inArray } from "@/lib/db";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { permissions, roles, rolesPermissions } from "@unkey/db/src/schema";
import { z } from "zod";

const resolvePermissionSlugsInput = z.object({
  roleIds: z.array(z.string()).default([]),
  permissionIds: z.array(z.string()).default([]),
});

const resolvePermissionSlugsResponse = z.object({
  slugs: z.array(z.string()),
  totalCount: z.number(),
  breakdown: z.object({
    fromRoles: z.number(),
    fromDirectPermissions: z.number(),
  }),
});

type PermissionSlug = {
  slug: string;
};

export const getPermissionSlugs = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(resolvePermissionSlugsInput)
  .output(resolvePermissionSlugsResponse)
  .query(async ({ ctx, input }) => {
    const { roleIds, permissionIds } = input;
    const workspaceId = ctx.workspace.id;

    try {
      // Role permissions query
      let rolePermissionsPromise: Promise<PermissionSlug[]> = Promise.resolve([]);
      if (roleIds.length > 0) {
        rolePermissionsPromise = db
          .selectDistinct({
            slug: permissions.slug,
          })
          .from(rolesPermissions)
          .innerJoin(permissions, eq(rolesPermissions.permissionId, permissions.id))
          .innerJoin(roles, eq(rolesPermissions.roleId, roles.id))
          .where(
            and(
              inArray(rolesPermissions.roleId, roleIds),
              eq(rolesPermissions.workspaceId, workspaceId),
              eq(roles.workspaceId, workspaceId),
            ),
          );
      }

      // Direct permissions query
      let directPermissionsPromise: Promise<PermissionSlug[]> = Promise.resolve([]);
      if (permissionIds.length > 0) {
        directPermissionsPromise = db
          .select({
            slug: permissions.slug,
          })
          .from(permissions)
          .where(
            and(inArray(permissions.id, permissionIds), eq(permissions.workspaceId, workspaceId)),
          );
      }

      const [rolePermissions, directPermissions] = await Promise.all([
        rolePermissionsPromise,
        directPermissionsPromise,
      ]);

      // Validate that all requested items were found
      if (roleIds.length > 0) {
        const roleCheck = await db
          .select({ id: roles.id })
          .from(roles)
          .where(and(inArray(roles.id, roleIds), eq(roles.workspaceId, workspaceId)));

        if (roleCheck.length !== roleIds.length) {
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: "One or more roles not found or access denied",
          });
        }
      }

      if (permissionIds.length > 0 && directPermissions.length !== permissionIds.length) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "One or more permissions not found or access denied",
        });
      }

      const slugsSet = new Set<string>();

      rolePermissions.forEach(({ slug }) => slugsSet.add(slug));
      directPermissions.forEach(({ slug }) => slugsSet.add(slug));

      const allSlugs = Array.from(slugsSet).sort();

      return {
        slugs: allSlugs,
        totalCount: allSlugs.length,
        breakdown: {
          fromRoles: rolePermissions.length,
          fromDirectPermissions: directPermissions.length,
        },
      };
    } catch (error) {
      // Re-throw TRPCErrors as-is
      if (error instanceof TRPCError) {
        throw error;
      }

      // Handle database connection errors
      if (error instanceof Error && error.message.includes("connection")) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Database connection failed",
        });
      }

      // Handle all other errors
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to resolve permission slugs",
        cause: error,
      });
    }
  });
