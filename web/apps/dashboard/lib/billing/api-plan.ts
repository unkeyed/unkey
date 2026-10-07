import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";

type BillingInfo = inferRouterOutputs<Router>["stripe"]["getBillingInfo"];

const BILLING_STATUSES = ["active", "trialing", "past_due"];

export function currentApiProduct({
  products,
  subscription,
  currentProductId,
}: {
  products: BillingInfo["products"];
  subscription?: BillingInfo["subscription"];
  currentProductId?: BillingInfo["currentProductId"];
}): BillingInfo["products"][number] | undefined {
  if (!subscription || !currentProductId || !BILLING_STATUSES.includes(subscription.status)) {
    return undefined;
  }
  return products.find((product) => product.id === currentProductId);
}
