import {
  CANCELLATION_COMMENT_MAX_LENGTH,
  CANCELLATION_REASONS,
} from "@/lib/billing/cancellation-reasons";
import { BILLING_PRODUCTS } from "@/lib/billing/products";
import { getStripeClient } from "@/lib/stripe";
import { isCancelling, isDeadSubscription } from "@/lib/stripe/subscriptionUtils";
import { requireWorkspaceAdmin, withUpdateRateLimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const SEND_FAILED_MESSAGE = "Could not send your feedback. Please email support@unkey.com instead.";

export const sendCancellationFeedback = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .use(withUpdateRateLimit())
  .input(
    z.object({
      product: z.enum(BILLING_PRODUCTS),
      reason: z.enum(CANCELLATION_REASONS),
      comment: z.string().trim().max(CANCELLATION_COMMENT_MAX_LENGTH),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const subscriptionId =
      input.product === "api"
        ? ctx.workspace.stripeSubscriptionId
        : ctx.workspace.stripeDeploySubscriptionId;
    if (!subscriptionId) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "This workspace has no subscription to attach feedback to.",
      });
    }

    const stripe = getStripeClient();
    try {
      const sub = await stripe.subscriptions.retrieve(subscriptionId);
      if (!isDeadSubscription(sub) && isCancelling(sub)) {
        await stripe.subscriptions.update(subscriptionId, {
          cancellation_details: { feedback: input.reason, comment: input.comment },
        });
        return;
      }
    } catch (error) {
      console.error("Failed to store cancellation feedback in Stripe:", error);
      throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: SEND_FAILED_MESSAGE });
    }
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "This subscription is not cancelling.",
    });
  });
