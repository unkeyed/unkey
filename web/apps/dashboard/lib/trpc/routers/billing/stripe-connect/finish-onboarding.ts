import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { createVaultClient } from "@/lib/vault-client";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../../trpc";

const vault = createVaultClient(VaultService);

/**
 * Completes onboarding after the customer returns from Stripe: checks the
 * pending account and flips it to "linked" once Stripe reports
 * details_submitted. Safe to call repeatedly — the settings page invokes it
 * on every ?connect=return visit, and Stripe-side state is the authority.
 */
export const finishStripeConnectOnboarding = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .mutation(async ({ ctx }) => {
    const settings = await db.query.workspaceBillingSettings.findFirst({
      where: (table, { eq: eqWhere }) => eqWhere(table.workspaceId, ctx.workspace.id),
    });

    if (!settings?.stripeConnectEncrypted || !settings.stripeConnectStatus) {
      return { status: "none" as const, detailsSubmitted: false, chargesEnabled: false };
    }
    if (settings.stripeConnectStatus === "linked") {
      return { status: "linked" as const, detailsSubmitted: true, chargesEnabled: true };
    }

    const { plaintext: accountId } = await vault.decrypt({
      keyring: ctx.workspace.id,
      encrypted: settings.stripeConnectEncrypted,
    });

    const stripe = getStripeClient();
    const account = await stripe.accounts.retrieve(accountId);

    if (!account.details_submitted) {
      return {
        status: "pending" as const,
        detailsSubmitted: false,
        chargesEnabled: Boolean(account.charges_enabled),
      };
    }

    await db.transaction(async (tx) => {
      await tx
        .update(schema.workspaceBillingSettings)
        .set({ stripeConnectStatus: "linked" })
        .where(eq(schema.workspaceBillingSettings.workspaceId, ctx.workspace.id));

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: `Completed Stripe Connect onboarding (${accountId}) for end-user billing`,
        resources: [
          {
            type: "workspace",
            id: ctx.workspace.id,
            name: ctx.workspace.name || "Unknown workspace",
            meta: { connectedAccountId: accountId },
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });

    return {
      status: "linked" as const,
      detailsSubmitted: true,
      chargesEnabled: Boolean(account.charges_enabled),
    };
  });
