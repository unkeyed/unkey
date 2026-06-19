import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import {
  deployBillingConfig,
  deployBillingConfigured,
  findApiItem,
} from "@/lib/stripe/deployBilling";
import { validateAndParseQuotas } from "@/lib/stripe/productUtils";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";
import { assertSubscriptionAttachable } from "./subscriptionGuards";

export const createSubscription = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(
    z.object({
      productId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const stripe = getStripeClient();
    const e = stripeEnv();
    if (!e) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Stripe is not set up",
      });
    }

    // Reject any product that is not on the configured allow-list, so a
    // workspace admin can only subscribe to plans the operator has explicitly
    // exposed. Without this, a known/internal Stripe product id with $0
    // default price (or permissive quota metadata) lets an admin self-grant
    // a higher tier.
    const allowedProductIds = new Set<string>([
      ...e.STRIPE_PRODUCT_IDS_PRO,
      ...e.STRIPE_PRODUCT_IDS_ENTERPRISE,
    ]);
    if (!allowedProductIds.has(input.productId)) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find product ${input.productId}.`,
      });
    }

    const product = await stripe.products.retrieve(input.productId);

    if (!product) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find product ${input.productId}.`,
      });
    }

    if (!product.default_price) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find product default price ${input.productId}.`,
      });
    }

    const quotas = validateAndParseQuotas(product);
    if (
      !quotas.valid ||
      quotas.requestsPerMonth === undefined ||
      quotas.logsRetentionDays === undefined ||
      quotas.auditLogsRetentionDays === undefined
    ) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: `Product ${input.productId} is missing required quota metadata.`,
      });
    }

    if (!ctx.workspace.stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspaces does not have a stripe account.",
      });
    }

    // A Compute-first workspace already has a subscription, carrying only
    // Deploy items. Both products live on one subscription (one invoice), so
    // the API item is appended to it — mirroring how subscribeDeploy appends
    // Deploy items to an API-first subscription. Only a subscription that
    // already carries an API plan item is refused.
    let existingSub: Stripe.Subscription | undefined;
    if (ctx.workspace.stripeSubscriptionId) {
      existingSub = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);
      // Fail closed when Deploy is configured but unresolved: findApiItem(null)
      // falls back to items[0], which on a Compute-first subscription is a
      // Deploy item, so the check below would wrongly report "already has an API
      // plan". Unconfigured (no Deploy) still uses the items[0] fallback safely.
      const deployConfig = await deployBillingConfig();
      if (!deployConfig && deployBillingConfigured()) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Billing is temporarily unavailable. Please try again in a moment.",
        });
      }
      if (findApiItem(deployConfig, existingSub.items.data)) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: `Customer ${ctx.workspace.stripeCustomerId} already has an API plan.`,
        });
      }
      // Same guards subscribeDeploy applies to its append path: don't attach
      // the API item to a subscription that is delinquent, scheduled to cancel,
      // or non-USD. cancel_at_period_end especially is accepted silently by
      // Stripe, so the item would never bill next cycle.
      assertSubscriptionAttachable(existingSub);
    }

    const customer = await stripe.customers.retrieve(ctx.workspace.stripeCustomerId);
    if (!customer) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Customer ${ctx.workspace.stripeCustomerId} could not be found.`,
      });
    }

    /**
     * `error_if_incomplete` makes Stripe reject the call with a 402 if the first
     * invoice cannot be paid (e.g. card declined). We rely on this to keep the workspace
     * on the Free tier when a first-time signup's payment fails — no DB writes happen
     * unless the subscription is fully active.
     */
    let sub: Stripe.Subscription;
    try {
      sub = existingSub
        ? await stripe.subscriptions.update(existingSub.id, {
            items: [
              {
                price: product.default_price.toString(),
              },
            ],
            proration_behavior: "always_invoice",
            payment_behavior: "error_if_incomplete",
          })
        : await stripe.subscriptions.create({
            customer: customer.id,
            items: [
              {
                price: product.default_price.toString(),
              },
            ],
            billing_cycle_anchor_config: {
              day_of_month: 1,
            },
            // Stripe API 2025-09-30 (clover) and later default new
            // subscriptions to the "flexible" billing mode, which itemizes
            // prorations differently and would change the Deploy
            // credit-grant net-fee math. Stay on classic.
            billing_mode: { type: "classic" },
            proration_behavior: "always_invoice",
            payment_behavior: "error_if_incomplete",
          });
    } catch (err) {
      if (err instanceof Stripe.errors.StripeCardError) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message:
            err.message ||
            "Your card was declined. Please update your payment method and try again.",
        });
      }
      if (err instanceof Stripe.errors.StripeError) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message:
            err.message ||
            "Payment could not be completed. Please update your payment method and try again.",
        });
      }
      throw err;
    }

    if (!existingSub && sub.status !== "active" && sub.status !== "trialing") {
      // Defensive guard: error_if_incomplete should make this unreachable, but never
      // grant tier access to a subscription that isn't actually paid. Only the
      // freshly created subscription is cancelled here — on the append path the
      // subscription pre-exists and carries the Compute plan.
      try {
        await stripe.subscriptions.cancel(sub.id);
      } catch (cancelErr) {
        console.error(
          `Failed to cancel non-active subscription ${sub.id} after creation:`,
          cancelErr,
        );
      }
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `Subscription was created but is not active (ID: ${sub.id}). Please contact support.`,
      });
    }

    await db.transaction(async (tx) => {
      await tx
        .update(schema.workspaces)
        .set({
          stripeSubscriptionId: sub.id,
          tier: product.name,
        })
        .where(eq(schema.workspaces.id, ctx.workspace.id));

      await tx
        .insert(schema.quotas)
        .values({
          workspaceId: ctx.workspace.id,
          requestsPerMonth: quotas.requestsPerMonth,
          logsRetentionDays: quotas.logsRetentionDays,
          auditLogsRetentionDays: quotas.auditLogsRetentionDays,
          team: true,
        })
        .onDuplicateKeyUpdate({
          set: {
            requestsPerMonth: quotas.requestsPerMonth,
            logsRetentionDays: quotas.logsRetentionDays,
            auditLogsRetentionDays: quotas.auditLogsRetentionDays,
            team: true,
          },
        });

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: {
          type: "user",
          id: ctx.user.id,
        },
        event: "workspace.update",
        description: `Subscribed to ${product.name} plan`,
        resources: [],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });

    // Invalidate workspace cache after subscription creation
    await invalidateWorkspaceCache(ctx.tenant.id);
  });
