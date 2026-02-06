import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

export const deleteRole = workspaceProcedure
  .input(
    z.object({
      roleId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    await db.transaction(async (tx) => {
      const role = await tx.query.roles.findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.workspaceId, ctx.workspace.id), eq(table.id, input.roleId)),
      });

      if (!role) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message:
            "We are unable to find the correct role. Please try again or contact support@unkey.com.",
        });
      }
      await tx
        .delete(schema.roles)
        .where(
          and(eq(schema.roles.id, input.roleId), eq(schema.roles.workspaceId, ctx.workspace.id)),
        )
        .catch((err) => {
          console.error("Failed to delete role:", err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to delete the role. Please try again or contact support@unkey.com",
          });
        });
      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "role.delete",
        description: `Deleted role ${input.roleId}`,
        resources: [
          {
            type: "role",
            id: input.roleId,
            name: role.name,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      }).catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to delete the role. Please try again or contact support@unkey.com.",
        });
      });
    });
  });
