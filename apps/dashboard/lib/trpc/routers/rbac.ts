import { type Permission, and, db, eq, schema } from "@/lib/db";

import { insertAuditLogs } from "@/lib/audit";
import type { UnkeyAuditLog } from "@/lib/tinybird";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";
import type { Context } from "../context";
import { t } from "../trpc";

const nameSchema = z
  .string()
  .min(3)
  .regex(/^[a-zA-Z0-9_:\-\.\*]+$/, {
    message:
      "Must be at least 3 characters long and only contain alphanumeric, colons, periods, dashes and underscores",
  });

export const rbacRouter = t.router({
  addPermissionToRootKey: rateLimitedProcedure(ratelimit.update)
    .input(
      z.object({
        rootKeyId: z.string(),
        permission: z.string(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const permission = unkeyPermissionValidation.safeParse(input.permission);
      if (!permission.success) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: `invalid permission [${input.permission}]: ${permission.error.message}`,
        });
      }

      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      });
      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }

      const rootKey = await db.query.keys.findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.forWorkspaceId, workspace.id), eq(table.id, input.rootKeyId)),
        with: {
          permissions: {
            with: {
              permission: true,
            },
          },
        },
      });
      if (!rootKey) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "root key not found",
        });
      }

      const { permissions, auditLogs } = await upsertPermissions(ctx, rootKey.workspaceId, [
        permission.data,
      ]);
      await db.transaction(async (tx) => {
        await tx
          .insert(schema.keysPermissions)
          .values({
            keyId: rootKey.id,
            permissionId: permissions[0].id,
            workspaceId: permissions[0].workspaceId,
          })
          .onDuplicateKeyUpdate({ set: { permissionId: permissions[0].id } });
        await insertAuditLogs(tx, auditLogs);
      });
    }),
  removePermissionFromRootKey: rateLimitedProcedure(ratelimit.update)
    .input(
      z.object({
        rootKeyId: z.string(),
        permissionName: z.string(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      });

      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }

      const key = await db.query.keys.findFirst({
        where: eq(schema.keys.forWorkspaceId, workspace.id) && eq(schema.keys.id, input.rootKeyId),
        with: {
          permissions: {
            with: {
              permission: true,
            },
          },
        },
      });
      if (!key) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `key ${input.rootKeyId} not found`,
        });
      }

      const permissionRelation = key.permissions.find(
        (kp) => kp.permission.name === input.permissionName,
      );
      if (!permissionRelation) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `key ${input.rootKeyId} did not have permission ${input.permissionName}`,
        });
      }

      await db.transaction(async (tx) => {
        await tx
          .delete(schema.keysPermissions)
          .where(
            and(
              eq(schema.keysPermissions.keyId, permissionRelation.keyId),
              eq(schema.keysPermissions.workspaceId, permissionRelation.workspaceId),
              eq(schema.keysPermissions.permissionId, permissionRelation.permissionId),
            ),
          );
        await insertAuditLogs(tx, {
          workspaceId: permissionRelation.workspaceId,
          actor: { type: "user", id: ctx.user!.id },
          event: "authorization.disconnect_permission_and_key",
          description: `Disconnected ${permissionRelation.keyId} from ${permissionRelation.permissionId}`,
          resources: [
            {
              type: "permission",
              id: permissionRelation.permissionId,
            },
            {
              type: "key",
              id: permissionRelation.keyId,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });
    }),
  connectPermissionToRole: rateLimitedProcedure(ratelimit.update)
    .input(
      z.object({
        roleId: z.string(),
        permissionId: z.string(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          roles: {
            where: (table, { eq }) => eq(table.id, input.roleId),
          },
          permissions: {
            where: (table, { eq }) => eq(table.id, input.permissionId),
          },
        },
      });
      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }
      const role = workspace.roles.at(0);
      if (!role) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "role not found",
        });
      }
      const permission = workspace.permissions.at(0);
      if (!permission) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "permission not found",
        });
      }

      const tuple = {
        workspaceId: workspace.id,
        permissionId: permission.id,
        roleId: role.id,
      };
      await db.transaction(async (tx) => {
        await tx
          .insert(schema.rolesPermissions)
          .values({ ...tuple, createdAt: new Date() })
          .onDuplicateKeyUpdate({
            set: { ...tuple, updatedAt: new Date() },
          });
        await insertAuditLogs(tx, {
          workspaceId: tuple.workspaceId,
          actor: { type: "user", id: ctx.user!.id },
          event: "authorization.connect_role_and_permission",
          description: `Connected ${tuple.roleId} to ${tuple.permissionId}`,
          resources: [
            {
              type: "permission",
              id: tuple.permissionId,
            },
            {
              type: "role",
              id: tuple.roleId,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });
    }),
  disconnectPermissionToRole: rateLimitedProcedure(ratelimit.update)
    .input(
      z.object({
        roleId: z.string(),
        permissionId: z.string(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      });
      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }
      await db.transaction(async (tx) => {
        await tx
          .delete(schema.rolesPermissions)
          .where(
            and(
              eq(schema.rolesPermissions.workspaceId, workspace.id),
              eq(schema.rolesPermissions.roleId, input.roleId),
              eq(schema.rolesPermissions.permissionId, input.permissionId),
            ),
          );
        await insertAuditLogs(tx, {
          workspaceId: workspace.id,
          actor: { type: "user", id: ctx.user!.id },
          event: "authorization.disconnect_role_and_permissions",
          description: `Disconnected ${input.roleId} from ${input.permissionId}`,
          resources: [
            {
              type: "permission",
              id: input.permissionId,
            },
            {
              type: "role",
              id: input.roleId,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });
    }),
  connectRoleToKey: rateLimitedProcedure(ratelimit.update)
    .input(
      z.object({
        roleId: z.string(),
        keyId: z.string(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          roles: {
            where: (table, { eq }) => eq(table.id, input.roleId),
          },
          keys: {
            where: (table, { eq }) => eq(table.id, input.keyId),
          },
        },
      });
      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }
      const role = workspace.roles.at(0);
      if (!role) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "role not found",
        });
      }
      const key = workspace.keys.at(0);
      if (!key) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "key not found",
        });
      }

      const tuple = {
        workspaceId: workspace.id,
        keyId: key.id,
        roleId: role.id,
      };
      await db.transaction(async (tx) => {
        await tx
          .insert(schema.keysRoles)
          .values({ ...tuple, createdAt: new Date() })
          .onDuplicateKeyUpdate({
            set: { ...tuple, updatedAt: new Date() },
          });
        await insertAuditLogs(tx, {
          workspaceId: tuple.workspaceId,
          actor: { type: "user", id: ctx.user!.id },
          event: "authorization.connect_role_and_key",
          description: `Connected ${tuple.roleId} with ${tuple.keyId}`,
          resources: [
            {
              type: "key",
              id: tuple.keyId,
            },
            {
              type: "role",
              id: tuple.roleId,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });
    }),
  disconnectRoleFromKey: rateLimitedProcedure(ratelimit.update)
    .input(
      z.object({
        roleId: z.string(),
        keyId: z.string(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      });
      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }
      await db
        .delete(schema.keysRoles)
        .where(
          and(
            eq(schema.keysRoles.workspaceId, workspace.id),
            eq(schema.keysRoles.roleId, input.roleId),
            eq(schema.keysRoles.keyId, input.keyId),
          ),
        );
    }),
  createRole: rateLimitedProcedure(ratelimit.create)
    .input(
      z.object({
        name: nameSchema,
        description: z.string().optional(),
        permissionIds: z.array(z.string()).optional(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      });

      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }
      const roleId = newId("role");
      await db.transaction(async (tx) => {
        await tx.insert(schema.roles).values({
          id: roleId,
          name: input.name,
          description: input.description,
          workspaceId: workspace.id,
        });
        await insertAuditLogs(tx, {
          workspaceId: workspace.id,
          event: "role.create",
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          description: `Created ${roleId}`,
          resources: [
            {
              type: "role",
              id: roleId,
            },
          ],

          context: {
            userAgent: ctx.audit.userAgent,
            location: ctx.audit.location,
          },
        });

        if (input.permissionIds && input.permissionIds.length > 0) {
          await tx.insert(schema.rolesPermissions).values(
            input.permissionIds.map((permissionId) => ({
              permissionId,
              roleId: roleId,
              workspaceId: workspace.id,
            })),
          );
          await insertAuditLogs(
            tx,
            input.permissionIds.map((permissionId) => ({
              workspaceId: workspace.id,
              event: "authorization.connect_role_and_permission",
              actor: {
                type: "user",
                id: ctx.user.id,
              },
              description: `Connected ${roleId} and ${permissionId}`,
              resources: [
                { type: "role", id: roleId },
                {
                  type: "permission",
                  id: permissionId,
                },
              ],

              context: {
                userAgent: ctx.audit.userAgent,
                location: ctx.audit.location,
              },
            })),
          );
        }
      });
      return { roleId };
    }),
  updateRole: rateLimitedProcedure(ratelimit.update)
    .input(
      z.object({
        id: z.string(),
        name: nameSchema,
        description: z.string().nullable(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          roles: {
            where: (table, { eq }) => eq(table.id, input.id),
          },
        },
      });

      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }
      if (workspace.roles.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "role not found",
        });
      }
      await db.transaction(async (tx) => {
        await tx.update(schema.roles).set(input).where(eq(schema.roles.id, input.id));
        await insertAuditLogs(tx, {
          workspaceId: workspace.id,
          event: "role.update",
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          description: `Updated ${input.id}`,
          resources: [{ type: "role", id: input.id }],

          context: {
            userAgent: ctx.audit.userAgent,
            location: ctx.audit.location,
          },
        });
      });
    }),
  deleteRole: rateLimitedProcedure(ratelimit.delete)
    .input(
      z.object({
        roleId: z.string(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          roles: {
            where: (table, { eq }) => eq(table.id, input.roleId),
          },
        },
      });

      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }
      if (workspace.roles.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "role not found",
        });
      }
      await db.transaction(async (tx) => {
        await tx
          .delete(schema.roles)
          .where(
            and(eq(schema.roles.id, input.roleId), eq(schema.roles.workspaceId, workspace.id)),
          );
        await insertAuditLogs(tx, {
          workspaceId: workspace.id,
          event: "role.delete",
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          description: `Deleted ${input.roleId}`,
          resources: [{ type: "role", id: input.roleId }],

          context: {
            userAgent: ctx.audit.userAgent,
            location: ctx.audit.location,
          },
        });
      });
    }),
  createPermission: rateLimitedProcedure(ratelimit.create)
    .input(
      z.object({
        name: nameSchema,
        description: z.string().optional(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      });

      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }
      const permissionId = newId("permission");
      await db.transaction(async (tx) => {
        await tx.insert(schema.permissions).values({
          id: permissionId,
          name: input.name,
          description: input.description,
          workspaceId: workspace.id,
        });
        await insertAuditLogs(tx, {
          workspaceId: workspace.id,
          event: "permission.create",
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          description: `Created ${permissionId}`,
          resources: [
            {
              type: "permission",
              id: permissionId,
            },
          ],

          context: {
            userAgent: ctx.audit.userAgent,
            location: ctx.audit.location,
          },
        });
      });

      return { permissionId };
    }),
  updatePermission: rateLimitedProcedure(ratelimit.update)
    .input(
      z.object({
        id: z.string(),
        name: nameSchema,
        description: z.string().nullable(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          permissions: {
            where: (table, { eq }) => eq(table.id, input.id),
          },
        },
      });

      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }
      if (workspace.permissions.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "permission not found",
        });
      }
      await db.transaction(async (tx) => {
        await tx
          .update(schema.permissions)
          .set({
            name: input.name,
            description: input.description,
            updatedAt: new Date(),
          })
          .where(eq(schema.permissions.id, input.id));
        await insertAuditLogs(tx, {
          workspaceId: workspace.id,
          event: "permission.update",
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          description: `Updated ${input.id}`,
          resources: [
            {
              type: "permission",
              id: input.id,
            },
          ],

          context: {
            userAgent: ctx.audit.userAgent,
            location: ctx.audit.location,
          },
        });
      });
    }),
  deletePermission: rateLimitedProcedure(ratelimit.delete)
    .input(
      z.object({
        permissionId: z.string(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          permissions: {
            where: (table, { eq }) => eq(table.id, input.permissionId),
          },
        },
      });

      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "workspace not found",
        });
      }
      if (workspace.permissions.length === 0) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "permission not found",
        });
      }
      await db.transaction(async (tx) => {
        await tx
          .delete(schema.permissions)
          .where(
            and(
              eq(schema.permissions.id, input.permissionId),
              eq(schema.permissions.workspaceId, workspace.id),
            ),
          );
        await insertAuditLogs(tx, {
          workspaceId: workspace.id,
          event: "permission.delete",
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          description: `Deleted ${input.permissionId}`,
          resources: [
            {
              type: "permission",
              id: input.permissionId,
            },
          ],

          context: {
            userAgent: ctx.audit.userAgent,
            location: ctx.audit.location,
          },
        });
      });
    }),
});

