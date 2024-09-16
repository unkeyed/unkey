import { and, db, eq, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "../../ratelimitProcedure";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";


export const disconnectPermissionFromRole = rateLimitedProcedure(ratelimit.update)
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
        message:
          "We are unable to find the correct workspace. Please contact support using support@unkey.dev.",
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
      )
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to disconnect the permission from the role. Please contact support using support@unkey.dev",
        });
      });

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
