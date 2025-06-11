import { updateKeyRbacSchema } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/components/edit-rbac/update-key-rbac.schema";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { requireUser, requireWorkspace, t } from "../../../trpc";

export const updateKeyRbac = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(updateKeyRbacSchema)
  .mutation(async ({ input, ctx }) => {
    const { keyId, roleIds, permissionIds } = input;
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
            "We were unable to update RBAC for this key. Please try again or contact support@unkey.dev",
        });
      });

    if (!key) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
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
            message: "Unable to validate roles. Please try again or contact support@unkey.dev",
          });
        });

      if (existingRoles.length !== roleIds.length) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "One or more roles do not exist in this workspace",
        });
      }
    }

    // Validate permissions exist in workspace
    if (permissionIds.length > 0) {
      const existingPermissions = await db.query.permissions
        .findMany({
          where: (table, { eq, and, inArray }) =>
            and(eq(table.workspaceId, workspaceId), inArray(table.id, permissionIds)),
          columns: { id: true },
        })
        .catch((_err) => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "Unable to validate permissions. Please try again or contact support@unkey.dev",
          });
        });

      if (existingPermissions.length !== permissionIds.length) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "One or more permissions do not exist in this workspace",
        });
      }
    }

    await db
      .transaction(async (tx) => {
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

        // Insert new permission assignments
        if (permissionIds.length > 0) {
          await tx
            .insert(schema.keysPermissions)
            .values(
              permissionIds.map((permissionId) => ({
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
          description: `Updated RBAC for key ${key.id}: ${roleIds.length} roles, ${permissionIds.length} permissions`,
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
            "We are unable to update RBAC for this key. Please try again or contact support@unkey.dev",
        });
      });

    return {
      keyId: key.id,
      success: true,
      rolesAssigned: roleIds.length,
      permissionsAssigned: permissionIds.length,
    };
  });
