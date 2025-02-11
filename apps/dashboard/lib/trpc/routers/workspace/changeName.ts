import { insertAuditLogs } from "@/lib/audit";
import { auth as authClient } from "@/lib/auth/server";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";
export const changeWorkspaceName = t.procedure
  .use(auth)
  .input(
    z.object({
      name: z.string().min(3, "workspace names must contain at least 3 characters"),
      workspaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.workspaces)
          .set({
            name: input.name,
          })
          .where(eq(schema.workspaces.id, input.workspaceId))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to update the workspace name. Please try again or contact support@unkey.dev",
            });
          });
        await insertAuditLogs(tx, ctx.workspace.auditLogBucket.id, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.update",
          description: `Changed name from ${ctx.workspace.name} to ${input.name}`,
          resources: [
            {
              type: "workspace",
              id: ctx.workspace.id,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });

        await authClient.updateOrg({
          id: ctx.tenant.id,
          name: input.name,
        });
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the workspace name. Please try again or contact support@unkey.dev",
        });
      });
  });
