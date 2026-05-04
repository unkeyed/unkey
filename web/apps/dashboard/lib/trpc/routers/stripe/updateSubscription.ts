import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import { validateAndParseQuotas } from "@/lib/stripe/productUtils";
import { invalidateWorkspaceCache } from "@/lib/workspace-cache";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

export const updateSubscription = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(
    z.object({
      newProductId: z.string(),
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

    // Reject any product not on the configured allow-list. Without this,
    // an admin can switch the workspace to any product in the connected
    // Stripe account — including test/internal products with $0 prices or
    // permissive quota metadata.
    const allowedProductIds = new Set<string>([
      ...e.STRIPE_PRODUCT_IDS_PRO,
      ...e.STRIPE_PRODUCT_IDS_ENTERPRISE,
    ]);
    if (!allowedProductIds.has(input.newProductId)) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find product ${input.newProductId}.`,
      });
    }

    if (!ctx.workspace.stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace doesn't have a stripe customer id.",
      });
    }
    if (!ctx.workspace.stripeSubscriptionId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Workspace doesn't have a stripe subscrption id.",
      });
    }

    const newProduct = await stripe.products.retrieve(input.newProductId);

    if (!newProduct) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find product ${input.newProductId}.`,
      });
    }

    const newQuotas = validateAndParseQuotas(newProduct);
    if (
      !newQuotas.valid ||
      newQuotas.requestsPerMonth === undefined ||
      newQuotas.logsRetentionDays === undefined ||
      newQuotas.auditLogsRetentionDays === undefined
    ) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: `Product ${newProduct.id} is missing required quota metadata.`,
      });
    }

    const sub = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);

    if (!sub) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Could not find subscription ${ctx.workspace.stripeSubscriptionId}.`,
      });
    }

    // Derive the current subscription item from the existing subscription
    // rather than trusting a client-supplied `oldProductId`. The client should
    // not be able to influence which item gets repriced.
    const item = sub.items.data[0];

    if (!item) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Subscription has no items to update.",
      });
    }

    if (!item.id) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Subscription item is missing an ID.",
      });
    }

    if (!newProduct.default_price) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: `Product ${newProduct.id} is missing a default price.`,
      });
    }

    /**
     * `error_if_incomplete` rejects the call with a 402 if the proration invoice cannot
     * be charged. We surface that to the user so they know to fix their payment method
     * before the plan switch is applied.
     */
    try {
      await stripe.subscriptionItems.update(item.id, {
        price: newProduct.default_price.toString(),
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

    if (sub.cancel_at) {
      await stripe.subscriptions.update(sub.id, {
        cancel_at_period_end: false,
      });
    }

    await db.transaction(async (tx) => {
      await tx
        .update(schema.workspaces)
        .set({
          tier: newProduct.name,
        })
        .where(eq(schema.workspaces.id, ctx.workspace.id));

      await tx
        .insert(schema.quotas)
        .values({
          workspaceId: ctx.workspace.id,
          requestsPerMonth: newQuotas.requestsPerMonth,
          logsRetentionDays: newQuotas.logsRetentionDays,
          auditLogsRetentionDays: newQuotas.auditLogsRetentionDays,
          team: true,
        })
        .onDuplicateKeyUpdate({
          set: {
            requestsPerMonth: newQuotas.requestsPerMonth,
            logsRetentionDays: newQuotas.logsRetentionDays,
            auditLogsRetentionDays: newQuotas.auditLogsRetentionDays,
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
        description: `Switched to ${newProduct.name} plan.`,
        resources: [],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });

    // Invalidate workspace cache after subscription update
    await invalidateWorkspaceCache(ctx.tenant.id);
  });
