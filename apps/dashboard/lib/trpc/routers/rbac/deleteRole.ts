import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const deleteRole = rateLimitedProcedure(ratelimit.delete)
  .input(
    z.object({
      roleId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          roles: {
            where: (table, { eq }) => eq(table.id, input.roleId),
          },
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "We are unable to delete role. Please contact support using support@unkey.dev",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please contact support using support@unkey.dev.",
      });
    }
    if (workspace.roles.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct role. Please contact support using support@unkey.dev.",
      });
    }
    await db.transaction(async (tx) => {
      await tx
        .delete(schema.roles)
        .where(and(eq(schema.roles.id, input.roleId), eq(schema.roles.workspaceId, workspace.id)))
        .catch((_err) => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We are unable to delete the role. Please contact support using support@unkey.dev",
          });
        });
      await insertAuditLogs(tx, {
        workspaceId: workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "role.delete",
        description: `Deleted role ${input.roleId}`,
        resources: [
          {
            type: "role",
            id: input.roleId,
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
            "We are unable to delete the role. Please contact support using support@unkey.dev.",
        });
      });
    });
  });
