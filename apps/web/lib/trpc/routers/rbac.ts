import { type Permission, and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { unkeyPermissionValidation } from "@unkey/rbac";
import { z } from "zod";
import { auth, t } from "../trpc";

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
  createRole: t.procedure
    .use(auth)
    .input(
      z.object({
        name: z.string(),
        key: z.string(),
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
      const rolePublicId = newId("role");
      await db.insert(schema.roles).values({
        publicId: rolePublicId,
        key: input.key,
        name: input.name,
        description: input.description,
        workspaceId: workspace.id,
      });
      const role = await db.query.roles.findFirst({
        columns: {
          id: true,
          publicId: true,
        },
        where: (table, { eq }) => eq(table.publicId, rolePublicId),
      });
      if (input.permissionIds && input.permissionIds.length > 0) {
        await db.insert(schema.rolesPermissions).values(
          input.permissionIds.map((permissionId) => ({
            permissionId,
            roleId: role!.id,
            workspaceId: workspace.id,
          })),
        );
      }
      return { roleId: role!.publicId };
    }),
  createPermission: t.procedure
    .use(auth)
    .input(
      z.object({
        name: z.string(),
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
      const permissionPublicId = newId("permission");
      await db.insert(schema.permissions).values({
        id: permissionPublicId,
        name: input.name,
        workspaceId: workspace.id,
      });

      return { permissionId: permissionPublicId };
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
    };

    await tx.insert(schema.permissions).values(permission);
    return permission;
  });
}
