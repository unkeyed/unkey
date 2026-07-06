import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { db, schema, sql } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { createVaultClient } from "@/lib/vault-client";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../../trpc";

const vault = createVaultClient(VaultService);

export const linkStripeConnectInputSchema = z.object({
  connectedAccountId: z
    .string()
    .regex(/^acct_[a-zA-Z0-9]+$/, "Must be a Stripe connected account id (acct_...)"),
});

/**
 * Links the workspace's Stripe connected account for end-user billing. The
 * account id is verified against Stripe first — retrieving an account with
 * the platform key succeeds only when it is connected to Unkey's platform —
 * so a member cannot bind an arbitrary merchant's account. The reference is
 * stored Vault-encrypted (keyring = workspace id, matching the period-close
 * decrypter), never plaintext.
 */
export const linkStripeConnect = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(linkStripeConnectInputSchema)
  .mutation(async ({ input, ctx }) => {
    const stripe = getStripeClient();
    try {
      await stripe.accounts.retrieve(input.connectedAccountId);
    } catch {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message:
          "This connected account could not be verified. Complete Stripe Connect onboarding for Unkey first.",
      });
    }

    const { encrypted, keyId } = await vault.encrypt({
      keyring: ctx.workspace.id,
      data: input.connectedAccountId,
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
          stripeConnectStatus: "linked",
          createdAt: Date.now(),
          updatedAt: null,
        })
        .onDuplicateKeyUpdate({
          set: {
            stripeConnectEncrypted: encrypted,
            stripeConnectEncryptionKeyId: keyId,
            stripeConnectStatus: "linked",
            updatedAt: sql`${Date.now()}`,
          },
        });

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "workspace.update",
        description: `Linked Stripe connected account ${input.connectedAccountId} for end-user billing`,
        resources: [
          {
            type: "workspace",
            id: ctx.workspace.id,
            name: ctx.workspace.name || "Unknown workspace",
            meta: { connectedAccountId: input.connectedAccountId },
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });

    return { connectedAccountId: input.connectedAccountId };
  });
