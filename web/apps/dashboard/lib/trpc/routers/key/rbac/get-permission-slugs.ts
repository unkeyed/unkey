import { and, db, eq, inArray } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { permissions, roles, rolesPermissions } from "@unkey/db/src/schema";
import { z } from "zod";

const resolvePermissionSlugsInput = z.object({
  roleNames: z.array(z.string()).prefault([]),
  permissionSlugs: z.array(z.string()).prefault([]),
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
    const roleNames = Array.from(new Set(input.roleNames));
    const permissionSlugs = Array.from(new Set(input.permissionSlugs));
    const workspaceId = ctx.workspace.id;

    if (roleNames.length === 0 && permissionSlugs.length === 0) {
      return {
        slugs: [],
        totalCount: 0,
        breakdown: { fromRoles: 0, fromDirectPermissions: 0 },
      };
    }

    try {
      let rolePermissionsPromise: Promise<PermissionSlug[]> = Promise.resolve([]);
      let directPermissionsPromise: Promise<PermissionSlug[]> = Promise.resolve([]);
      let roleNamesPromise: Promise<Array<{ name: string }>> = Promise.resolve([]);

      if (roleNames.length > 0) {
        roleNamesPromise = db
          .select({ name: roles.name })
          .from(roles)
          .where(and(inArray(roles.name, roleNames), eq(roles.workspaceId, workspaceId)));

        rolePermissionsPromise = db
          .selectDistinct({ slug: permissions.slug })
          .from(rolesPermissions)
          .innerJoin(roles, eq(rolesPermissions.roleId, roles.id))
          .innerJoin(permissions, eq(rolesPermissions.permissionId, permissions.id))
          .where(
            and(
              inArray(roles.name, roleNames),
              eq(rolesPermissions.workspaceId, workspaceId),
              eq(roles.workspaceId, workspaceId),
              eq(permissions.workspaceId, workspaceId),
            ),
          );
      }

      if (permissionSlugs.length > 0) {
        directPermissionsPromise = db
          .select({ slug: permissions.slug })
          .from(permissions)
          .where(
            and(
              inArray(permissions.slug, permissionSlugs),
              eq(permissions.workspaceId, workspaceId),
            ),
          );
      }

      const [rolePermissions, directPermissions, roleRows] = await Promise.all([
        rolePermissionsPromise,
        directPermissionsPromise,
        roleNamesPromise,
      ]);

      if (roleNames.length > 0 && roleRows.length !== roleNames.length) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "One or more roles not found or access denied",
        });
      }

      if (
        permissionSlugs.length > 0 &&
        directPermissions.length !== permissionSlugs.length
      ) {
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
