import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const disconnectPermissionFromRole = t.procedure
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
  });
