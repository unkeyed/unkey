import { type Permission, and, db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";
import type { Context } from "../context";
import { auth, t } from "../trpc";

const nameSchema = z
  .string()
  .min(3)
  .regex(/^[a-zA-Z0-9_\-\.\*]+$/, {
    message:
      "Must be at least 3 characters long and only contain alphanumeric, periods, dashes and underscores",
  });

export const rbacRouter = t.router({
  addPermissionToRootKey: t.procedure
    .use(auth)
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
        throw new TRPCError({ code: "NOT_FOUND", message: "root key not found" });
      }

      const p = await upsertPermission(ctx, rootKey.workspaceId, permission.data);

      await db
        .insert(schema.keysPermissions)
        .values({
          keyId: rootKey.id,
          permissionId: p.id,
          workspaceId: p.workspaceId,
        })
        .onDuplicateKeyUpdate({ set: { permissionId: p.id } });
    }),
  removePermissionFromRootKey: t.procedure
    .use(auth)
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

      await db
        .delete(schema.keysPermissions)
        .where(
          and(
            eq(schema.keysPermissions.keyId, permissionRelation.keyId),
            eq(schema.keysPermissions.workspaceId, permissionRelation.workspaceId),
            eq(schema.keysPermissions.permissionId, permissionRelation.permissionId),
          ),
        );
    }),
  connectPermissionToRole: t.procedure
    .use(auth)
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
      await db
        .insert(schema.rolesPermissions)
        .values({ ...tuple, createdAt: new Date() })
        .onDuplicateKeyUpdate({
          set: { ...tuple, updatedAt: new Date() },
        });
    }),
  disconnectPermissionToRole: t.procedure
    .use(auth)
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
      await db
        .delete(schema.rolesPermissions)
        .where(
          and(
            eq(schema.rolesPermissions.workspaceId, workspace.id),
            eq(schema.rolesPermissions.roleId, input.roleId),
            eq(schema.rolesPermissions.permissionId, input.permissionId),
          ),
        );
    }),
  connectRoleToKey: t.procedure
    .use(auth)
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
      await db
        .insert(schema.keysRoles)
        .values({ ...tuple, createdAt: new Date() })
        .onDuplicateKeyUpdate({
          set: { ...tuple, updatedAt: new Date() },
        });
    }),
  disconnectRoleFromKey: t.procedure
    .use(auth)
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
  createRole: t.procedure
    .use(auth)
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
      await db.insert(schema.roles).values({
        id: roleId,
        name: input.name,
        description: input.description,
        workspaceId: workspace.id,
      });
      await ingestAuditLogs({
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
        await db.insert(schema.rolesPermissions).values(
          input.permissionIds.map((permissionId) => ({
            permissionId,
            roleId: roleId,
            workspaceId: workspace.id,
          })),
        );
        await ingestAuditLogs(
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
      return { roleId };
    }),
  updateRole: t.procedure
    .use(auth)
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
      await db.update(schema.roles).set(input).where(eq(schema.roles.id, input.id));
    }),
  deleteRole: t.procedure
    .use(auth)
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
      await db
        .delete(schema.roles)
        .where(and(eq(schema.roles.id, input.roleId), eq(schema.roles.workspaceId, workspace.id)));
    }),
  createPermission: t.procedure
    .use(auth)
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
      await db.insert(schema.permissions).values({
        id: permissionId,
        name: input.name,
        description: input.description,
        workspaceId: workspace.id,
      });
      await ingestAuditLogs({
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

      return { permissionId };
    }),
  updatePermission: t.procedure
    .use(auth)
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
      await db
        .update(schema.permissions)
        .set({
          name: input.name,
          description: input.description,
          updatedAt: new Date(),
        })
        .where(eq(schema.permissions.id, input.id));
    }),
  deletePermission: t.procedure
    .use(auth)
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
      await db
        .delete(schema.permissions)
        .where(
          and(
            eq(schema.permissions.id, input.permissionId),
            eq(schema.permissions.workspaceId, workspace.id),
          ),
        );
    }),
});

export async function upsertPermission(
  ctx: Context,
  workspaceId: string,
  name: string,
): Promise<Permission> {
  return await db.transaction(async (tx) => {
    const existingPermission = await tx.query.permissions.findFirst({
      where: (table, { and, eq }) => and(eq(table.workspaceId, workspaceId), eq(table.name, name)),
    });
    if (existingPermission) {
      return existingPermission;
    }

    const permission: Permission = {
      id: newId("permission"),
      workspaceId,
      name,
      description: null,
      createdAt: new Date(),
      updatedAt: null,
    };

    await tx.insert(schema.permissions).values(permission);
    await ingestAuditLogs({
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
}
