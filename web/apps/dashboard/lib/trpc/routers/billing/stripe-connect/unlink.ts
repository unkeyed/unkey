import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../../trpc";

/**
 * Unlinks the workspace's Stripe connected account: the period-close push
 * skips the workspace afterwards. Recorded billing periods are unaffected.
 */
export const unlinkStripeConnect = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .mutation(async ({ ctx }) => {
    await db.transaction(async (tx) => {
      await tx
        .update(schema.workspaceBillingSettings)
        .set({
          stripeConnectEncrypted: null,
          stripeConnectEncryptionKeyId: null,
          stripeConnectStatus: null,
        })
        .where(eq(schema.workspaceBillingSettings.workspaceId, ctx.workspace.id));

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: "Unlinked the Stripe connected account for end-user billing",
        resources: [
          {
            type: "workspace",
            id: ctx.workspace.id,
            name: ctx.workspace.name || "Unknown workspace",
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });

    return { unlinked: true };
  });
