"use client";

import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { ItemContent, ItemGroup, ItemHeader, ItemSeparator, ItemTitle } from "@unkey/ui";
import { ApiPlanRow } from "./api-plan-row";
import { CancelLinks } from "./cancel-links";
import { ComputePlanRow } from "./compute-plan-row";

type BillingInfo = inferRouterOutputs<Router>["stripe"]["getBillingInfo"];

type PlansCardProps = {
  isAdmin: boolean;
  hasPaymentMethod: boolean;
  workspaceSlug: string;
  products: BillingInfo["products"];
  subscription?: BillingInfo["subscription"];
  currentProductId?: BillingInfo["currentProductId"];
  /** Which plan picker to reopen after a checkout round-trip. */
  checkoutIntent: "compute" | "api" | null;
};

/**
 * One plan per product, in one card. A row states what the product is, what it
 * includes, and what the tier costs — the tier sits with the price rather than in
 * the product name, because the tier is what you pay for.
 */
export function PlansCard({
  isAdmin,
  hasPaymentMethod,
  workspaceSlug,
  products,
  subscription,
  currentProductId,
  checkoutIntent,
}: PlansCardProps) {
  const { data: deploySubscription } = trpc.stripe.getDeploySubscription.useQuery(undefined, {
    staleTime: 30_000,
  });

  const hasPaidApiPlan = Boolean(
    subscription &&
      currentProductId &&
      ["active", "trialing", "past_due"].includes(subscription.status),
  );
  // Nothing subscribed at all: the only state where a plan row is the page's
  // primary action, since without Compute the workspace cannot deploy.
  const emphasize = !deploySubscription?.plan && !hasPaidApiPlan;

  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemContent>
          <ItemTitle>Plans</ItemTitle>
        </ItemContent>
      </ItemHeader>

      <ItemSeparator />
      <ComputePlanRow
        isAdmin={isAdmin}
        hasPaymentMethod={hasPaymentMethod}
        workspaceSlug={workspaceSlug}
        emphasize={emphasize}
        autoOpenPlanModal={checkoutIntent === "compute" && hasPaymentMethod}
      />

      <ItemSeparator />
      <ApiPlanRow
        isAdmin={isAdmin}
        hasPaymentMethod={hasPaymentMethod}
        workspaceSlug={workspaceSlug}
        emphasize={emphasize}
        products={products}
        subscription={subscription}
        currentProductId={currentProductId}
        autoOpenPlanModal={checkoutIntent === "api" && hasPaymentMethod}
      />

      <CancelLinks
        isAdmin={isAdmin}
        canCancelApi={Boolean(
          subscription && subscription.status === "active" && !subscription.cancelAt,
        )}
      />
    </ItemGroup>
  );
}
