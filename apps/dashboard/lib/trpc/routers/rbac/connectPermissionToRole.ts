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
        message:
          "Sorry, we are unable to find the correct workspace. Please contact support using support@unkey.dev.",
      });
    }
    const role = workspace.roles.at(0);
    if (!role) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "Sorry, we are unable to find the correct role. Please contact support using support@unkey.dev.",
      });
    }
    const permission = workspace.permissions.at(0);
    if (!permission) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "Sorry, we are unable to find the correct permission. Please contact support using support@unkey.dev.",
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
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Sorry, we are unable to connect the permission to the role. Please contact support using support@unkey.dev.",
        });
      });
  });
