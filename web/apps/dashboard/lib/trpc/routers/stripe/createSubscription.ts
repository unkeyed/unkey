import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import { validateAndParseQuotas } from "@/lib/stripe/productUtils";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";
import { clearWorkspaceCache } from "../workspace/getCurrent";

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
    if (ctx.workspace.stripeSubscriptionId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: `Customer ${ctx.workspace.stripeCustomerId} already has a subscription.`,
      });
    }

    const customer = await stripe.customers.retrieve(ctx.workspace.stripeCustomerId);
    if (!customer) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Customer ${ctx.workspace.stripeCustomerId} could not be found.`,
      });
    }

    // ctx.workspace was loaded at context-creation time (and may come from a cache),
    // so the guard above can be stale. Re-read right before creating the Stripe
    // subscription so a request racing a just-completed subscribe fails here instead
    // of charging the customer a second time.
    const freshWorkspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, ctx.workspace.id), isNull(table.deletedAtM)),
      columns: { stripeSubscriptionId: true },
    });
    if (!freshWorkspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found.",
      });
    }
    if (freshWorkspace.stripeSubscriptionId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: `Customer ${ctx.workspace.stripeCustomerId} already has a subscription.`,
      });
    }

    /**
     * `error_if_incomplete` makes Stripe reject the create call with a 402 if the first
     * invoice cannot be paid (e.g. card declined). We rely on this to keep the workspace
     * on the Free tier when a first-time signup's payment fails — no DB writes happen
     * unless the subscription is fully active.
     */
    // Deterministic idempotency key so concurrent duplicate calls (double-clicked
    // subscribe button, HTTP retry of the mutation) collapse to a single subscription
    // on Stripe's side instead of billing the customer twice. The 5-minute time
    // bucket keeps the key stable across the realistic race window while still
    // letting a user make a fresh attempt after a card decline — Stripe replays
    // error responses for a reused key for 24h, which would otherwise lock a user
    // out after fixing their payment method.
    const idempotencyKey = `create-subscription:${ctx.workspace.id}:${
      input.productId
    }:${Math.floor(Date.now() / 300_000)}`;

    let sub: Stripe.Subscription;
    try {
      sub = await stripe.subscriptions.create(
        {
          customer: customer.id,
          items: [
            {
              price: product.default_price.toString(),
            },
          ],
          billing_cycle_anchor_config: {
            day_of_month: 1,
          },
          proration_behavior: "always_invoice",
          payment_behavior: "error_if_incomplete",
        },
        { idempotencyKey },
      );
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

    if (sub.status !== "active" && sub.status !== "trialing") {
      // Defensive guard: error_if_incomplete should make this unreachable, but never
      // grant tier access to a subscription that isn't actually paid.
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

    // Only attach the new subscription if the workspace still has none. If a
    // concurrent request won the race between our pre-check and here, an
    // unconditional write would silently overwrite its subscription id, leaving the
    // other subscription active and billing but untracked by the app.
    //
    // "alreadyRecorded" covers the idempotent-replay case: a concurrent request with
    // the same idempotency key received this exact subscription from Stripe and
    // committed it first. That is a success, and crucially we must NOT cancel the
    // subscription — it is the one the workspace is now using.
    const outcome = await db.transaction(
      async (tx): Promise<"created" | "alreadyRecorded" | "lostRace"> => {
        const [updateResult] = await tx
          .update(schema.workspaces)
          .set({
            stripeSubscriptionId: sub.id,
            tier: product.name,
          })
          .where(
            and(
              eq(schema.workspaces.id, ctx.workspace.id),
              isNull(schema.workspaces.stripeSubscriptionId),
            ),
          );

        if (updateResult.affectedRows === 0) {
          const current = await tx.query.workspaces.findFirst({
            where: (table, { eq }) => eq(table.id, ctx.workspace.id),
            columns: { stripeSubscriptionId: true },
          });
          return current?.stripeSubscriptionId === sub.id ? "alreadyRecorded" : "lostRace";
        }

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

        return "created";
      },
    );

    if (outcome === "lostRace") {
      // Another request attached a different subscription first. Cancelling stops
      // future billing, but `error_if_incomplete` means an active subscription has
      // already paid its first invoice — so we must also refund that invoice, or the
      // customer keeps a duplicate paid charge.
      try {
        await stripe.subscriptions.cancel(sub.id);
      } catch (cancelErr) {
        console.error(
          `Failed to cancel duplicate subscription ${sub.id} for workspace ${ctx.workspace.id}. It is still active in Stripe and must be cancelled manually:`,
          cancelErr,
        );
      }
      try {
        const invoiceId =
          typeof sub.latest_invoice === "string" ? sub.latest_invoice : sub.latest_invoice?.id;
        if (invoiceId) {
          const invoice = await stripe.invoices.retrieve(invoiceId);
          const paymentIntentId =
            typeof invoice.payment_intent === "string"
              ? invoice.payment_intent
              : invoice.payment_intent?.id;
          if (invoice.amount_paid > 0 && paymentIntentId) {
            await stripe.refunds.create({ payment_intent: paymentIntentId });
          }
        }
      } catch (refundErr) {
        console.error(
          `Failed to refund the first invoice of duplicate subscription ${sub.id} for workspace ${ctx.workspace.id}. The customer was charged twice and must be refunded manually:`,
          refundErr,
        );
      }
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: `Customer ${ctx.workspace.stripeCustomerId} already has a subscription.`,
      });
    }

    // Invalidate workspace cache after subscription creation
    await invalidateWorkspaceCache(ctx.tenant.id);

    // Also clear the tRPC workspace cache to ensure fresh data on next request
    clearWorkspaceCache(ctx.tenant.id);
  });
