import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const disconnectPermissionFromRole = workspaceProcedure
  .input(
    z.object({
      roleId: z.string(),
      permissionId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    await db
      .transaction(async (tx) => {
        await tx
          .delete(schema.rolesPermissions)
          .where(
            and(
              eq(schema.rolesPermissions.workspaceId, ctx.workspace.id),
              eq(schema.rolesPermissions.roleId, input.roleId),
              eq(schema.rolesPermissions.permissionId, input.permissionId),
            ),
          );
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
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
      })
      .catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to disconnect the permission from the role. Please try again or contact support@unkey.com",
        });
      });
  });
