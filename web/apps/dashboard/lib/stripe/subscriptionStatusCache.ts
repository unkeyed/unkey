import { getStripeClient } from "@/lib/stripe";
import { retrieveSubscriptionOrNull } from "@/lib/stripe/notFound";
import { revalidateTag, unstable_cache } from "next/cache";
import type Stripe from "stripe";

const CACHE_TTL_SECONDS = 5 * 60;

function customerTag(customerId: string): string {
  return `stripe-customer:${customerId}`;
}

export function cachedSubscriptionStatus(
  customerId: string,
  subscriptionId: string,
): Promise<Stripe.Subscription.Status | null> {
  return unstable_cache(
    async () => {
      const subscription = await retrieveSubscriptionOrNull(getStripeClient(), subscriptionId);
      return subscription?.status ?? null;
    },
    ["billing", "subscription-status", subscriptionId],
    { revalidate: CACHE_TTL_SECONDS, tags: [customerTag(customerId)] },
  )();
}

export function expireSubscriptionStatus(customerId: string): void {
  revalidateTag(customerTag(customerId), { expire: 0 });
}

export function subscriptionStatusCustomer(event: Stripe.Event): string | null {
  switch (event.type) {
    case "customer.subscription.created":
    case "customer.subscription.updated":
    case "customer.subscription.deleted":
    case "invoice.paid":
    case "invoice.payment_failed":
    case "invoice.payment_succeeded": {
      const { customer } = event.data.object;
      if (!customer) {
        return null;
      }
      return typeof customer === "string" ? customer : customer.id;
    }
    default:
      return null;
  }
}
