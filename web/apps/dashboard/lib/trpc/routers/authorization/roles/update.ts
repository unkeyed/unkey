import { rbacRoleSchema } from "@/app/(app)/[workspaceSlug]/authorization/roles/components/upsert-role/upsert-role.schema";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import type { Transaction } from "@unkey/db";
import { newId } from "@unkey/id";
import { z } from "zod";

// Without these checks the caller could attach a role to keys or permissions
// owned by another workspace, which leaks role permissions into a victim's
// keys at verification time (the Go authoritative join chain has no
// workspace filter).
async function assertKeysInWorkspace(tx: Transaction, workspaceId: string, keyIds: string[]) {
  if (keyIds.length === 0) {
    return;
  }
  const found = await tx.query.keys.findMany({
    where: (table, { and, eq, inArray }) =>
      and(eq(table.workspaceId, workspaceId), inArray(table.id, keyIds)),
    columns: { id: true },
  });
  const foundIds = new Set(found.map((k) => k.id));
  const missing = keyIds.filter((id) => !foundIds.has(id));
  if (missing.length > 0) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: `Key(s) not found or not in this workspace: ${missing.join(", ")}`,
    });
  }
}

async function assertPermissionsInWorkspace(
  tx: Transaction,
  workspaceId: string,
  permissionIds: string[],
) {
  if (permissionIds.length === 0) {
    return;
  }
  const found = await tx.query.permissions.findMany({
    where: (table, { and, eq, inArray }) =>
      and(eq(table.workspaceId, workspaceId), inArray(table.id, permissionIds)),
    columns: { id: true },
  });
  const foundIds = new Set(found.map((p) => p.id));
  const missing = permissionIds.filter((id) => !foundIds.has(id));
  if (missing.length > 0) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: `Permission(s) not found or not in this workspace: ${missing.join(", ")}`,
    });
  }
}

export const updateRole = workspaceProcedure
  .input(rbacRoleSchema.extend({ roleId: z.string().startsWith("role_") }))
  .mutation(async ({ input, ctx }) => {
    const roleId = input.roleId;

    // Mint a shared correlation so role.update + N permission binds + N key
    // binds from the multiple insertAuditLogs calls below all link to one user
    // action in the audit log drill-down.
    const correlationId = newId("correlation");

    await db.transaction(async (tx) => {
      // Get the existing role to compare names and verify existence
      const existingRole = await tx.query.roles.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.id, roleId), eq(table.workspaceId, ctx.workspace.id)),
        columns: {
          id: true,
          name: true,
        },
      });

      if (!existingRole) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Role not found or access denied",
        });
      }

      if (input.permissionIds && input.permissionIds.length > 0) {
        await assertPermissionsInWorkspace(tx, ctx.workspace.id, input.permissionIds);
      }
      if (input.keyIds && input.keyIds.length > 0) {
        await assertKeysInWorkspace(tx, ctx.workspace.id, input.keyIds);
      }

      // Only check for name conflicts if the name is actually changing
      if (existingRole.name !== input.roleName) {
        const nameConflict = await tx.query.roles.findFirst({
          where: (table, { and, eq, ne }) =>
            and(
              eq(table.workspaceId, ctx.workspace.id),
              eq(table.name, input.roleName),
              ne(table.id, roleId),
            ),
          columns: { id: true },
        });

        if (nameConflict) {
          throw new TRPCError({
            code: "CONFLICT",
            message: `Role with name '${input.roleName}' already exists`,
          });
        }
      }

      // Update role
      await tx
        .update(schema.roles)
        .set({
          name: input.roleName,
          description: input.roleDescription,
        })
        .where(and(eq(schema.roles.id, roleId), eq(schema.roles.workspaceId, ctx.workspace.id)))
        .catch(() => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "Failed to update role",
          });
        });

      // Handle permissions - only modify if explicitly provided
      if (input.permissionIds !== undefined) {
        // Remove existing role-permission relationships
        await tx
          .delete(schema.rolesPermissions)
          .where(
            and(
              eq(schema.rolesPermissions.roleId, roleId),
              eq(schema.rolesPermissions.workspaceId, ctx.workspace.id),
            ),
          );

        // Add new permissions if any
        if (input.permissionIds.length > 0) {
          await tx
            .insert(schema.rolesPermissions)
            .values(
              input.permissionIds.map((permissionId) => ({
                permissionId,
                roleId,
                workspaceId: ctx.workspace.id,
              })),
            )
            .catch(() => {
              throw new TRPCError({
                code: "INTERNAL_SERVER_ERROR",
                message: "Failed to assign permissions to role",
              });
            });

          await insertAuditLogs(
            tx,
            input.permissionIds.map((permissionId) => ({
              workspaceId: ctx.workspace.id,
              event: "authorization.connect_role_and_permission",
              actor: {
                type: "user",
                id: ctx.user.id,
              },
              description: `Connected role ${roleId} and permission ${permissionId}`,
              resources: [
                { type: "role", id: roleId, name: input.roleName },
                { type: "permission", id: permissionId },
              ],
              context: {
                userAgent: ctx.audit.userAgent,
                location: ctx.audit.location,
              },
              correlationId,
            })),
          );
        }
      }

      // Handle keys - only modify if explicitly provided
      if (input.keyIds !== undefined) {
        // Remove existing key-role relationships
        await tx
          .delete(schema.keysRoles)
          .where(
            and(
              eq(schema.keysRoles.roleId, roleId),
              eq(schema.keysRoles.workspaceId, ctx.workspace.id),
            ),
          );

        // Add new keys if any
        if (input.keyIds.length > 0) {
          await tx
            .insert(schema.keysRoles)
            .values(
              input.keyIds.map((keyId) => ({
                keyId,
                roleId,
                workspaceId: ctx.workspace.id,
              })),
            )
            .catch(() => {
              throw new TRPCError({
                code: "INTERNAL_SERVER_ERROR",
                message: "Failed to assign keys to role",
              });
            });

          await insertAuditLogs(
            tx,
            input.keyIds.map((keyId) => ({
              workspaceId: ctx.workspace.id,
              event: "authorization.connect_role_and_key",
              actor: {
                type: "user",
                id: ctx.user.id,
              },
              description: `Connected key ${keyId} and role ${roleId}`,
              resources: [
                { type: "key", id: keyId },
                { type: "role", id: roleId, name: input.roleName },
              ],
              context: {
                userAgent: ctx.audit.userAgent,
                location: ctx.audit.location,
              },
              correlationId,
            })),
          );
        }
      }

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        event: "role.update",
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        description: `Updated role ${roleId}`,
        resources: [
          {
            type: "role",
            id: roleId,
            name: input.roleName,
          },
        ],
        context: {
          userAgent: ctx.audit.userAgent,
          location: ctx.audit.location,
        },
        correlationId,
      });
    });

    return { roleId, message: "Role updated successfully" };
  });
