import { type Permission, and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";
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

      const p = await upsertPermission(rootKey.workspaceId, permission.data);

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

      await db.insert(schema.rolesPermissions).values({
        workspaceId: workspace.id,
        permissionId: permission.id,
        roleId: role.id,
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

      if (input.permissionIds && input.permissionIds.length > 0) {
        await db.insert(schema.rolesPermissions).values(
          input.permissionIds.map((permissionId) => ({
            permissionId,
            roleId: roleId,
            workspaceId: workspace.id,
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

export async function upsertPermission(workspaceId: string, name: string): Promise<Permission> {
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
    return permission;
  });
}
