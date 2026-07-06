import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, schema } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { createVaultClient } from "@/lib/vault-client";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../../trpc";

const vault = createVaultClient(VaultService);

export const createCustomerForIdentityInputSchema = z.object({
  identityId: z.string(),
  // Customer details are forwarded to Stripe and NOT persisted by Unkey — only
  // the resulting customer id is stored on the identity.
  email: z.string().trim().email(),
  name: z.string().trim().max(256).optional(),
  phone: z.string().trim().max(64).optional(),
  description: z.string().trim().max(512).optional(),
  // Optional rate card to assign at the same time; null = workspace default.
  rateCardId: z.string().nullable().default(null),
});

/**
 * Creates a Stripe customer on the workspace's connected account for an
 * end-user identity, then binds the identity to it (billing_provider =
 * stripe_connect, billing_external_customer_id = the new customer id). The
 * customer details (email/name/phone) are sent to Stripe and deliberately NOT
 * stored by Unkey; only the returned customer id is persisted. The Unkey
 * identity id is written to the Stripe customer metadata for traceability.
 */
export const createCustomerForIdentity = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(createCustomerForIdentityInputSchema)
  .mutation(async ({ input, ctx }) => {
    const identity = await db.query.identities.findFirst({
      where: (table, { and: andW, eq: eqW }) =>
        andW(eqW(table.id, input.identityId), eqW(table.workspaceId, ctx.workspace.id)),
    });
    if (!identity) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Identity not found." });
    }

    if (input.rateCardId !== null) {
      const rateCardId = input.rateCardId;
      const card = await db.query.rateCards.findFirst({
        where: (table, { and: andW, eq: eqW }) =>
          andW(
            eqW(table.id, rateCardId),
            eqW(table.workspaceId, ctx.workspace.id),
            eqW(table.archived, false),
          ),
      });
      if (!card) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Rate card not found in this workspace.",
        });
      }
    }

    const settings = await db.query.workspaceBillingSettings.findFirst({
      where: (table, { eq: eqW }) => eqW(table.workspaceId, ctx.workspace.id),
      columns: { stripeConnectEncrypted: true, stripeConnectStatus: true },
    });
    if (!settings?.stripeConnectEncrypted || settings.stripeConnectStatus !== "linked") {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Connect a Stripe account on the Monetization page before creating customers.",
      });
    }

    const { plaintext: connectedAccountId } = await vault.decrypt({
      keyring: ctx.workspace.id,
      encrypted: settings.stripeConnectEncrypted,
    });

    const stripe = getStripeClient();
    let customerId: string;
    try {
      // Created ON the connected account (direct-charge model); the customer
      // and its details live on the customer's Stripe account, not ours.
      const customer = await stripe.customers.create(
        {
          email: input.email,
          name: input.name || undefined,
          phone: input.phone || undefined,
          description: input.description || undefined,
          metadata: {
            unkey_identity_id: identity.id,
            unkey_external_id: identity.externalId,
          },
        },
        { stripeAccount: connectedAccountId },
      );
      customerId = customer.id;
    } catch (err) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `Stripe rejected the customer: ${err instanceof Error ? err.message : String(err)}`,
      });
    }

    await db.transaction(async (tx) => {
      await tx
        .update(schema.identities)
        .set({
          billingProvider: "stripe_connect",
          billingExternalCustomerId: customerId,
          ...(input.rateCardId !== null ? { rateCardId: input.rateCardId } : {}),
        })
        .where(
          and(
            eq(schema.identities.id, input.identityId),
            eq(schema.identities.workspaceId, ctx.workspace.id),
          ),
        );

      // Audit deliberately records only the resulting customer id + identity,
      // never the forwarded PII (email/name/phone), matching "we don't save the
      // customer details".
      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "identity.update",
        description: `Created Stripe customer ${customerId} for identity ${input.identityId} and bound it for billing`,
        resources: [
          {
            type: "identity",
            id: input.identityId,
            name: identity.externalId,
            meta: {
              billingProvider: "stripe_connect",
              customerId,
              rateCardId: input.rateCardId,
            },
          },
        ],
        context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
      });
    });

    return { customerId, externalId: identity.externalId };
  });
