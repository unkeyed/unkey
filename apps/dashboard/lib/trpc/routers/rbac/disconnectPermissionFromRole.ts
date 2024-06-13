import { and, db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
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

    await ingestAuditLogs({
      workspaceId: workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "authorization.disconnect_role_and_permissions",
      description: `Disconnect role ${input.roleId} from permission ${input.permissionId}`,
      resources: [
        {
          type: "role",
          id: input.roleId,
        },
        {
          type: "permission",
          id: input.permissionId,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
  });
