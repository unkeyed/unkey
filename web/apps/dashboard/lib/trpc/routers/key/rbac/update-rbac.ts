import { updateKeyRbacSchema } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/components/edit-rbac/update-key-rbac.schema";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../../trpc";

export const updateKeyRbac = workspaceProcedure
  .input(updateKeyRbacSchema)
  .mutation(async ({ input, ctx }) => {
    const { keyId, roleIds, directPermissionIds } = input;
    const workspaceId = ctx.workspace.id;

    // Verify key exists and belongs to workspace
    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, isNull, and }) =>
          and(eq(table.workspaceId, workspaceId), eq(table.id, keyId), isNull(table.deletedAtM)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update RBAC for this key. Please try again or contact support@unkey.com",
        });
      });

    if (!key) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please try again or contact support@unkey.com.",
        code: "NOT_FOUND",
      });
    }

    // Validate roles exist in workspace
    if (roleIds.length > 0) {
      const existingRoles = await db.query.roles
        .findMany({
          where: (table, { eq, and, inArray }) =>
            and(eq(table.workspaceId, workspaceId), inArray(table.id, roleIds)),
          columns: { id: true },
        })
        .catch((_err) => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Unable to validate roles. Please try again or contact support@unkey.com",
          });
        });

      if (existingRoles.length !== roleIds.length) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "One or more roles do not exist in this workspace",
        });
      }
    }

    // Validate direct permissions exist in workspace
    if (directPermissionIds.length > 0) {
      const existingPermissions = await db.query.permissions
        .findMany({
          where: (table, { eq, and, inArray }) =>
            and(eq(table.workspaceId, workspaceId), inArray(table.id, directPermissionIds)),
          columns: { id: true },
        })
        .catch((_err) => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "Unable to validate permissions. Please try again or contact support@unkey.com",
          });
        });

      if (existingPermissions.length !== directPermissionIds.length) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "One or more permissions do not exist in this workspace",
        });
      }
    }

    // Calculate total effective permissions for response
    let totalEffectivePermissions = directPermissionIds.length;

    await db
      .transaction(async (tx) => {
        // Get permissions that come from the requested roles for audit/response purposes
        const rolePermissions =
          roleIds.length > 0
            ? await tx.query.rolesPermissions
                .findMany({
                  where: (table, { inArray, eq, and }) =>
                    and(inArray(table.roleId, roleIds), eq(table.workspaceId, workspaceId)),
                  columns: { permissionId: true },
                })
                .catch((_err) => {
                  throw new TRPCError({
                    code: "INTERNAL_SERVER_ERROR",
                    message:
                      "Unable to resolve role permissions. Please try again or contact support@unkey.com",
                  });
                })
            : [];

        // Calculate unique role permissions for total count
        const uniqueRolePermissionIds = new Set(rolePermissions.map((rp) => rp.permissionId));

        // Add role permissions to total, avoiding double-counting
        directPermissionIds.forEach((id) => {
          if (!uniqueRolePermissionIds.has(id)) {
            uniqueRolePermissionIds.add(id);
          }
        });

        totalEffectivePermissions =
          uniqueRolePermissionIds.size +
          directPermissionIds.filter((id) => !uniqueRolePermissionIds.has(id)).length;

        // Remove existing role assignments
        await tx
          .delete(schema.keysRoles)
          .where(
            and(eq(schema.keysRoles.keyId, keyId), eq(schema.keysRoles.workspaceId, workspaceId)),
          )
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message: "Failed to remove existing role assignments",
            });
          });

        // Remove existing permission assignments
        await tx
          .delete(schema.keysPermissions)
          .where(
            and(
              eq(schema.keysPermissions.keyId, keyId),
              eq(schema.keysPermissions.workspaceId, workspaceId),
            ),
          )
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message: "Failed to remove existing permission assignments",
            });
          });

        // Insert new role assignments
        if (roleIds.length > 0) {
          await tx
            .insert(schema.keysRoles)
            .values(
              roleIds.map((roleId) => ({
                keyId,
                roleId,
                workspaceId,
              })),
            )
            .catch((_err) => {
              throw new TRPCError({
                code: "INTERNAL_SERVER_ERROR",
                message: "Failed to assign new roles",
              });
            });
        }

        // Insert direct permission assignments
        if (directPermissionIds.length > 0) {
          await tx
            .insert(schema.keysPermissions)
            .values(
              directPermissionIds.map((permissionId) => ({
                keyId,
                permissionId,
                workspaceId,
              })),
            )
            .catch((_err) => {
              throw new TRPCError({
                code: "INTERNAL_SERVER_ERROR",
                message: "Failed to assign new permissions",
              });
            });
        }

        // Audit log
        await insertAuditLogs(tx, {
          workspaceId,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "key.update",
          description: `Updated RBAC for key ${key.id}: ${roleIds.length} roles, ${directPermissionIds.length} direct permissions (${totalEffectivePermissions} total effective permissions)`,
          resources: [
            {
              type: "key",
              id: key.id,
              name: key.name || undefined,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })
      .catch((err) => {
        if (err instanceof TRPCError) {
          throw err;
        }
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update RBAC for this key. Please try again or contact support@unkey.com",
        });
      });

    return {
      keyId: key.id,
      success: true,
      rolesAssigned: roleIds.length,
      directPermissionsAssigned: directPermissionIds.length,
      totalEffectivePermissions,
    };
  });
