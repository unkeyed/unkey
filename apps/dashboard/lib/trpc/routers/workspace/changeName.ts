import { insertAuditLogs } from "@/lib/audit";
import { auth as authClient } from "@/lib/auth/server";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";
export const changeWorkspaceName = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      name: z
        .string()
        .min(3, "Workspace names must contain at least 3 characters")
        .max(50, "Workspace names must contain less than 50 characters"),
    })
  )
  .mutation(async ({ ctx, input }) => {
    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.workspaces)
          .set({
            name: input.name,
          })
          .where(eq(schema.workspaces.id, ctx.workspace.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to update the workspace name. Please try again or contact support@unkey.dev",
            });
          });
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.update",
          description: `Changed name from ${ctx.workspace.name} to ${input.name}`,
          resources: [
            {
              type: "workspace",
              id: ctx.workspace.id,
              name: input.name,
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
