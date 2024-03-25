import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const connectPermissionToRole = t.procedure
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
  });
