import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const disconnectRoleFromKey = rateLimitedProcedure(ratelimit.update)
  .input(
    z.object({
      roleId: z.string(),
      keyId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to disconnect the role from the key. Please try again or contact support@unkey.dev",
        });
      });
    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please try again or contact support@unkey.dev.",
      });
    }
    await db
      .transaction(async (tx) => {
        await tx
          .delete(schema.keysRoles)
          .where(
            and(
              eq(schema.keysRoles.workspaceId, workspace.id),
              eq(schema.keysRoles.roleId, input.roleId),
              eq(schema.keysRoles.keyId, input.keyId),
            ),
          );
        await insertAuditLogs(tx, {
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
      })
      .catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to disconnect the role from the key. Please try again or contact support@unkey.dev",
        });
      });
  });
