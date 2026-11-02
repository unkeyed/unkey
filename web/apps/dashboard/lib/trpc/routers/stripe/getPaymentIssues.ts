import { logOperation } from "@/lib/logging";
import {
  SUBSCRIPTION_PRODUCTS,
  type SubscriptionProduct,
  subscriptionIdFor,
} from "@/lib/stripe/billingSubscriptions";
import { cachedSubscriptionStatus } from "@/lib/stripe/subscriptionStatusCache";
import { needsPaymentAction } from "@/lib/stripe/subscriptionUtils";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";

export const getPaymentIssues = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx }): Promise<SubscriptionProduct[]> => {
    const customerId = ctx.workspace.stripeCustomerId;
    if (!customerId) {
      return [];
    }

    const failing = await Promise.all(
      SUBSCRIPTION_PRODUCTS.map(async (product) => {
        const subscriptionId = subscriptionIdFor(ctx.workspace, product);
        if (!subscriptionId) {
          return null;
        }
        try {
          const status = await cachedSubscriptionStatus(customerId, subscriptionId);
          return status && needsPaymentAction(status) ? product : null;
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
