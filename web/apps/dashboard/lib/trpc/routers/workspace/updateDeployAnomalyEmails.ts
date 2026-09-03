import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema, transactionWithRetry } from "@/lib/db";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

export const updateDeployAnomalyEmailsInput = z.object({
  muted: z.boolean(),
});

export const updateDeployAnomalyEmails = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(updateDeployAnomalyEmailsInput)
  .mutation(async ({ ctx, input }) => {
    await transactionWithRetry(db, async (tx) => {
      await tx
        .update(schema.workspaces)
        .set({
          betaFeatures: {
            ...ctx.workspace.betaFeatures,
            deploy_anomaly_alerts_muted: input.muted,
          },
        })
        .where(eq(schema.workspaces.id, ctx.workspace.id));

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: input.muted
          ? "Muted Deploy anomaly emails."
          : "Enabled Deploy anomaly emails.",
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
    });

    return { muted: input.muted };
  });
