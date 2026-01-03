import { and, db, eq, inArray } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
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

export const getPermissionSlugs = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(resolvePermissionSlugsInput)
  .output(resolvePermissionSlugsResponse)
  .query(async ({ ctx, input }) => {
    const { roleIds, permissionIds } = input;
    const workspaceId = ctx.workspace.id;

    // Early return if no input
    if (roleIds.length === 0 && permissionIds.length === 0) {
      return {
        slugs: [],
        totalCount: 0,
        breakdown: { fromRoles: 0, fromDirectPermissions: 0 },
      };
    }

    try {
      let rolePermissionsPromise: Promise<PermissionSlug[]> = Promise.resolve([]);
      let directPermissionsPromise: Promise<PermissionSlug[]> = Promise.resolve([]);

      // Role permissions
      if (roleIds.length > 0) {
        rolePermissionsPromise = db
          .selectDistinct({ slug: permissions.slug })
          .from(rolesPermissions)
          .innerJoin(permissions, eq(rolesPermissions.permissionId, permissions.id))
          .where(
            and(
              inArray(rolesPermissions.roleId, roleIds),
              eq(rolesPermissions.workspaceId, workspaceId),
            ),
          );
      }

      // Direct permissions
      if (permissionIds.length > 0) {
        directPermissionsPromise = db
          .select({ slug: permissions.slug })
          .from(permissions)
          .where(
            and(inArray(permissions.id, permissionIds), eq(permissions.workspaceId, workspaceId)),
          );
      }

      const [rolePermissions, directPermissions] = await Promise.all([
        rolePermissionsPromise,
        directPermissionsPromise,
      ]);

      if (roleIds.length > 0 && rolePermissions.length === 0) {
        // Double-check if roles exist in workspace
        const roleExists = await db
          .select({ id: roles.id })
          .from(roles)
          .where(and(inArray(roles.id, roleIds), eq(roles.workspaceId, workspaceId)))
          .limit(1);

        if (roleExists.length === 0) {
          throw new TRPCError({
            code: "BAD_REQUEST",
            message: "One or more roles not found or access denied",
          });
        }
      }

      if (permissionIds.length > 0 && directPermissions.length === 0) {
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

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to resolve permission slugs",
        cause: error,
      });
    }
  });
