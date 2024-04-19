import { and, db, eq, schema } from "@/lib/db";
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
        message: "workspace not found",
      });
    }
    if (workspace.permissions.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "permission not found",
      });
    }
    await db
      .delete(schema.permissions)
      .where(
        and(
          eq(schema.permissions.id, input.permissionId),
          eq(schema.permissions.workspaceId, workspace.id),
        ),
      );
  });
