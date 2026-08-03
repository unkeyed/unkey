import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import { changeSubscriptionPrice } from "@/lib/stripe/changeSubscriptionPrice";
import { deployBillingConfig, findApiItem } from "@/lib/stripe/deployBilling";
import { parseDeployPlan } from "@/lib/stripe/deployPlan";
import { validateAndParseQuotas } from "@/lib/stripe/productUtils";
import { setComputeQuotas } from "@/lib/stripe/setComputeQuotas";
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

    // Reprice the API plan item specifically. findApiItem skips any Deploy price
    // (metered or plan-fee) the subscription might also carry, so we never
    // reprice a Deploy line and charge a bad proration. Derived from the
    // subscription, not a client-supplied product id, so the client cannot
    // influence which item gets repriced.
    const config = await deployBillingConfig();
    const item = findApiItem(config, sub.items.data);

    if (!item) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Subscription has no API plan item to update.",
      });
    }

    if (!newProduct.default_price) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: `Product ${newProduct.id} is missing a default price.`,
      });
    }
    const newPriceId =
      typeof newProduct.default_price === "string"
        ? newProduct.default_price
        : newProduct.default_price.id;
    const newPrice =
      typeof newProduct.default_price === "string"
        ? await stripe.prices.retrieve(newPriceId)
        : newProduct.default_price;
    const upgraded =
      item.price.unit_amount !== null &&
      newPrice.unit_amount !== null &&
      newPrice.unit_amount > item.price.unit_amount;

    if (sub.cancel_at_period_end) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Resume your API plan before changing it.",
      });
    }

    let result: Awaited<ReturnType<typeof changeSubscriptionPrice>>;
    try {
      result = await changeSubscriptionPrice(stripe, {
        subscriptionId: sub.id,
        subscriptionItemId: item.id,
        newPriceId,
        prorationBehavior: "always_invoice",
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

    if (result.kind === "payment_required") {
      return result;
    }

    // Workspace API rate limits are manually applied safety limits, not plan
    // quotas. Clear them when a customer pays for a higher tier, but preserve
    // deliberate limits on same-price changes and downgrades.
    const rateLimitReset = upgraded ? { ratelimitApiLimit: null, ratelimitApiDuration: null } : {};

    await db.transaction(async (tx) => {
      await tx
        .update(schema.workspaceBilling)
        .set({
          tier: newProduct.name,
        })
        .where(eq(schema.workspaceBilling.workspaceId, ctx.workspace.id));

      await setComputeQuotas(tx, {
        workspaceId: ctx.workspace.id,
        plan:
          parseDeployPlan(ctx.workspace.deployPlanOverride) ??
          parseDeployPlan(ctx.workspace.deployPlan),
        preserveApiQuotas: true,
        quotaUpdate: {
          requestsPerMonth: newQuotas.requestsPerMonth,
          logsRetentionDays: newQuotas.logsRetentionDays,
          auditLogsRetentionDays: newQuotas.auditLogsRetentionDays,
          team: true,
          ...rateLimitReset,
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

    return result;
  });
