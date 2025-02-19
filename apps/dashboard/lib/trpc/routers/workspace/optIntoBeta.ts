import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const optWorkspaceIntoBeta = t.procedure
  .use(auth)
  .input(
    z.object({
      feature: z.enum(["rbac", "ratelimit", "identities", "logsPage"]),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    switch (input.feature) {
      case "rbac": {
        ctx.workspace.betaFeatures.rbac = true;
        break;
      }
      case "identities": {
        ctx.workspace.betaFeatures.identities = true;
        break;
      }
    }
    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.workspaces)
          .set({
            betaFeatures: ctx.workspace.betaFeatures,
          })
          .where(eq(schema.workspaces.id, ctx.workspace.id));
        await insertAuditLogs(tx, ctx.workspace.auditLogBucket.id, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.opt_in",
          description: `Opted ${ctx.workspace.id} into beta: ${input.feature}`,
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
      })
      .catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to update workspace, Please try again or contact support@unkey.dev.",
        });
      });
  });
