import { and, db, eq, schema } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const disconnectRoleFromKey = t.procedure
  .use(auth)
  .input(
    z.object({
      roleId: z.string(),
      keyId: z.string(),
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
      .delete(schema.keysRoles)
      .where(
        and(
          eq(schema.keysRoles.workspaceId, workspace.id),
          eq(schema.keysRoles.roleId, input.roleId),
          eq(schema.keysRoles.keyId, input.keyId),
        ),
      );

    await ingestAuditLogs({
      workspaceId: workspace.id,
      actor: { type: "user", id: ctx.user.id },
      event: "authorization.disconnect_role_and_key",
      description: `Disconnect role ${input.roleId} from ${input.keyId}`,
      resources: [
        {
          type: "role",
          id: input.roleId,
        },
        {
          type: "key",
          id: input.keyId,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });
  });
