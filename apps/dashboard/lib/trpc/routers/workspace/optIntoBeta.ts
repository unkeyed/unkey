import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const optWorkspaceIntoBeta = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      feature: z.enum(["rbac", "ratelimit", "identities", "logsPage", "deployments"]),
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
      case "deployments": {
        ctx.workspace.betaFeatures.deployments = true;
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
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.opt_in",
          description: `Opted ${ctx.workspace.id} into beta: ${input.feature}`,
          resources: [
            {
              type: "workspace",
              id: ctx.workspace.id,
              name: ctx.workspace.name,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });

        // Invalidate workspace cache after successful update
        await invalidateWorkspaceCache(ctx.tenant.id);
      })
      .catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to update workspace, Please try again or contact support@unkey.dev.",
        });
      });
  });
