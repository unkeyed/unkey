import { getStripeClient } from "@/lib/stripe";
import { subscriptionIdFor, subscriptionProductSchema } from "@/lib/stripe/billingSubscriptions";
import { retrieveSubscriptionOrNull } from "@/lib/stripe/notFound";
import { hostedInvoiceUrl, needsPaymentAction } from "@/lib/stripe/subscriptionUtils";
import { TRPCError } from "@trpc/server";
import type Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

export async function resolveSubscriptionPaymentUrl(
  stripe: Stripe,
  subscriptionId: string,
  customerId: string,
): Promise<string> {
  const subscription = await retrieveSubscriptionOrNull(stripe, subscriptionId, {
    expand: ["latest_invoice"],
  });
  if (!subscription) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "The pending subscription payment no longer exists.",
    });
  }

  const subscriptionCustomerId =
    typeof subscription.customer === "string" ? subscription.customer : subscription.customer?.id;
  if (subscriptionCustomerId !== customerId) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "The pending subscription payment could not be found.",
    });
  }
  if (!needsPaymentAction(subscription.status)) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "This subscription does not have a payment requiring action.",
    });
  }

  const paymentUrl = hostedInvoiceUrl(subscription);
  if (!paymentUrl) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "Stripe does not have an open invoice for this subscription.",
    });
  }

  return paymentUrl;
}

export const getSubscriptionPaymentUrl = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(z.object({ product: subscriptionProductSchema }))
  .mutation(async ({ ctx, input }) => {
    const subscriptionId = subscriptionIdFor(ctx.workspace, input.product);
    if (!subscriptionId || !ctx.workspace.stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "No pending subscription payment was found.",
      });
    }

    const paymentUrl = await resolveSubscriptionPaymentUrl(
      getStripeClient(),
      subscriptionId,
      ctx.workspace.stripeCustomerId,
    );
    return { paymentUrl };
  });
