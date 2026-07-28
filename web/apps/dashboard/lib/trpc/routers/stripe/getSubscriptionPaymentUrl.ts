import { getStripeClient } from "@/lib/stripe";
import { hostedInvoiceUrl } from "@/lib/stripe/subscriptionUtils";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

const recoverableStatuses = new Set<Stripe.Subscription.Status>([
  "incomplete",
  "past_due",
  "unpaid",
]);

export async function resolveSubscriptionPaymentUrl(
  stripe: Stripe,
  subscriptionId: string,
  customerId: string,
): Promise<string> {
  let subscription: Stripe.Subscription;
  try {
    subscription = await stripe.subscriptions.retrieve(subscriptionId, {
      expand: ["latest_invoice"],
    });
  } catch (error) {
    if (error instanceof Stripe.errors.StripeError && error.code === "resource_missing") {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "The pending subscription payment no longer exists.",
      });
    }
    throw error;
  }

  const subscriptionCustomerId =
    typeof subscription.customer === "string" ? subscription.customer : subscription.customer?.id;
  if (subscriptionCustomerId !== customerId) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "The pending subscription payment could not be found.",
    });
  }
  if (!recoverableStatuses.has(subscription.status)) {
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

/**
 * Returns the bearer URL for the recorded subscription's open invoice. This is
 * intentionally admin-gated and separate from member-visible billing queries.
 */
export const getSubscriptionPaymentUrl = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .input(z.object({ product: z.enum(["api", "compute"]) }))
  .output(z.object({ paymentUrl: z.string().url() }))
  .mutation(async ({ ctx, input }) => {
    const subscriptionId =
      input.product === "api"
        ? ctx.workspace.stripeSubscriptionId
        : ctx.workspace.stripeDeploySubscriptionId;
    if (!subscriptionId || !ctx.workspace.stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "No pending subscription payment was found.",
      });
    }

    const stripe = getStripeClient();
    const paymentUrl = await resolveSubscriptionPaymentUrl(
      stripe,
      subscriptionId,
      ctx.workspace.stripeCustomerId,
    );

    return { paymentUrl };
  });
