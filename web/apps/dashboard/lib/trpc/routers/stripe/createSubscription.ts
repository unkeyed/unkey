import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import { createSubscriptionCheckout } from "@/lib/stripe/createSubscriptionCheckout";
import { validateAndParseQuotas } from "@/lib/stripe/productUtils";
import { isDeadSubscription } from "@/lib/stripe/subscriptionUtils";
import { getBaseUrl } from "@/lib/utils";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

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
    const defaultPriceId =
      typeof product.default_price === "string" ? product.default_price : product.default_price.id;

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

    // The API product owns its own subscription now, so a live recorded API
    // subscription means the workspace already has an API plan. A corpse
    // (cancelled mid-month, deleted-webhook that clears the column lagging) or a
    // subscription gone from Stripe counts as absent, so a mid-month cancel can
    // resubscribe. resource_missing is the same "dead counts as absent" case,
    // not a 500; anything else propagates so a transient failure never silently
    // downgrades a live subscription to "absent".
    if (ctx.workspace.stripeSubscriptionId) {
      const recorded = await stripe.subscriptions
        .retrieve(ctx.workspace.stripeSubscriptionId)
        .catch((err: unknown) => {
          if (err instanceof Stripe.errors.StripeError && err.code === "resource_missing") {
            return null;
          }
          throw err;
        });
      if (recorded && !isDeadSubscription(recorded)) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: `Customer ${ctx.workspace.stripeCustomerId} already has an API plan.`,
        });
      }
    }

    const customer = await stripe.customers.retrieve(ctx.workspace.stripeCustomerId);
    if (!customer) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Customer ${ctx.workspace.stripeCustomerId} could not be found.`,
      });
    }

    // Setup-mode Checkout only vaults a card; it does not prove that the first
    // invoice can be paid. Collect the initial subscription payment in Checkout
    // too, so Stripe can render CVC recollection and 3DS instead of returning a
    // requires_action error to this server-only mutation. The paid subscription
    // is linked and grants its quotas by /success and the completed webhook.
    const successUrl = `${getBaseUrl()}/success?session_id={CHECKOUT_SESSION_ID}&intent=api-subscription`;
    let checkoutUrl: string | null;
    try {
      const destination = await createSubscriptionCheckout(stripe, {
        workspaceId: ctx.workspace.id,
        product: "api",
        customerId: customer.id,
        lineItems: [{ price: defaultPriceId, quantity: 1 }],
        successUrl,
        idempotencyKey: `api-checkout:${ctx.workspace.id}:${input.productId}`,
      });
      checkoutUrl = destination.kind === "success" ? destination.url : destination.session.url;
    } catch (err) {
      if (err instanceof Stripe.errors.StripeError) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Payment could not be started. Please try again or update your payment method.",
        });
      }
      throw err;
    }

    if (!checkoutUrl) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Stripe did not return a payment URL. Please try again.",
      });
    }

    return { checkoutUrl };
  });
