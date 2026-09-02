import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { getStripeClient } from "@/lib/stripe";
import { hostedInvoiceUrl } from "@/lib/stripe/subscriptionUtils";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

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
  if (
    !(["incomplete", "past_due", "unpaid"] as Stripe.Subscription.Status[]).includes(
      subscription.status,
    )
  ) {
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

/** Returns the admin-only bearer URL for the API subscription's open invoice. */
export const getSubscriptionPaymentUrl = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .mutation(async ({ ctx }) => {
    if (!ctx.workspace.stripeSubscriptionId || !ctx.workspace.stripeCustomerId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "No pending subscription payment was found.",
      });
    }

    const paymentUrl = await resolveSubscriptionPaymentUrl(
      getStripeClient(),
      ctx.workspace.stripeSubscriptionId,
      ctx.workspace.stripeCustomerId,
    );
    return { paymentUrl };
  });
