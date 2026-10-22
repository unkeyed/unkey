import { auth } from "@/lib/auth/server";
import {
  CANCELLATION_COMMENT_MAX_LENGTH,
  CANCELLATION_REASONS,
  type CancellationReason,
} from "@/lib/billing/cancellation-reasons";
import { BILLING_PRODUCTS, type BillingProduct } from "@/lib/billing/products";
import type { loadWorkspace } from "@/lib/db/load-workspace";
import { getStripeClient } from "@/lib/stripe";
import { isDeadSubscription } from "@/lib/stripe/subscriptionUtils";
import type { Context } from "@/lib/trpc/context";
import { requireWorkspaceAdmin, withCreateRateLimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { type CancellationFeedback, alertCancellationFeedback } from "@/lib/utils/slackAlerts";
import { TRPCError } from "@trpc/server";
import type Stripe from "stripe";
import { z } from "zod";

const SEND_FAILED_MESSAGE = "Could not send your feedback. Please email support@unkey.com instead.";

type Workspace = NonNullable<Awaited<ReturnType<typeof loadWorkspace>>>;

type StripeCustomerRef = NonNullable<CancellationFeedback["stripe"]>;

type FeedbackTarget =
  | { kind: "stripe"; subscriptionId: string; stripe: StripeCustomerRef }
  | { kind: "slack-only"; stripe?: StripeCustomerRef };

export const sendCancellationFeedback = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .use(withCreateRateLimit())
  .input(
    z.object({
      product: z.enum(BILLING_PRODUCTS),
      reason: z.enum(CANCELLATION_REASONS),
      comment: z.string().trim().max(CANCELLATION_COMMENT_MAX_LENGTH),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const user = await lookUpUser(ctx.user);
    const target = await resolveFeedbackTarget(input.product, ctx.workspace);
    if (target.kind === "stripe") {
      await recordInStripe(target.subscriptionId, input.reason, input.comment);
    }

    const slackStatus = await alertCancellationFeedback({
      product: input.product,
      reason: input.reason,
      comment: input.comment || undefined,
      ...user,
      workspaceId: ctx.workspace.id,
      workspaceName: ctx.workspace.name,
      stripe: target.stripe,
    });
    if (target.kind === "slack-only" && slackStatus !== "sent") {
      throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: SEND_FAILED_MESSAGE });
    }
  });

async function lookUpUser(
  user: NonNullable<Context["user"]>,
): Promise<Pick<CancellationFeedback, "name" | "email">> {
  const unknown = { name: "Unknown", email: "Unknown" };
  try {
    const profile = user.profile ?? (await auth.getUser(user.id));
    return profile ? { name: profile.fullName ?? "Unknown", email: profile.email } : unknown;
  } catch (error) {
    console.error("Failed to look up the user for cancellation feedback:", error);
    return unknown;
  }
}

async function resolveFeedbackTarget(
  product: BillingProduct,
  workspace: Workspace,
): Promise<FeedbackTarget> {
  const subscriptionId =
    product === "api" ? workspace.stripeSubscriptionId : workspace.stripeDeploySubscriptionId;

  if (!subscriptionId) {
    if (product === "api" || workspace.deployPlan !== null) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "This workspace has no cancelled subscription to attach feedback to.",
      });
    }
    return { kind: "slack-only", stripe: await lookUpCustomer(workspace.stripeCustomerId) };
  }

  let sub: Stripe.Subscription;
  try {
    sub = await getStripeClient().subscriptions.retrieve(subscriptionId);
  } catch (error) {
    console.error("Failed to retrieve subscription for cancellation feedback:", error);
    throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: SEND_FAILED_MESSAGE });
  }

  const stripe: StripeCustomerRef = {
    customerId: typeof sub.customer === "string" ? sub.customer : sub.customer.id,
    livemode: sub.livemode,
  };
  if (isDeadSubscription(sub)) {
    return { kind: "slack-only", stripe };
  }
  if (!sub.cancel_at_period_end && sub.cancel_at === null) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "This subscription is not cancelled.",
    });
  }
  return { kind: "stripe", subscriptionId, stripe };
}

async function lookUpCustomer(customerId: string | null): Promise<StripeCustomerRef | undefined> {
  if (!customerId) {
    return undefined;
  }
  try {
    const customer = await getStripeClient().customers.retrieve(customerId);
    return customer.deleted ? undefined : { customerId, livemode: customer.livemode };
  } catch (error) {
    console.error("Failed to retrieve customer for cancellation feedback:", error);
    return undefined;
  }
}

async function recordInStripe(
  subscriptionId: string,
  reason: CancellationReason,
  comment: string,
): Promise<void> {
  try {
    await getStripeClient().subscriptions.update(subscriptionId, {
      cancellation_details: comment ? { feedback: reason, comment } : { feedback: reason },
    });
  } catch (error) {
    console.error("Failed to store cancellation feedback in Stripe:", error);
    throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: SEND_FAILED_MESSAGE });
  }
}
