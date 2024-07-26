import { and, db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const deletePermission = t.procedure
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
        message:
          "Sorry, we are unable to find the correct workspace. Please contact support using support@unkey.dev.",
      });
    }
    if (workspace.permissions.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "Sorry, we are unable to find the correct permission. Please contact support using support@unkey.dev.",
      });
    }
    await db
      .delete(schema.permissions)
      .where(
        and(
          eq(schema.permissions.id, input.permissionId),
          eq(schema.permissions.workspaceId, workspace.id),
        ),
      )
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Sorry, we are unable to delete the permission. Please contact support using support@unkey.dev",
        });
      });

    await ingestAuditLogs({
      workspaceId: workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "permission.delete",
      description: `Deleted permission ${input.permissionId}`,
      resources: [
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
