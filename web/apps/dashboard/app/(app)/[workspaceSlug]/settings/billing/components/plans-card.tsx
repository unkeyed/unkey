"use client";

import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterOutputs } from "@trpc/server";
import { ItemContent, ItemGroup, ItemHeader, ItemSeparator, ItemTitle } from "@unkey/ui";
import { currentApiProduct } from "./api-plan";
import { ApiPlanRow } from "./api-plan-row";
import { ComputePlanRow } from "./compute-plan-row";

type BillingInfo = inferRouterOutputs<Router>["stripe"]["getBillingInfo"];

type PlansCardProps = {
  isAdmin: boolean | undefined;
  hasPaymentMethod: boolean;
  workspaceSlug: string;
  products: BillingInfo["products"];
  subscription?: BillingInfo["subscription"];
  currentProductId?: BillingInfo["currentProductId"];
  checkoutIntent: "compute" | "api" | null;
};

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

  const hasPaidApiPlan = Boolean(currentApiProduct({ products, subscription, currentProductId }));
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
    </ItemGroup>
  );
}
