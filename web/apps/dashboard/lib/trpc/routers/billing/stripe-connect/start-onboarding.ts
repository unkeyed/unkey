import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { db, schema, sql } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { getBaseUrl } from "@/lib/utils";
import { createVaultClient } from "@/lib/vault-client";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../../trpc";

const vault = createVaultClient(VaultService);

/**
 * Starts Stripe-hosted Connect onboarding: creates (or reuses) a Standard
 * connected account for the workspace and returns a single-use Stripe
 * account-link URL to redirect the customer to. The account is stored
 * Vault-encrypted with status "pending" — it is never billed until
 * finishOnboarding observes details_submitted and flips it to "linked".
 *
 * Standard accounts keep the customer as merchant-of-record: they complete
 * Stripe's own KYC and own the dashboard, disputes, and payouts.
 */
export const startStripeConnectOnboarding = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .mutation(async ({ ctx }) => {
    const stripe = getStripeClient();

    const settings = await db.query.workspaceBillingSettings.findFirst({
      where: (table, { eq }) => eq(table.workspaceId, ctx.workspace.id),
    });

    if (settings?.stripeConnectStatus === "linked") {
      throw new TRPCError({
        code: "CONFLICT",
        message: "A Stripe account is already linked. Unlink it first to connect a different one.",
      });
    }

    // Resume a half-finished onboarding with the same account rather than
    // minting a fresh one per click.
    let accountId: string | null = null;
    if (settings?.stripeConnectStatus === "pending" && settings.stripeConnectEncrypted) {
      try {
        const { plaintext } = await vault.decrypt({
          keyring: ctx.workspace.id,
          encrypted: settings.stripeConnectEncrypted,
        });
        accountId = plaintext;
      } catch (err) {
        // Decrypting a stale/rotated pending reference can fail; log it and
        // fall through to minting a fresh connected account rather than
        // silently swallowing the error.
        console.warn("stripe connect: failed to decrypt pending account, creating fresh", {
          workspaceId: ctx.workspace.id,
          error: err instanceof Error ? err.message : String(err),
        });
        accountId = null;
      }
    }

    if (!accountId) {
      const account = await stripe.accounts.create({ type: "standard" });
      accountId = account.id;

      const { encrypted, keyId } = await vault.encrypt({
        keyring: ctx.workspace.id,
        data: accountId,
      });

      await db.transaction(async (tx) => {
        await tx
          .insert(schema.workspaceBillingSettings)
          .values({
            id: newId("rateCard"),
            workspaceId: ctx.workspace.id,
            defaultRateCardId: null,
            stripeConnectEncrypted: encrypted,
            stripeConnectEncryptionKeyId: keyId,
            stripeConnectStatus: "pending",
            createdAt: Date.now(),
            updatedAt: null,
          })
          .onDuplicateKeyUpdate({
            set: {
              stripeConnectEncrypted: encrypted,
              stripeConnectEncryptionKeyId: keyId,
              stripeConnectStatus: "pending",
              updatedAt: sql`${Date.now()}`,
            },
          });

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "workspace.update",
          description: "Started Stripe Connect onboarding for end-user billing",
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
    }

    // Return to the Monetization page (where the Stripe Connect card now lives,
    // moved out of Settings > Billing). The ?connect=return handler that
    // finishes onboarding (pending -> linked) is rendered by that card, so the
    // redirect must land on the page that hosts it.
    const monetizationUrl = `${getBaseUrl()}/${ctx.workspace.slug}/monetization`;
    const link = await stripe.accountLinks.create({
      account: accountId,
      refresh_url: `${monetizationUrl}?connect=refresh`,
      return_url: `${monetizationUrl}?connect=return`,
      type: "account_onboarding",
    });

    return { url: link.url };
  });
