import { logOperation } from "@/lib/logging";
import { getStripeClient } from "@/lib/stripe";
import {
  SUBSCRIPTION_PRODUCTS,
  type SubscriptionProduct,
  subscriptionIdFor,
} from "@/lib/stripe/billingSubscriptions";
import { retrieveSubscriptionOrNull } from "@/lib/stripe/notFound";
import { needsPaymentAction } from "@/lib/stripe/subscriptionUtils";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

export const getPaymentIssues = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }): Promise<SubscriptionProduct[]> => {
    const stripe = getStripeClient();

    const failing = await Promise.all(
      SUBSCRIPTION_PRODUCTS.map(async (product) => {
        const subscriptionId = subscriptionIdFor(ctx.workspace, product);
        if (!subscriptionId) {
          return null;
        }
        try {
          const subscription = await retrieveSubscriptionOrNull(stripe, subscriptionId);
          return subscription && needsPaymentAction(subscription.status) ? product : null;
        } catch (error) {
          logOperation("error", "Payment issue check failed", {
            workspace_id: ctx.workspace.id,
            product,
            error: error instanceof Error ? error.message : String(error),
          });
          return null;
        }
      }),
    );
    return failing.filter((product): product is SubscriptionProduct => product !== null);
  });
