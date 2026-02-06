import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const deletePermission = workspaceProcedure
  .input(
    z.object({
      permissionId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    await db
      .transaction(async (tx) => {
        const permission = await tx.query.permissions.findFirst({
          where: (table, { and, eq }) =>
            and(eq(table.workspaceId, ctx.workspace.id), eq(table.id, input.permissionId)),
        });

        if (!permission) {
          throw new TRPCError({
            code: "NOT_FOUND",
            message:
              "We are unable to find the correct permission. Please try again or contact support@unkey.com.",
          });
        }
        await tx
          .delete(schema.permissions)
          .where(
            and(
              eq(schema.permissions.id, permission.id),
              eq(schema.permissions.workspaceId, ctx.workspace.id),
            ),
          );
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "permission.delete",
          description: `Deleted permission ${input.permissionId}`,
          resources: [
            {
              type: "permission",
              id: input.permissionId,
              name: permission.name,
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
            "We are unable to delete the permission. Please try again or contact support@unkey.com",
        });
      });
  });
