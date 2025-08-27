import { rbacRoleSchema } from "@/app/(app)/[workspace]/authorization/roles/components/upsert-role/upsert-role.schema";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { requireUser, requireWorkspace, t } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";

export const upsertRole = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(rbacRoleSchema)
  .mutation(async ({ input, ctx }) => {
    const isUpdate = Boolean(input.roleId);
    let roleId = input.roleId;

    if (!isUpdate) {
      roleId = newId("role");
      if (!roleId) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to generate role ID",
        });
      }
    }

    if (!roleId) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Invalid role ID",
      });
    }

    await db.transaction(async (tx) => {
      if (isUpdate && input.roleId) {
        const updateRoleId: string = input.roleId;

        // Get the existing role to compare names and verify existence
        const existingRole = await tx.query.roles.findFirst({
          where: (table, { and, eq }) =>
            and(eq(table.id, updateRoleId), eq(table.workspaceId, ctx.workspace.id)),
        });

        if (!existingRole) {
          throw new TRPCError({
            code: "NOT_FOUND",
            message: "Role not found or access denied",
          });
        }

        // Only check for name conflicts if the name is actually changing
        if (existingRole.name !== input.roleName) {
          const nameConflict = await tx.query.roles.findFirst({
            where: (table, { and, eq, ne }) =>
              and(
                eq(table.workspaceId, ctx.workspace.id),
                eq(table.name, input.roleName),
                ne(table.id, updateRoleId),
              ),
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
        });
      } else {
        // Create mode - always check for name conflicts
        const nameConflict = await tx.query.roles.findFirst({
          where: (table, { and, eq }) =>
            and(eq(table.workspaceId, ctx.workspace.id), eq(table.name, input.roleName)),
        });

        if (nameConflict) {
          throw new TRPCError({
            code: "CONFLICT",
            message: `Role with name '${input.roleName}' already exists`,
          });
        }

        // Create new role
        await tx
          .insert(schema.roles)
          .values({
            id: roleId,
            name: input.roleName,
            description: input.roleDescription,
            workspaceId: ctx.workspace.id,
          })
          .catch(() => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message: "Failed to create role",
            });
          });

        // For creation, treat undefined as empty array (no associations initially)
        const permissionIds = input.permissionIds ?? [];
        const keyIds = input.keyIds ?? [];

        // Add role-permission relationships
        if (permissionIds.length > 0) {
          await tx
            .insert(schema.rolesPermissions)
            .values(
              permissionIds.map((permissionId) => ({
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
            permissionIds.map((permissionId) => ({
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
            })),
          );
        }

        // Add key-role relationships
        if (keyIds.length > 0) {
          await tx
            .insert(schema.keysRoles)
            .values(
              keyIds.map((keyId) => ({
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
            keyIds.map((keyId) => ({
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
            })),
          );
        }

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          event: "role.create",
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          description: `Created role ${roleId}`,
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
        });
      }
    });

    return {
      roleId,
      isUpdate,
      message: isUpdate ? "Role updated successfully" : "Role created successfully",
    };
  });
