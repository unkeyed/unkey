import { and, db, eq } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { keys, keysRoles, permissions, roles, rolesPermissions } from "@unkey/db/src/schema";
import { z } from "zod";

const roleDetailsInput = z.object({
  roleId: z.string().min(1, "Role ID is required"),
});

const roleKey = z.object({
  id: z.string(),
  name: z.string().nullable(),
});
export type RoleKey = z.infer<typeof roleKey>;

const rolePermission = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  description: z.string().nullable(),
});
export type RolePermission = z.infer<typeof rolePermission>;

const roleDetailsResponse = z.object({
  roleId: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  lastUpdated: z.number(),
  keys: z.array(roleKey),
  permissions: z.array(rolePermission),
});

export type RoleDetails = z.infer<typeof roleDetailsResponse>;

export const getConnectedKeysAndPerms = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(roleDetailsInput)
  .output(roleDetailsResponse)
  .query(async ({ ctx, input }) => {
    const { roleId } = input;
    const workspaceId = ctx.workspace.id;

    try {
      // First, verify the role exists in this workspace - security check
      const roleResult = await db
        .select({
          id: roles.id,
          name: roles.name,
          description: roles.description,
          updated_at_m: roles.updatedAtM,
          created_at_m: roles.createdAtM,
        })
        .from(roles)
        .where(and(eq(roles.id, roleId), eq(roles.workspaceId, workspaceId)))
        .limit(1);

      if (roleResult.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Role not found or access denied",
        });
      }

      const role = roleResult[0];
      const [keyResults, permissionResults] = await Promise.all([
        db
          .selectDistinct({
            id: keys.id,
            name: keys.name,
          })
          .from(keysRoles)
          .innerJoin(keys, eq(keysRoles.keyId, keys.id))
          .where(and(eq(keysRoles.roleId, roleId), eq(keysRoles.workspaceId, workspaceId)))
          .orderBy(keys.name),

        db
          .selectDistinct({
            id: permissions.id,
            name: permissions.name,
            slug: permissions.slug,
            description: permissions.description,
          })
          .from(rolesPermissions)
          .innerJoin(permissions, eq(rolesPermissions.permissionId, permissions.id))
          .where(
            and(eq(rolesPermissions.roleId, roleId), eq(rolesPermissions.workspaceId, workspaceId)),
          )
          .orderBy(permissions.name),
      ]);

      return {
        roleId: role.id,
        name: role.name,
        description: role.description,
        lastUpdated: Number(role.updated_at_m ?? role.created_at_m),
        keys: keyResults.map((row) => {
          return {
            id: row.id,
            name: row.name,
          };
        }),
        permissions: permissionResults.map((row) => {
          return {
            id: row.id,
            name: row.name,
            slug: row.slug,
            description: row.description,
          };
        }),
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
        message: "Failed to fetch role details",
        cause: error,
      });
    }
  });