export async function upsertPermissions(
  ctx: Context,
  workspaceId: string,
  names: string[],
): Promise<{
  permissions: Permission[];
  auditLogs: UnkeyAuditLog[];
}> {
  return await db.transaction(async (tx) => {
    const existingPermissions = await tx.query.permissions.findMany({
      where: (table, { inArray, and, eq }) =>
        and(eq(table.workspaceId, workspaceId), inArray(table.name, names)),
    });

    const newPermissions: Permission[] = [];
    const auditLogs: UnkeyAuditLog[] = [];

    const permissions = names.map((name) => {
      const existingPermission = existingPermissions.find((p) => p.name === name);

      if (existingPermission) {
        return existingPermission;
      }

      const permission = {
        id: newId("permission"),
        workspaceId,
        name,
        description: null,
        createdAt: new Date(),
        updatedAt: null,
      };

      newPermissions.push(permission);
      auditLogs.push({
        workspaceId,
        actor: { type: "user", id: ctx.user!.id },
        event: "permission.create",
        description: `Created ${permission.id}`,
        resources: [
          {
            type: "permission",
            id: permission.id,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });

      return permission;
    });

    if (newPermissions.length) {
      await tx.insert(schema.permissions).values(newPermissions);
    }

    return { permissions, auditLogs };
  });
}
