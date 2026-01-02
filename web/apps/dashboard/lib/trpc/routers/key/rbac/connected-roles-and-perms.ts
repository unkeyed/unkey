import { and, db, eq } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import {
  keys,
  keysPermissions,
  keysRoles,
  permissions,
  roles,
  rolesPermissions,
} from "@unkey/db/src/schema";
import { z } from "zod";

const keyDetailsInput = z.object({
  keyId: z.string().min(1, "Key ID is required"),
});

const keyRole = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().nullable(),
});
export type KeyRole = z.infer<typeof keyRole>;

const keyPermission = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  description: z.string().nullable(),
  source: z.enum(["direct", "role"]).optional(), // Track if permission comes from direct assignment or role
  roleId: z.string().optional(),
});
export type KeyPermission = z.infer<typeof keyPermission>;

const keyDetailsResponse = z.object({
  keyId: z.string(),
  name: z.string().nullable(),
  lastUpdated: z.number(),
  roles: z.array(keyRole),
  permissions: z.array(keyPermission),
});
export type KeyRbacDetails = z.infer<typeof keyDetailsResponse>;

export const getConnectedRolesAndPerms = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(keyDetailsInput)
  .output(keyDetailsResponse)
  .query(async ({ ctx, input }) => {
    const { keyId } = input;
    const workspaceId = ctx.workspace.id;

    try {
      // First, verify the key exists in this workspace - security check
      const keyResult = await db
        .select({
          id: keys.id,
          name: keys.name,
          updated_at_m: keys.updatedAtM,
        })
        .from(keys)
        .where(and(eq(keys.id, keyId), eq(keys.workspaceId, workspaceId)))
        .limit(1);

      if (keyResult.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Key not found or access denied",
        });
      }

      const key = keyResult[0];

      const [roleResults, directPermissionResults, rolePermissionResults] = await Promise.all([
        // Get roles directly assigned to the key
        db
          .selectDistinct({
            id: roles.id,
            name: roles.name,
            description: roles.description,
          })
          .from(keysRoles)
          .innerJoin(roles, eq(keysRoles.roleId, roles.id))
          .where(and(eq(keysRoles.keyId, keyId), eq(keysRoles.workspaceId, workspaceId)))
          .orderBy(roles.name),

        // Get permissions directly assigned to the key
        db
          .selectDistinct({
            id: permissions.id,
            name: permissions.name,
            slug: permissions.slug,
            description: permissions.description,
          })
          .from(keysPermissions)
          .innerJoin(permissions, eq(keysPermissions.permissionId, permissions.id))
          .where(
            and(eq(keysPermissions.keyId, keyId), eq(keysPermissions.workspaceId, workspaceId)),
          )
          .orderBy(permissions.name),

        // Get permissions inherited from roles
        db
          .selectDistinct({
            id: permissions.id,
            name: permissions.name,
            slug: permissions.slug,
            description: permissions.description,
            roleId: keysRoles.roleId,
          })
          .from(keysRoles)
          .innerJoin(rolesPermissions, eq(keysRoles.roleId, rolesPermissions.roleId))
          .innerJoin(permissions, eq(rolesPermissions.permissionId, permissions.id))
          .where(and(eq(keysRoles.keyId, keyId), eq(keysRoles.workspaceId, workspaceId)))
          .orderBy(permissions.name),
      ]);

      // Combine and dedup permissions
      const allPermissions = new Map<string, KeyPermission>();

      // Add direct permissions first
      directPermissionResults.forEach((perm) => {
        allPermissions.set(perm.id, {
          id: perm.id,
          name: perm.name,
          slug: perm.slug,
          description: perm.description,
          source: "direct",
        });
      });

      // Add role permissions (if not already direct)
      rolePermissionResults.forEach((perm) => {
        if (!allPermissions.has(perm.id)) {
          allPermissions.set(perm.id, {
            id: perm.id,
            name: perm.name,
            slug: perm.slug,
            description: perm.description,
            roleId: perm.roleId,
            source: "role",
          });
        }
      });

      return {
        keyId: key.id,
        name: key.name,
        lastUpdated: key.updated_at_m || Date.now(),
        roles: roleResults
          .map((row) => ({
            id: row.id,
            name: row.name,
            description: row.description,
          }))
          .sort((a, b) => a.name.localeCompare(b.name)),
        permissions: Array.from(allPermissions.values()).sort((a, b) =>
          a.name.localeCompare(b.name),
        ),
      };
    } catch (error) {
      // Re-throw TRPCErrors as-is
      if (error instanceof TRPCError) {
        throw error;
      }

      // Handle all other errors
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch key details",
        cause: error,
      });
    }
  });
